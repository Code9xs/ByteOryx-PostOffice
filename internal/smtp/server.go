package smtp

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/byteoryx/postoffice/internal/config"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
	"github.com/emersion/go-smtp"
)

type Server struct {
	inbound    *smtp.Server
	submission *smtp.Server
	cfg        *config.SMTPConfig
	logger     *slog.Logger
}

func NewServer(
	cfg *config.SMTPConfig,
	hostname string,
	mailboxRepo repository.MailboxRepository,
	messageRepo repository.MessageRepository,
	folderRepo repository.FolderRepository,
	queueRepo repository.SendQueueRepository,
	userRepo repository.UserRepository,
	msgStore *storage.MessageStore,
	tlsConfig *tls.Config,
	logger *slog.Logger,
) *Server {
	backend := &Backend{
		mailboxRepo: mailboxRepo,
		messageRepo: messageRepo,
		folderRepo:  folderRepo,
		queueRepo:   queueRepo,
		userRepo:    userRepo,
		msgStore:    msgStore,
		hostname:    hostname,
		logger:      logger,
	}

	inbound := smtp.NewServer(backend)
	inbound.Addr = cfg.ListenAddr
	inbound.Domain = hostname
	inbound.MaxMessageBytes = cfg.MaxMessageSize
	inbound.MaxRecipients = cfg.MaxRecipients
	inbound.ReadTimeout = 60 * time.Second
	inbound.WriteTimeout = 60 * time.Second
	inbound.AllowInsecureAuth = true
	if tlsConfig != nil {
		inbound.TLSConfig = tlsConfig
	}

	submissionBackend := &Backend{
		mailboxRepo:    mailboxRepo,
		messageRepo:    messageRepo,
		folderRepo:     folderRepo,
		queueRepo:      queueRepo,
		userRepo:       userRepo,
		msgStore:       msgStore,
		hostname:       hostname,
		logger:         logger,
		requireAuth:    true,
	}

	submission := smtp.NewServer(submissionBackend)
	submission.Addr = cfg.SubmissionAddr
	submission.Domain = hostname
	submission.MaxMessageBytes = cfg.MaxMessageSize
	submission.MaxRecipients = cfg.MaxRecipients
	submission.ReadTimeout = 60 * time.Second
	submission.WriteTimeout = 60 * time.Second
	submission.AllowInsecureAuth = false
	if tlsConfig != nil {
		submission.TLSConfig = tlsConfig
	}

	return &Server{
		inbound:    inbound,
		submission: submission,
		cfg:        cfg,
		logger:     logger,
	}
}

func (s *Server) Start(ctx context.Context) error {
	go func() {
		s.logger.Info("starting SMTP inbound server", "addr", s.cfg.ListenAddr)
		if err := s.inbound.ListenAndServe(); err != nil {
			s.logger.Error("SMTP inbound server error", "error", err)
		}
	}()

	go func() {
		s.logger.Info("starting SMTP submission server", "addr", s.cfg.SubmissionAddr)
		if err := s.submission.ListenAndServe(); err != nil {
			s.logger.Error("SMTP submission server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.inbound.Close()
	s.submission.Close()
	return nil
}
