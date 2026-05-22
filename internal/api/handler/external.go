package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExternalHandler struct {
	apiKeyService *service.APIKeyService
}

func NewExternalHandler(apiKeyService *service.APIKeyService) *ExternalHandler {
	return &ExternalHandler{apiKeyService: apiKeyService}
}

type CreateAPIKeyRequest struct {
	Name        string      `json:"name" binding:"required"`
	MailboxIDs  []uuid.UUID `json:"mailbox_ids" binding:"required"`
	Permissions []string    `json:"permissions"`
	ExpiresIn   *int        `json:"expires_in_days"`
}

func (h *ExternalHandler) CreateAPIKey(c *gin.Context) {
	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))

	var expiresAt *time.Time
	if req.ExpiresIn != nil {
		t := time.Now().AddDate(0, 0, *req.ExpiresIn)
		expiresAt = &t
	}

	resp, err := h.apiKeyService.CreateKey(c.Request.Context(), service.CreateAPIKeyRequest{
		UserID:      userID,
		Name:        req.Name,
		MailboxIDs:  req.MailboxIDs,
		Permissions: req.Permissions,
		ExpiresAt:   expiresAt,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":     resp.Key,
		"api_key": resp.APIKey,
		"message": "Save this key now. It will not be shown again.",
	})
}

func (h *ExternalHandler) ListAPIKeys(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	keys, err := h.apiKeyService.ListKeys(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": keys})
}

func (h *ExternalHandler) RevokeAPIKey(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key id"})
		return
	}

	if err := h.apiKeyService.RevokeKey(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "api key revoked"})
}

func (h *ExternalHandler) FetchMessages(c *gin.Context) {
	apiKey, exists := c.Get("api_key")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	folder := c.DefaultQuery("folder", "INBOX")
	unseenOnly := c.Query("unseen_only") == "true"

	var since *string
	if s := c.Query("since"); s != "" {
		since = &s
	}

	messages, total, err := h.apiKeyService.FetchMessages(c.Request.Context(), apiKey.(*model.APIKey), service.FetchMessagesRequest{
		Address:    address,
		Limit:      limit,
		Offset:     offset,
		Folder:     folder,
		Since:      since,
		UnseenOnly: unseenOnly,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "access denied to mailbox: "+address {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    total,
		"messages": messages,
	})
}
