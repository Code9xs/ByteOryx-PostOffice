package imap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	"github.com/byteoryx/postoffice/internal/config"
	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
	"github.com/byteoryx/postoffice/internal/service"
)

type Server struct {
	cfg         *config.IMAPConfig
	authService *service.AuthService
	userRepo    repository.UserRepository
	mailboxRepo repository.MailboxRepository
	folderRepo  repository.FolderRepository
	messageRepo repository.MessageRepository
	msgStore    *storage.MessageStore
	tlsConfig   *tls.Config
	logger      *slog.Logger
	listener    net.Listener
	stop        chan struct{}
}

func NewServer(
	cfg *config.IMAPConfig,
	authService *service.AuthService,
	userRepo repository.UserRepository,
	mailboxRepo repository.MailboxRepository,
	folderRepo repository.FolderRepository,
	messageRepo repository.MessageRepository,
	msgStore *storage.MessageStore,
	tlsConfig *tls.Config,
	logger *slog.Logger,
) *Server {
	return &Server{
		cfg:         cfg,
		authService: authService,
		userRepo:    userRepo,
		mailboxRepo: mailboxRepo,
		folderRepo:  folderRepo,
		messageRepo: messageRepo,
		msgStore:    msgStore,
		tlsConfig:   tlsConfig,
		logger:      logger,
		stop:        make(chan struct{}),
	}
}

func (s *Server) Start() error {
	var ln net.Listener
	var err error

	if s.tlsConfig != nil {
		ln, err = tls.Listen("tcp", s.cfg.ListenAddr, s.tlsConfig)
	} else {
		ln, err = net.Listen("tcp", s.cfg.ListenAddr)
	}
	if err != nil {
		return fmt.Errorf("imap listen: %w", err)
	}
	s.listener = ln

	s.logger.Info("IMAP server started", "addr", s.cfg.ListenAddr)

	go s.acceptLoop()
	return nil
}

func (s *Server) Stop() error {
	close(s.stop)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				return
			default:
				s.logger.Error("IMAP accept error", "error", err)
				continue
			}
		}
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	session := newSession(conn, s)
	defer session.close()
	session.serve()
}

func (s *Server) authenticate(ctx context.Context, username, password string) (*model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, username)
	if err != nil {
		// Try username as mailbox address
		mailbox, mErr := s.mailboxRepo.GetByAddress(ctx, username)
		if mErr != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
		user, err = s.userRepo.GetByID(ctx, mailbox.UserID)
		if err != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
	}
	// TODO: verify password properly (reuse auth service logic)
	_ = password
	return user, nil
}
