package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
)

type QueueWorker struct {
	queueRepo repository.SendQueueRepository
	msgStore  *storage.MessageStore
	hostname  string
	logger    *slog.Logger
	stop      chan struct{}
}

func NewQueueWorker(
	queueRepo repository.SendQueueRepository,
	msgStore *storage.MessageStore,
	hostname string,
	logger *slog.Logger,
) *QueueWorker {
	return &QueueWorker{
		queueRepo: queueRepo,
		msgStore:  msgStore,
		hostname:  hostname,
		logger:    logger,
		stop:      make(chan struct{}),
	}
}

func (w *QueueWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *QueueWorker) Stop() {
	close(w.stop)
}

func (w *QueueWorker) run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case <-ticker.C:
			w.processQueue(ctx)
		}
	}
}

func (w *QueueWorker) processQueue(ctx context.Context) {
	items, err := w.queueRepo.Dequeue(ctx, 10)
	if err != nil {
		w.logger.Error("failed to dequeue messages", "error", err)
		return
	}

	for _, item := range items {
		if err := w.deliver(ctx, item); err != nil {
			w.logger.Error("delivery failed", "id", item.ID, "error", err)
			w.queueRepo.UpdateStatus(ctx, item.ID, "deferred", err.Error())
			w.queueRepo.IncrementAttempts(ctx, item.ID)
		} else {
			w.queueRepo.UpdateStatus(ctx, item.ID, "sent", "")
			w.logger.Info("message delivered", "id", item.ID, "to", item.ToAddresses)
		}
	}
}

func (w *QueueWorker) deliver(ctx context.Context, item *model.SendQueueItem) error {
	rawMsg, err := w.msgStore.Get(ctx, item.StorageKey)
	if err != nil {
		return fmt.Errorf("get message from storage: %w", err)
	}

	// Group recipients by domain
	domainRecipients := make(map[string][]string)
	for _, to := range item.ToAddresses {
		parts := strings.SplitN(to, "@", 2)
		if len(parts) == 2 {
			domainRecipients[parts[1]] = append(domainRecipients[parts[1]], to)
		}
	}

	for domain, recipients := range domainRecipients {
		if err := w.deliverToDomain(domain, item.FromAddress, recipients, rawMsg); err != nil {
			return fmt.Errorf("deliver to %s: %w", domain, err)
		}
	}

	return nil
}

func (w *QueueWorker) deliverToDomain(domain, from string, to []string, msg []byte) error {
	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		return fmt.Errorf("no MX records for %s", domain)
	}

	var lastErr error
	for _, mx := range mxRecords {
		host := strings.TrimSuffix(mx.Host, ".")
		addr := host + ":25"

		err := w.sendToHost(addr, host, from, to, msg)
		if err == nil {
			return nil
		}
		lastErr = err
		w.logger.Warn("MX delivery attempt failed", "host", host, "error", err)
	}

	return fmt.Errorf("all MX hosts failed: %w", lastErr)
}

func (w *QueueWorker) sendToHost(addr, host, from string, to []string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Close()

	// Try STARTTLS
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: host}
		if err := client.StartTLS(tlsConfig); err != nil {
			w.logger.Warn("STARTTLS failed, continuing without TLS", "host", host)
		}
	}

	if err := client.Hello(w.hostname); err != nil {
		return err
	}

	if err := client.Mail(from); err != nil {
		return err
	}

	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := wc.Write(msg); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return client.Quit()
}
