package repository

import (
	"context"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*model.User, int, error)
}

type DomainRepository interface {
	Create(ctx context.Context, domain *model.Domain) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Domain, error)
	GetByName(ctx context.Context, name string) (*model.Domain, error)
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Domain, error)
	Update(ctx context.Context, domain *model.Domain) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MailboxRepository interface {
	Create(ctx context.Context, mailbox *model.Mailbox) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Mailbox, error)
	GetByAddress(ctx context.Context, address string) (*model.Mailbox, error)
	GetCatchAll(ctx context.Context, domainID uuid.UUID) (*model.Mailbox, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.Mailbox, error)
	ListByDomain(ctx context.Context, domainID uuid.UUID) ([]*model.Mailbox, error)
	Update(ctx context.Context, mailbox *model.Mailbox) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type FolderRepository interface {
	Create(ctx context.Context, folder *model.Folder) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Folder, error)
	GetByName(ctx context.Context, mailboxID uuid.UUID, name string) (*model.Folder, error)
	ListByMailbox(ctx context.Context, mailboxID uuid.UUID) ([]*model.Folder, error)
	Update(ctx context.Context, folder *model.Folder) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementUIDNext(ctx context.Context, id uuid.UUID) (int, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *model.Message) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error)
	GetByUID(ctx context.Context, folderID uuid.UUID, uid int) (*model.Message, error)
	ListByFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]*model.Message, int, error)
	ListByMailbox(ctx context.Context, mailboxID uuid.UUID, opts MessageListOptions) ([]*model.Message, int, error)
	UpdateFlags(ctx context.Context, id uuid.UUID, flags MessageFlags) error
	Move(ctx context.Context, id uuid.UUID, targetFolderID uuid.UUID, newUID int) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MessageListOptions struct {
	FolderName string
	Offset     int
	Limit      int
	Since      *string
	UnseenOnly bool
}

type MessageFlags struct {
	IsSeen     *bool
	IsAnswered *bool
	IsFlagged  *bool
	IsDeleted  *bool
	IsDraft    *bool
}

type SendQueueRepository interface {
	Enqueue(ctx context.Context, item *model.SendQueueItem) error
	Dequeue(ctx context.Context, batchSize int) ([]*model.SendQueueItem, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastError string) error
	IncrementAttempts(ctx context.Context, id uuid.UUID) error
}

type APIKeyRepository interface {
	Create(ctx context.Context, key *model.APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

type DKIMKeyRepository interface {
	Create(ctx context.Context, key *model.DKIMKey) error
	GetActive(ctx context.Context, domainID uuid.UUID) (*model.DKIMKey, error)
	ListByDomain(ctx context.Context, domainID uuid.UUID) ([]*model.DKIMKey, error)
}
