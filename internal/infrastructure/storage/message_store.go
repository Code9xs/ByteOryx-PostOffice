package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type MessageStore struct {
	client *minio.Client
	bucket string
}

func NewMessageStore(client *minio.Client, bucket string) *MessageStore {
	return &MessageStore{client: client, bucket: bucket}
}

func (s *MessageStore) Store(ctx context.Context, mailboxID uuid.UUID, data []byte) (string, error) {
	key := s.generateKey(mailboxID)
	reader := bytes.NewReader(data)

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "message/rfc822",
	})
	if err != nil {
		return "", fmt.Errorf("store message: %w", err)
	}
	return key, nil
}

func (s *MessageStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("read message: %w", err)
	}
	return data, nil
}

func (s *MessageStore) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *MessageStore) generateKey(mailboxID uuid.UUID) string {
	now := time.Now()
	return path.Join(
		mailboxID.String(),
		now.Format("2006/01/02"),
		uuid.New().String()+".eml",
	)
}
