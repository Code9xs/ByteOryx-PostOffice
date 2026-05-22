package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
	"github.com/byteoryx/postoffice/internal/pkg/email"
	"github.com/google/uuid"
)

type APIKeyService struct {
	apiKeyRepo  repository.APIKeyRepository
	mailboxRepo repository.MailboxRepository
	messageRepo repository.MessageRepository
	msgStore    *storage.MessageStore
}

func NewAPIKeyService(
	apiKeyRepo repository.APIKeyRepository,
	mailboxRepo repository.MailboxRepository,
	messageRepo repository.MessageRepository,
	msgStore *storage.MessageStore,
) *APIKeyService {
	return &APIKeyService{
		apiKeyRepo:  apiKeyRepo,
		mailboxRepo: mailboxRepo,
		messageRepo: messageRepo,
		msgStore:    msgStore,
	}
}

type CreateAPIKeyRequest struct {
	UserID      uuid.UUID
	Name        string
	MailboxIDs  []uuid.UUID
	Permissions []string
	ExpiresAt   *time.Time
}

type CreateAPIKeyResponse struct {
	Key    string        `json:"key"`
	APIKey *model.APIKey `json:"api_key"`
}

func (s *APIKeyService) CreateKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	rawKey := generateAPIKey()
	keyHash := hashAPIKey(rawKey)
	keyPrefix := rawKey[:8]

	if len(req.Permissions) == 0 {
		req.Permissions = []string{"read"}
	}

	apiKey := &model.APIKey{
		ID:          uuid.New(),
		UserID:      req.UserID,
		KeyHash:     keyHash,
		KeyPrefix:   keyPrefix,
		Name:        req.Name,
		MailboxIDs:  req.MailboxIDs,
		Permissions: req.Permissions,
		IsActive:    true,
		ExpiresAt:   req.ExpiresAt,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}

	return &CreateAPIKeyResponse{
		Key:    rawKey,
		APIKey: apiKey,
	}, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := hashAPIKey(rawKey)
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("api key expired")
	}

	go s.apiKeyRepo.UpdateLastUsed(context.Background(), apiKey.ID)

	return apiKey, nil
}

func (s *APIKeyService) HasMailboxAccess(apiKey *model.APIKey, mailboxID uuid.UUID) bool {
	for _, id := range apiKey.MailboxIDs {
		if id == mailboxID {
			return true
		}
	}
	return false
}

func (s *APIKeyService) ListKeys(ctx context.Context, userID uuid.UUID) ([]*model.APIKey, error) {
	return s.apiKeyRepo.ListByUser(ctx, userID)
}

func (s *APIKeyService) RevokeKey(ctx context.Context, id uuid.UUID) error {
	return s.apiKeyRepo.Delete(ctx, id)
}

type FetchMessagesRequest struct {
	Address    string
	Limit      int
	Offset     int
	Folder     string
	Since      *string
	UnseenOnly bool
}

type MessageWithBody struct {
	*model.Message
	BodyText    string       `json:"body_text"`
	BodyHTML    string       `json:"body_html"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	Size        int    `json:"size"`
	ContentType string `json:"content_type"`
	DownloadURL string `json:"download_url"`
}

func (s *APIKeyService) FetchMessages(ctx context.Context, apiKey *model.APIKey, req FetchMessagesRequest) ([]*MessageWithBody, int, error) {
	mailbox, err := s.mailboxRepo.GetByAddress(ctx, req.Address)
	if err != nil {
		return nil, 0, fmt.Errorf("mailbox not found: %s", req.Address)
	}

	if !s.HasMailboxAccess(apiKey, mailbox.ID) {
		return nil, 0, fmt.Errorf("access denied to mailbox: %s", req.Address)
	}

	folder := req.Folder
	if folder == "" {
		folder = "INBOX"
	}

	limit := req.Limit
	if limit == -1 {
		limit = 0 // 0 means no limit in the query
	} else if limit <= 0 {
		limit = 20
	}

	opts := repository.MessageListOptions{
		FolderName: folder,
		Offset:     req.Offset,
		Limit:      limit,
		Since:      req.Since,
		UnseenOnly: req.UnseenOnly,
	}

	messages, total, err := s.messageRepo.ListByMailbox(ctx, mailbox.ID, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("list messages: %w", err)
	}

	result := make([]*MessageWithBody, 0, len(messages))
	for _, msg := range messages {
		msgWithBody := &MessageWithBody{Message: msg}

		rawData, err := s.msgStore.Get(ctx, msg.StorageKey)
		if err == nil {
			bodyText, bodyHTML, attachments := parseMessageContent(rawData, msg.ID.String())
			msgWithBody.BodyText = bodyText
			msgWithBody.BodyHTML = bodyHTML
			msgWithBody.Attachments = attachments
		}

		result = append(result, msgWithBody)
	}

	return result, total, nil
}

func generateAPIKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return "po_" + hex.EncodeToString(b)[:32]
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func parseMessageContent(rawData []byte, msgID string) (string, string, []Attachment) {
	parsed, err := email.Parse(rawData)
	if err != nil {
		return string(rawData), "", nil
	}

	var attachments []Attachment
	for i, att := range parsed.Attachments {
		attachments = append(attachments, Attachment{
			Filename:    att.Filename,
			Size:        att.Size,
			ContentType: att.ContentType,
			DownloadURL: fmt.Sprintf("/api/v1/external/messages/%s/attachments/%d", msgID, i),
		})
	}

	return parsed.BodyText, parsed.BodyHTML, attachments
}
