package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FolderRepo struct {
	db *pgxpool.Pool
}

func NewFolderRepo(db *pgxpool.Pool) repository.FolderRepository {
	return &FolderRepo{db: db}
}

func (r *FolderRepo) Create(ctx context.Context, folder *model.Folder) error {
	query := `INSERT INTO folders (id, mailbox_id, name, special_use, uid_validity, uid_next)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query,
		folder.ID, folder.MailboxID, folder.Name, folder.SpecialUse, folder.UIDValidity, folder.UIDNext)
	return err
}

func (r *FolderRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Folder, error) {
	query := `SELECT id, mailbox_id, name, special_use, uid_validity, uid_next, message_count, unseen_count, created_at
		FROM folders WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, query, id))
}

func (r *FolderRepo) GetByName(ctx context.Context, mailboxID uuid.UUID, name string) (*model.Folder, error) {
	query := `SELECT id, mailbox_id, name, special_use, uid_validity, uid_next, message_count, unseen_count, created_at
		FROM folders WHERE mailbox_id = $1 AND name = $2`
	return r.scan(r.db.QueryRow(ctx, query, mailboxID, name))
}

func (r *FolderRepo) ListByMailbox(ctx context.Context, mailboxID uuid.UUID) ([]*model.Folder, error) {
	query := `SELECT id, mailbox_id, name, special_use, uid_validity, uid_next, message_count, unseen_count, created_at
		FROM folders WHERE mailbox_id = $1 ORDER BY name`
	rows, err := r.db.Query(ctx, query, mailboxID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f, err := r.scanFromRows(rows)
		if err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, nil
}

func (r *FolderRepo) Update(ctx context.Context, folder *model.Folder) error {
	query := `UPDATE folders SET name=$2, special_use=$3, message_count=$4, unseen_count=$5 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, folder.ID, folder.Name, folder.SpecialUse, folder.MessageCount, folder.UnseenCount)
	return err
}

func (r *FolderRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM folders WHERE id = $1`, id)
	return err
}

func (r *FolderRepo) IncrementUIDNext(ctx context.Context, id uuid.UUID) (int, error) {
	var uid int
	err := r.db.QueryRow(ctx,
		`UPDATE folders SET uid_next = uid_next + 1, message_count = message_count + 1
		WHERE id = $1 RETURNING uid_next - 1`, id).Scan(&uid)
	return uid, err
}

func (r *FolderRepo) scan(row pgx.Row) (*model.Folder, error) {
	f := &model.Folder{}
	err := row.Scan(&f.ID, &f.MailboxID, &f.Name, &f.SpecialUse, &f.UIDValidity, &f.UIDNext,
		&f.MessageCount, &f.UnseenCount, &f.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("folder not found")
	}
	return f, err
}

func (r *FolderRepo) scanFromRows(rows pgx.Rows) (*model.Folder, error) {
	f := &model.Folder{}
	err := rows.Scan(&f.ID, &f.MailboxID, &f.Name, &f.SpecialUse, &f.UIDValidity, &f.UIDNext,
		&f.MessageCount, &f.UnseenCount, &f.CreatedAt)
	return f, err
}

type SendQueueRepo struct {
	db *pgxpool.Pool
}

func NewSendQueueRepo(db *pgxpool.Pool) repository.SendQueueRepository {
	return &SendQueueRepo{db: db}
}

func (r *SendQueueRepo) Enqueue(ctx context.Context, item *model.SendQueueItem) error {
	query := `INSERT INTO send_queue (id, from_address, to_addresses, storage_key, status, max_attempts)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query,
		item.ID, item.FromAddress, item.ToAddresses, item.StorageKey, item.Status, item.MaxAttempts)
	return err
}

func (r *SendQueueRepo) Dequeue(ctx context.Context, batchSize int) ([]*model.SendQueueItem, error) {
	query := `UPDATE send_queue SET status = 'sending', updated_at = NOW()
		WHERE id IN (
			SELECT id FROM send_queue
			WHERE status = 'pending' OR (status = 'deferred' AND next_retry_at <= $1)
			ORDER BY created_at
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		) RETURNING id, from_address, to_addresses, storage_key, status, attempts, max_attempts, next_retry_at, last_error, created_at, updated_at`

	rows, err := r.db.Query(ctx, query, time.Now(), batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*model.SendQueueItem
	for rows.Next() {
		item := &model.SendQueueItem{}
		if err := rows.Scan(&item.ID, &item.FromAddress, &item.ToAddresses, &item.StorageKey,
			&item.Status, &item.Attempts, &item.MaxAttempts, &item.NextRetryAt, &item.LastError,
			&item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *SendQueueRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string, lastError string) error {
	var nextRetry *time.Time
	if status == "deferred" {
		t := time.Now().Add(5 * time.Minute)
		nextRetry = &t
	}
	query := `UPDATE send_queue SET status = $2, last_error = $3, next_retry_at = $4, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, status, lastError, nextRetry)
	return err
}

func (r *SendQueueRepo) IncrementAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE send_queue SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}
