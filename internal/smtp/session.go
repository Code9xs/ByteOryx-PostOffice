package smtp

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/emersion/go-smtp"
	"github.com/google/uuid"
)

type Session struct {
	backend *Backend
	conn    *smtp.Conn
	logger  *slog.Logger
	from    string
	to      []string
	authed  bool
	userID  *uuid.UUID
}

func (s *Session) AuthPlain(username, password string) error {
	ctx := context.Background()
	user, err := s.backend.userRepo.GetByEmail(ctx, username)
	if err != nil {
		return fmt.Errorf("invalid credentials")
	}
	_ = user
	s.authed = true
	s.userID = &user.ID
	s.logger.Info("SMTP auth success", "user", username)
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	if s.backend.requireAuth && !s.authed {
		return fmt.Errorf("authentication required")
	}
	s.from = from
	s.logger.Debug("MAIL FROM", "from", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	if !s.backend.requireAuth {
		// Inbound: verify recipient exists
		ctx := context.Background()
		parts := strings.SplitN(to, "@", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid recipient address")
		}

		_, err := s.backend.mailboxRepo.GetByAddress(ctx, to)
		if err != nil {
			// Try catch-all
			domain, dErr := getDomainFromAddress(to)
			if dErr != nil {
				return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 1, 1}, Message: "user not found"}
			}
			_ = domain
			return &smtp.SMTPError{Code: 550, EnhancedCode: smtp.EnhancedCode{5, 1, 1}, Message: "user not found"}
		}
	}

	s.to = append(s.to, to)
	s.logger.Debug("RCPT TO", "to", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(r, 26*1024*1024))
	if err != nil {
		return err
	}

	ctx := context.Background()

	if s.backend.requireAuth {
		return s.handleOutbound(ctx, data)
	}
	return s.handleInbound(ctx, data)
}

func (s *Session) handleInbound(ctx context.Context, data []byte) error {
	for _, rcpt := range s.to {
		mailbox, err := s.backend.mailboxRepo.GetByAddress(ctx, rcpt)
		if err != nil {
			continue
		}

		storageKey, err := s.backend.msgStore.Store(ctx, mailbox.ID, data)
		if err != nil {
			s.logger.Error("failed to store message", "error", err)
			continue
		}

		folder, err := s.backend.folderRepo.GetByName(ctx, mailbox.ID, "INBOX")
		if err != nil {
			s.logger.Error("INBOX not found", "mailbox", rcpt)
			continue
		}

		uid, err := s.backend.folderRepo.IncrementUIDNext(ctx, folder.ID)
		if err != nil {
			s.logger.Error("failed to get next UID", "error", err)
			continue
		}

		subject, messageID := parseBasicHeaders(data)

		msg := &model.Message{
			ID:          uuid.New(),
			FolderID:    folder.ID,
			MailboxID:   mailbox.ID,
			UID:         uid,
			MessageID:   messageID,
			Subject:     subject,
			FromAddress: s.from,
			ToAddresses: s.to,
			Date:        time.Now(),
			SizeBytes:   len(data),
			StorageKey:  storageKey,
		}

		if err := s.backend.messageRepo.Create(ctx, msg); err != nil {
			s.logger.Error("failed to save message metadata", "error", err)
			continue
		}

		s.logger.Info("message delivered", "to", rcpt, "from", s.from, "size", len(data))
	}
	return nil
}

func (s *Session) handleOutbound(ctx context.Context, data []byte) error {
	// Store message and enqueue for delivery
	mailboxID := uuid.Nil
	if s.userID != nil {
		mailboxes, _ := s.backend.mailboxRepo.ListByUser(ctx, *s.userID)
		for _, mb := range mailboxes {
			if mb.Address == s.from {
				mailboxID = mb.ID
				break
			}
		}
	}

	storageKey, err := s.backend.msgStore.Store(ctx, mailboxID, data)
	if err != nil {
		return fmt.Errorf("failed to store outbound message: %w", err)
	}

	item := &model.SendQueueItem{
		ID:          uuid.New(),
		FromAddress: s.from,
		ToAddresses: s.to,
		StorageKey:  storageKey,
		Status:      "pending",
		MaxAttempts: 5,
	}

	if err := s.backend.queueRepo.Enqueue(ctx, item); err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	s.logger.Info("message queued for delivery", "from", s.from, "to", s.to)
	return nil
}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}

func getDomainFromAddress(addr string) (string, error) {
	parts := strings.SplitN(addr, "@", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid address")
	}
	return parts[1], nil
}

func parseBasicHeaders(data []byte) (subject, messageID string) {
	lines := strings.Split(string(data), "\r\n")
	for _, line := range lines {
		if line == "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "subject: ") {
			subject = strings.TrimPrefix(line, "Subject: ")
			subject = strings.TrimPrefix(subject, "subject: ")
		}
		if strings.HasPrefix(lower, "message-id: ") {
			messageID = strings.TrimPrefix(line, "Message-ID: ")
			messageID = strings.TrimPrefix(messageID, "message-id: ")
		}
	}
	return
}
