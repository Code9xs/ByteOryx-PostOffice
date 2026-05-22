package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	QuotaBytes   int64     `json:"quota_bytes"`
	UsedBytes    int64     `json:"used_bytes"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Domain struct {
	ID                uuid.UUID `json:"id"`
	Name              string    `json:"name"`
	OwnerID           uuid.UUID `json:"owner_id"`
	IsVerified        bool      `json:"is_verified"`
	MXVerified        bool      `json:"mx_verified"`
	SPFVerified       bool      `json:"spf_verified"`
	DKIMVerified      bool      `json:"dkim_verified"`
	DMARCVerified     bool      `json:"dmarc_verified"`
	VerificationToken string    `json:"verification_token"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type DKIMKey struct {
	ID         uuid.UUID `json:"id"`
	DomainID   uuid.UUID `json:"domain_id"`
	Selector   string    `json:"selector"`
	PrivateKey string    `json:"-"`
	PublicKey  string    `json:"public_key"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type Mailbox struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	DomainID   uuid.UUID `json:"domain_id"`
	LocalPart  string    `json:"local_part"`
	Address    string    `json:"address"`
	IsActive   bool      `json:"is_active"`
	IsCatchAll bool      `json:"is_catchall"`
	CreatedAt  time.Time `json:"created_at"`
}

type Folder struct {
	ID           uuid.UUID `json:"id"`
	MailboxID    uuid.UUID `json:"mailbox_id"`
	Name         string    `json:"name"`
	SpecialUse   string    `json:"special_use,omitempty"`
	UIDValidity  int       `json:"uid_validity"`
	UIDNext      int       `json:"uid_next"`
	MessageCount int       `json:"message_count"`
	UnseenCount  int       `json:"unseen_count"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID             uuid.UUID  `json:"id"`
	FolderID       uuid.UUID  `json:"folder_id"`
	MailboxID      uuid.UUID  `json:"mailbox_id"`
	UID            int        `json:"uid"`
	MessageID      string     `json:"message_id,omitempty"`
	InReplyTo      string     `json:"in_reply_to,omitempty"`
	Subject        string     `json:"subject"`
	FromAddress    string     `json:"from"`
	ToAddresses    []string   `json:"to"`
	CCAddresses    []string   `json:"cc,omitempty"`
	Date           time.Time  `json:"date"`
	SizeBytes      int        `json:"size"`
	HasAttachments bool       `json:"has_attachments"`
	IsSeen         bool       `json:"is_seen"`
	IsAnswered     bool       `json:"is_answered"`
	IsFlagged      bool       `json:"is_flagged"`
	IsDeleted      bool       `json:"is_deleted"`
	IsDraft        bool       `json:"is_draft"`
	StorageKey     string     `json:"-"`
	SpamScore      *float64   `json:"spam_score,omitempty"`
	IsSpam         bool       `json:"is_spam"`
	CreatedAt      time.Time  `json:"created_at"`
}

type SendQueueItem struct {
	ID          uuid.UUID  `json:"id"`
	FromAddress string     `json:"from_address"`
	ToAddresses []string   `json:"to_addresses"`
	StorageKey  string     `json:"storage_key"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type APIKey struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	KeyHash     string      `json:"-"`
	KeyPrefix   string      `json:"key_prefix"`
	Name        string      `json:"name"`
	MailboxIDs  []uuid.UUID `json:"mailbox_ids"`
	Permissions []string    `json:"permissions"`
	IsActive    bool        `json:"is_active"`
	LastUsedAt  *time.Time  `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type Alias struct {
	ID             uuid.UUID `json:"id"`
	SourceAddress  string    `json:"source_address"`
	Destination    string    `json:"destination"`
	DomainID       uuid.UUID `json:"domain_id"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type Webhook struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
	Events    []string  `json:"events"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
