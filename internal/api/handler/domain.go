package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DomainHandler struct {
	domainRepo  repository.DomainRepository
	mailboxRepo repository.MailboxRepository
	folderRepo  repository.FolderRepository
}

func NewDomainHandler(
	domainRepo repository.DomainRepository,
	mailboxRepo repository.MailboxRepository,
	folderRepo repository.FolderRepository,
) *DomainHandler {
	return &DomainHandler{
		domainRepo:  domainRepo,
		mailboxRepo: mailboxRepo,
		folderRepo:  folderRepo,
	}
}

type AddDomainRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateMailboxRequest struct {
	LocalPart string `json:"local_part" binding:"required"`
	DomainID  string `json:"domain_id" binding:"required"`
}

func (h *DomainHandler) AddDomain(c *gin.Context) {
	var req AddDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	token := generateVerificationToken()

	domain := &model.Domain{
		ID:                uuid.New(),
		Name:              req.Name,
		OwnerID:           userID,
		VerificationToken: token,
	}

	if err := h.domainRepo.Create(c.Request.Context(), domain); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "domain already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"domain": domain,
		"dns_records": gin.H{
			"txt_verification": "postoffice-verify=" + token,
			"mx":               "10 " + c.GetHeader("Host"),
			"spf":              "v=spf1 a mx -all",
		},
	})
}

func (h *DomainHandler) ListDomains(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	domains, err := h.domainRepo.ListByOwner(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

func (h *DomainHandler) GetDomain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		return
	}

	domain, err := h.domainRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"domain": domain})
}

func (h *DomainHandler) DeleteDomain(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		return
	}

	if err := h.domainRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "domain deleted"})
}

func (h *DomainHandler) CreateMailbox(c *gin.Context) {
	var req CreateMailboxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	domainID, _ := uuid.Parse(req.DomainID)

	domain, err := h.domainRepo.GetByID(c.Request.Context(), domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	address := req.LocalPart + "@" + domain.Name
	mailbox := &model.Mailbox{
		ID:        uuid.New(),
		UserID:    userID,
		DomainID:  domainID,
		LocalPart: req.LocalPart,
		Address:   address,
		IsActive:  true,
	}

	if err := h.mailboxRepo.Create(c.Request.Context(), mailbox); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "mailbox already exists"})
		return
	}

	// Create default folders
	defaultFolders := []struct {
		Name       string
		SpecialUse string
	}{
		{"INBOX", "\\Inbox"},
		{"Sent", "\\Sent"},
		{"Drafts", "\\Drafts"},
		{"Trash", "\\Trash"},
		{"Junk", "\\Junk"},
	}

	for _, f := range defaultFolders {
		folder := &model.Folder{
			ID:          uuid.New(),
			MailboxID:   mailbox.ID,
			Name:        f.Name,
			SpecialUse:  f.SpecialUse,
			UIDValidity: 1,
			UIDNext:     1,
		}
		h.folderRepo.Create(c.Request.Context(), folder)
	}

	c.JSON(http.StatusCreated, gin.H{"mailbox": mailbox})
}

func (h *DomainHandler) ListMailboxes(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	mailboxes, err := h.mailboxRepo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mailboxes": mailboxes})
}

func generateVerificationToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
