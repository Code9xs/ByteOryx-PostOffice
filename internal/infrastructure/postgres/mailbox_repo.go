package postgres

import (
	"context"
	"fmt"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MailboxRepo struct {
	db *pgxpool.Pool
}

func NewMailboxRepo(db *pgxpool.Pool) repository.MailboxRepository {
	return &MailboxRepo{db: db}
}

func (r *MailboxRepo) Create(ctx context.Context, mailbox *model.Mailbox) error {
	query := `INSERT INTO mailboxes (id, user_id, domain_id, local_part, address, is_active, is_catchall)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		mailbox.ID, mailbox.UserID, mailbox.DomainID, mailbox.LocalPart, mailbox.Address, mailbox.IsActive, mailbox.IsCatchAll)
	return err
}

func (r *MailboxRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Mailbox, error) {
	query := `SELECT id, user_id, domain_id, local_part, address, is_active, is_catchall, created_at
		FROM mailboxes WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, query, id))
}

func (r *MailboxRepo) GetByAddress(ctx context.Context, address string) (*model.Mailbox, error) {
	query := `SELECT id, user_id, domain_id, local_part, address, is_active, is_catchall, created_at
		FROM mailboxes WHERE address = $1 AND is_active = true`
	return r.scan(r.db.QueryRow(ctx, query, address))
}

func (r *MailboxRepo) GetCatchAll(ctx context.Context, domainID uuid.UUID) (*model.Mailbox, error) {
	query := `SELECT id, user_id, domain_id, local_part, address, is_active, is_catchall, created_at
		FROM mailboxes WHERE domain_id = $1 AND is_catchall = true AND is_active = true`
	return r.scan(r.db.QueryRow(ctx, query, domainID))
}

func (r *MailboxRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.Mailbox, error) {
	query := `SELECT id, user_id, domain_id, local_part, address, is_active, is_catchall, created_at
		FROM mailboxes WHERE user_id = $1 ORDER BY created_at`
	return r.list(ctx, query, userID)
}

func (r *MailboxRepo) ListByDomain(ctx context.Context, domainID uuid.UUID) ([]*model.Mailbox, error) {
	query := `SELECT id, user_id, domain_id, local_part, address, is_active, is_catchall, created_at
		FROM mailboxes WHERE domain_id = $1 ORDER BY local_part`
	return r.list(ctx, query, domainID)
}

func (r *MailboxRepo) Update(ctx context.Context, mailbox *model.Mailbox) error {
	query := `UPDATE mailboxes SET is_active=$2, is_catchall=$3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, mailbox.ID, mailbox.IsActive, mailbox.IsCatchAll)
	return err
}

func (r *MailboxRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM mailboxes WHERE id = $1`, id)
	return err
}

func (r *MailboxRepo) scan(row pgx.Row) (*model.Mailbox, error) {
	m := &model.Mailbox{}
	err := row.Scan(&m.ID, &m.UserID, &m.DomainID, &m.LocalPart, &m.Address, &m.IsActive, &m.IsCatchAll, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("mailbox not found")
	}
	return m, err
}

func (r *MailboxRepo) list(ctx context.Context, query string, arg interface{}) ([]*model.Mailbox, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Mailbox
	for rows.Next() {
		m := &model.Mailbox{}
		if err := rows.Scan(&m.ID, &m.UserID, &m.DomainID, &m.LocalPart, &m.Address, &m.IsActive, &m.IsCatchAll, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, nil
}
