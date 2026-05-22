package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/byteoryx/postoffice/internal/api"
	"github.com/byteoryx/postoffice/internal/config"
	imapserver "github.com/byteoryx/postoffice/internal/imap"
	"github.com/byteoryx/postoffice/internal/infrastructure/postgres"
	"github.com/byteoryx/postoffice/internal/infrastructure/storage"
	"github.com/byteoryx/postoffice/internal/service"
	smtpserver "github.com/byteoryx/postoffice/internal/smtp"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
)

type App struct {
	cfg         *config.Config
	logger      *slog.Logger
	db          *pgxpool.Pool
	rdb         *redis.Client
	s3          *minio.Client
	httpServer  *http.Server
	smtpServer  *smtpserver.Server
	imapServer  *imapserver.Server
	queueWorker *smtpserver.QueueWorker
}

func New(cfg *config.Config, logger *slog.Logger) (*App, error) {
	a := &App{
		cfg:    cfg,
		logger: logger,
	}

	if err := a.initDB(); err != nil {
		return nil, fmt.Errorf("init database: %w", err)
	}

	if err := a.initRedis(); err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}

	if err := a.initStorage(); err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	return a, nil
}

func (a *App) Start(ctx context.Context) error {
	// Initialize repositories
	userRepo := postgres.NewUserRepo(a.db)
	domainRepo := postgres.NewDomainRepo(a.db)
	mailboxRepo := postgres.NewMailboxRepo(a.db)
	messageRepo := postgres.NewMessageRepo(a.db)
	folderRepo := postgres.NewFolderRepo(a.db)
	queueRepo := postgres.NewSendQueueRepo(a.db)
	apiKeyRepo := postgres.NewAPIKeyRepo(a.db)

	// Initialize storage
	msgStore := storage.NewMessageStore(a.s3, a.cfg.Storage.Bucket)

	// Initialize services
	authService := service.NewAuthService(userRepo, &a.cfg.JWT)
	apiKeyService := service.NewAPIKeyService(apiKeyRepo, mailboxRepo, messageRepo, msgStore)

	// Setup HTTP API
	router := api.SetupRouter(authService, apiKeyService, domainRepo, mailboxRepo, folderRepo, a.logger)
	a.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.cfg.Server.HTTPPort),
		Handler: router,
	}

	go func() {
		a.logger.Info("starting HTTP server", "port", a.cfg.Server.HTTPPort)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server error", "error", err)
		}
	}()

	// Start SMTP server
	a.smtpServer = smtpserver.NewServer(
		&a.cfg.SMTP,
		a.cfg.Server.Hostname,
		mailboxRepo,
		messageRepo,
		folderRepo,
		queueRepo,
		userRepo,
		msgStore,
		nil, // TLS config - will be added later
		a.logger,
	)
	if err := a.smtpServer.Start(ctx); err != nil {
		return fmt.Errorf("start SMTP server: %w", err)
	}

	// Start queue worker
	a.queueWorker = smtpserver.NewQueueWorker(queueRepo, msgStore, a.cfg.Server.Hostname, a.logger)
	a.queueWorker.Start(ctx)

	// Start IMAP server
	a.imapServer = imapserver.NewServer(
		&a.cfg.IMAP,
		authService,
		userRepo,
		mailboxRepo,
		folderRepo,
		messageRepo,
		msgStore,
		nil, // TLS config - will be added later
		a.logger,
	)
	if err := a.imapServer.Start(); err != nil {
		return fmt.Errorf("start IMAP server: %w", err)
	}

	a.logger.Info("PostOffice started",
		"hostname", a.cfg.Server.Hostname,
		"http", a.cfg.Server.HTTPPort,
		"smtp", a.cfg.SMTP.ListenAddr,
		"submission", a.cfg.SMTP.SubmissionAddr,
		"imap", a.cfg.IMAP.ListenAddr,
	)

	return nil
}

func (a *App) Stop() error {
	if a.queueWorker != nil {
		a.queueWorker.Stop()
	}
	if a.imapServer != nil {
		a.imapServer.Stop()
	}
	if a.smtpServer != nil {
		a.smtpServer.Stop()
	}
	if a.httpServer != nil {
		if err := a.httpServer.Close(); err != nil {
			return err
		}
	}
	if a.db != nil {
		a.db.Close()
	}
	if a.rdb != nil {
		a.rdb.Close()
	}
	return nil
}

func (a *App) initDB() error {
	pool, err := pgxpool.New(context.Background(), a.cfg.Database.DSN())
	if err != nil {
		return err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	a.db = pool
	a.logger.Info("connected to PostgreSQL")
	return nil
}

func (a *App) initRedis() error {
	opts, err := redis.ParseURL(a.cfg.Redis.URL)
	if err != nil {
		return err
	}
	a.rdb = redis.NewClient(opts)
	if err := a.rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	a.logger.Info("connected to Redis")
	return nil
}

func (a *App) initStorage() error {
	client, err := minio.New(a.cfg.Storage.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(a.cfg.Storage.AccessKey, a.cfg.Storage.SecretKey, ""),
		Secure: a.cfg.Storage.UseSSL,
	})
	if err != nil {
		return err
	}

	ctx := context.Background()
	exists, err := client.BucketExists(ctx, a.cfg.Storage.Bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, a.cfg.Storage.Bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}

	a.s3 = client
	a.logger.Info("connected to S3/MinIO")
	return nil
}
