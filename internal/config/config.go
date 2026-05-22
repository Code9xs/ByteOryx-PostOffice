package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	SMTP     SMTPConfig     `yaml:"smtp"`
	IMAP     IMAPConfig     `yaml:"imap"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Storage  StorageConfig  `yaml:"storage"`
	TLS      TLSConfig      `yaml:"tls"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type ServerConfig struct {
	Hostname string `yaml:"hostname"`
	HTTPPort int    `yaml:"http_port"`
}

type SMTPConfig struct {
	ListenAddr     string `yaml:"listen_addr"`
	SubmissionAddr string `yaml:"submission_addr"`
	MaxMessageSize int64  `yaml:"max_message_size"`
	MaxRecipients  int    `yaml:"max_recipients"`
}

type IMAPConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode"`
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode)
}

type RedisConfig struct {
	URL string `yaml:"url"`
}

type StorageConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
}

type TLSConfig struct {
	Enabled   bool   `yaml:"enabled"`
	CertFile  string `yaml:"cert_file"`
	KeyFile   string `yaml:"key_file"`
	ACMEEmail string `yaml:"acme_email"`
}

type JWTConfig struct {
	Secret     string `yaml:"secret"`
	ExpireHour int    `yaml:"expire_hour"`
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Hostname: "localhost",
			HTTPPort: 8080,
		},
		SMTP: SMTPConfig{
			ListenAddr:     ":25",
			SubmissionAddr: ":587",
			MaxMessageSize: 25 * 1024 * 1024,
			MaxRecipients:  100,
		},
		IMAP: IMAPConfig{
			ListenAddr: ":993",
		},
		Database: DatabaseConfig{
			Host:    "localhost",
			Port:    5432,
			Name:    "postoffice",
			User:    "postoffice",
			SSLMode: "disable",
		},
		Redis: RedisConfig{
			URL: "redis://localhost:6379",
		},
		Storage: StorageConfig{
			Endpoint: "localhost:9000",
			Bucket:   "postoffice",
		},
		JWT: JWTConfig{
			ExpireHour: 72,
		},
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("PO_HOSTNAME"); v != "" {
		cfg.Server.Hostname = v
	}
	if v := os.Getenv("PO_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.HTTPPort = port
		}
	}
	if v := os.Getenv("PO_DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("PO_DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("PO_DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("PO_DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("PO_DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("PO_REDIS_URL"); v != "" {
		cfg.Redis.URL = v
	}
	if v := os.Getenv("PO_S3_ENDPOINT"); v != "" {
		cfg.Storage.Endpoint = v
	}
	if v := os.Getenv("PO_S3_ACCESS_KEY"); v != "" {
		cfg.Storage.AccessKey = v
	}
	if v := os.Getenv("PO_S3_SECRET_KEY"); v != "" {
		cfg.Storage.SecretKey = v
	}
	if v := os.Getenv("PO_S3_BUCKET"); v != "" {
		cfg.Storage.Bucket = v
	}
	if v := os.Getenv("PO_S3_USE_SSL"); v != "" {
		cfg.Storage.UseSSL = strings.EqualFold(v, "true")
	}
	if v := os.Getenv("PO_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("PO_TLS_ACME_EMAIL"); v != "" {
		cfg.TLS.ACMEEmail = v
	}
}
