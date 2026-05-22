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

type DomainRepo struct {
	db *pgxpool.Pool
}

func NewDomainRepo(db *pgxpool.Pool) repository.DomainRepository {
	return &DomainRepo{db: db}
}

func (r *DomainRepo) Create(ctx context.Context, domain *model.Domain) error {
	query := `INSERT INTO domains (id, name, owner_id, verification_token)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, domain.ID, domain.Name, domain.OwnerID, domain.VerificationToken)
	return err
}

func (r *DomainRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Domain, error) {
	query := `SELECT id, name, owner_id, is_verified, mx_verified, spf_verified, dkim_verified, dmarc_verified,
		verification_token, created_at, updated_at FROM domains WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, query, id))
}

func (r *DomainRepo) GetByName(ctx context.Context, name string) (*model.Domain, error) {
	query := `SELECT id, name, owner_id, is_verified, mx_verified, spf_verified, dkim_verified, dmarc_verified,
		verification_token, created_at, updated_at FROM domains WHERE name = $1`
	return r.scan(r.db.QueryRow(ctx, query, name))
}

func (r *DomainRepo) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]*model.Domain, error) {
	query := `SELECT id, name, owner_id, is_verified, mx_verified, spf_verified, dkim_verified, dmarc_verified,
		verification_token, created_at, updated_at FROM domains WHERE owner_id = $1 ORDER BY name`
	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*model.Domain
	for rows.Next() {
		d := &model.Domain{}
		if err := rows.Scan(&d.ID, &d.Name, &d.OwnerID, &d.IsVerified, &d.MXVerified, &d.SPFVerified,
			&d.DKIMVerified, &d.DMARCVerified, &d.VerificationToken, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, nil
}

func (r *DomainRepo) Update(ctx context.Context, domain *model.Domain) error {
	query := `UPDATE domains SET is_verified=$2, mx_verified=$3, spf_verified=$4, dkim_verified=$5,
		dmarc_verified=$6, updated_at=NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, domain.ID, domain.IsVerified, domain.MXVerified,
		domain.SPFVerified, domain.DKIMVerified, domain.DMARCVerified)
	return err
}

func (r *DomainRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM domains WHERE id = $1`, id)
	return err
}

func (r *DomainRepo) scan(row pgx.Row) (*model.Domain, error) {
	d := &model.Domain{}
	err := row.Scan(&d.ID, &d.Name, &d.OwnerID, &d.IsVerified, &d.MXVerified, &d.SPFVerified,
		&d.DKIMVerified, &d.DMARCVerified, &d.VerificationToken, &d.CreatedAt, &d.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("domain not found")
	}
	return d, err
}

type APIKeyRepo struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepo(db *pgxpool.Pool) repository.APIKeyRepository {
	return &APIKeyRepo{db: db}
}

func (r *APIKeyRepo) Create(ctx context.Context, key *model.APIKey) error {
	query := `INSERT INTO api_keys (id, user_id, key_hash, key_prefix, name, mailbox_ids, permissions, is_active, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.Exec(ctx, query,
		key.ID, key.UserID, key.KeyHash, key.KeyPrefix, key.Name, key.MailboxIDs, key.Permissions, key.IsActive, key.ExpiresAt)
	return err
}

func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `SELECT id, user_id, key_hash, key_prefix, name, mailbox_ids, permissions, is_active, last_used_at, expires_at, created_at
		FROM api_keys WHERE key_hash = $1 AND is_active = true`
	k := &model.APIKey{}
	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.MailboxIDs, &k.Permissions, &k.IsActive, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("api key not found")
	}
	return k, err
}

func (r *APIKeyRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*model.APIKey, error) {
	query := `SELECT id, user_id, key_hash, key_prefix, name, mailbox_ids, permissions, is_active, last_used_at, expires_at, created_at
		FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.APIKey
	for rows.Next() {
		k := &model.APIKey{}
		if err := rows.Scan(&k.ID, &k.UserID, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.MailboxIDs,
			&k.Permissions, &k.IsActive, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (r *APIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET is_active = false WHERE id = $1`, id)
	return err
}

func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}
