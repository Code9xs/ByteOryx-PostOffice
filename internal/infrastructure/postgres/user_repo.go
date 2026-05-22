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

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) repository.UserRepository {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (id, email, password_hash, display_name, role, is_active, quota_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.DisplayName, user.Role, user.IsActive, user.QuotaBytes)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `SELECT id, email, password_hash, display_name, role, is_active, quota_bytes, used_bytes, created_at, updated_at
		FROM users WHERE id = $1`
	return r.scanUser(r.db.QueryRow(ctx, query, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `SELECT id, email, password_hash, display_name, role, is_active, quota_bytes, used_bytes, created_at, updated_at
		FROM users WHERE email = $1`
	return r.scanUser(r.db.QueryRow(ctx, query, email))
}

func (r *UserRepo) Update(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET email=$2, display_name=$3, role=$4, is_active=$5, quota_bytes=$6, used_bytes=$7, updated_at=NOW()
		WHERE id = $1`
	_, err := r.db.Exec(ctx, query,
		user.ID, user.Email, user.DisplayName, user.Role, user.IsActive, user.QuotaBytes, user.UsedBytes)
	return err
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *UserRepo) List(ctx context.Context, offset, limit int) ([]*model.User, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, email, password_hash, display_name, role, is_active, quota_bytes, used_bytes, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u, err := r.scanUserFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *UserRepo) scanUser(row pgx.Row) (*model.User, error) {
	u := &model.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role,
		&u.IsActive, &u.QuotaBytes, &u.UsedBytes, &u.CreatedAt, &u.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	return u, err
}

func (r *UserRepo) scanUserFromRows(rows pgx.Rows) (*model.User, error) {
	u := &model.User{}
	err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.Role,
		&u.IsActive, &u.QuotaBytes, &u.UsedBytes, &u.CreatedAt, &u.UpdatedAt)
	return u, err
}
