package smtp

import (
	"log/slog"

	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
	"github.com/emersion/go-smtp"
)

type Backend struct {
	mailboxRepo repository.MailboxRepository
	messageRepo repository.MessageRepository
	folderRepo  repository.FolderRepository
	queueRepo   repository.SendQueueRepository
	userRepo    repository.UserRepository
	msgStore    *storage.MessageStore
	hostname    string
	logger      *slog.Logger
	requireAuth bool
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		backend: b,
		conn:    c,
		logger:  b.logger,
	}, nil
}
