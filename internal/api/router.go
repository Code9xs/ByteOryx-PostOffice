package api

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/byteoryx/postoffice/internal/api/handler"
	"github.com/byteoryx/postoffice/internal/api/middleware"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/byteoryx/postoffice/internal/service"
	"github.com/gin-gonic/gin"
)

func SetupRouter(
	authService *service.AuthService,
	apiKeyService *service.APIKeyService,
	domainRepo repository.DomainRepository,
	mailboxRepo repository.MailboxRepository,
	folderRepo repository.FolderRepository,
	logger *slog.Logger,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))
	r.Use(corsMiddleware())
	r.Use(middleware.RateLimit(120))
	r.Use(middleware.MetricsCollector())

	authHandler := handler.NewAuthHandler(authService)
	externalHandler := handler.NewExternalHandler(apiKeyService)
	domainHandler := handler.NewDomainHandler(domainRepo, mailboxRepo, folderRepo)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/metrics", middleware.MetricsHandler())

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Authenticated user routes
		user := v1.Group("")
		user.Use(middleware.JWTAuth(authService))
		{
			// Domain management
			domains := user.Group("/domains")
			{
				domains.POST("", domainHandler.AddDomain)
				domains.GET("", domainHandler.ListDomains)
				domains.GET("/:id", domainHandler.GetDomain)
				domains.DELETE("/:id", domainHandler.DeleteDomain)
			}

			// Mailbox management
			mailboxes := user.Group("/mailboxes")
			{
				mailboxes.POST("", domainHandler.CreateMailbox)
				mailboxes.GET("", domainHandler.ListMailboxes)
			}

			// API Key management
			apikeys := user.Group("/apikeys")
			{
				apikeys.POST("", externalHandler.CreateAPIKey)
				apikeys.GET("", externalHandler.ListAPIKeys)
				apikeys.DELETE("/:id", externalHandler.RevokeAPIKey)
			}
		}

		// External API (API Key auth)
		external := v1.Group("/external")
		external.Use(middleware.APIKeyAuth(apiKeyService))
		{
			external.GET("/mailboxes/:address/messages", externalHandler.FetchMessages)
		}
	}

	// Serve frontend static files
	webDir := "/var/www/postoffice"
	if dir := os.Getenv("PO_WEB_DIR"); dir != "" {
		webDir = dir
	}
	if _, err := os.Stat(webDir); err == nil {
		r.Static("/assets", filepath.Join(webDir, "assets"))
		r.NoRoute(func(c *gin.Context) {
			c.File(filepath.Join(webDir, "index.html"))
		})
	}

	return r
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
