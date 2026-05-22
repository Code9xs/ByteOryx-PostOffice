package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageRepo struct {
	db *pgxpool.Pool
}

func NewMessageRepo(db *pgxpool.Pool) repository.MessageRepository {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(ctx context.Context, msg *model.Message) error {
	query := `INSERT INTO messages (id, folder_id, mailbox_id, uid, message_id, in_reply_to, subject,
		from_address, to_addresses, cc_addresses, date, size_bytes, has_attachments,
		is_seen, is_answered, is_flagged, is_deleted, is_draft, storage_key, spam_score, is_spam)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`
	_, err := r.db.Exec(ctx, query,
		msg.ID, msg.FolderID, msg.MailboxID, msg.UID, msg.MessageID, msg.InReplyTo, msg.Subject,
		msg.FromAddress, msg.ToAddresses, msg.CCAddresses, msg.Date, msg.SizeBytes, msg.HasAttachments,
		msg.IsSeen, msg.IsAnswered, msg.IsFlagged, msg.IsDeleted, msg.IsDraft, msg.StorageKey, msg.SpamScore, msg.IsSpam)
	return err
}

func (r *MessageRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Message, error) {
	query := `SELECT id, folder_id, mailbox_id, uid, message_id, in_reply_to, subject,
		from_address, to_addresses, cc_addresses, date, size_bytes, has_attachments,
		is_seen, is_answered, is_flagged, is_deleted, is_draft, storage_key, spam_score, is_spam, created_at
		FROM messages WHERE id = $1`
	return r.scan(r.db.QueryRow(ctx, query, id))
}

func (r *MessageRepo) GetByUID(ctx context.Context, folderID uuid.UUID, uid int) (*model.Message, error) {
	query := `SELECT id, folder_id, mailbox_id, uid, message_id, in_reply_to, subject,
		from_address, to_addresses, cc_addresses, date, size_bytes, has_attachments,
		is_seen, is_answered, is_flagged, is_deleted, is_draft, storage_key, spam_score, is_spam, created_at
		FROM messages WHERE folder_id = $1 AND uid = $2`
	return r.scan(r.db.QueryRow(ctx, query, folderID, uid))
}

func (r *MessageRepo) ListByFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]*model.Message, int, error) {
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE folder_id = $1`, folderID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, folder_id, mailbox_id, uid, message_id, in_reply_to, subject,
		from_address, to_addresses, cc_addresses, date, size_bytes, has_attachments,
		is_seen, is_answered, is_flagged, is_deleted, is_draft, storage_key, spam_score, is_spam, created_at
		FROM messages WHERE folder_id = $1 ORDER BY date DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, query, folderID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	msgs, err := r.scanRows(rows)
	return msgs, total, err
}

func (r *MessageRepo) ListByMailbox(ctx context.Context, mailboxID uuid.UUID, opts repository.MessageListOptions) ([]*model.Message, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("m.mailbox_id = $%d", argIdx))
	args = append(args, mailboxID)
	argIdx++

	if opts.FolderName != "" {
		conditions = append(conditions, fmt.Sprintf("f.name = $%d", argIdx))
		args = append(args, opts.FolderName)
		argIdx++
	}

	if opts.Since != nil {
		t, err := time.Parse(time.RFC3339, *opts.Since)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("m.date >= $%d", argIdx))
			args = append(args, t)
			argIdx++
		}
	}

	if opts.UnseenOnly {
		conditions = append(conditions, "m.is_seen = false")
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM messages m JOIN folders f ON m.folder_id = f.id WHERE %s`, where)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT m.id, m.folder_id, m.mailbox_id, m.uid, m.message_id, m.in_reply_to, m.subject,
		m.from_address, m.to_addresses, m.cc_addresses, m.date, m.size_bytes, m.has_attachments,
		m.is_seen, m.is_answered, m.is_flagged, m.is_deleted, m.is_draft, m.storage_key, m.spam_score, m.is_spam, m.created_at
		FROM messages m JOIN folders f ON m.folder_id = f.id WHERE %s ORDER BY m.date DESC`, where)

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	msgs, err := r.scanRows(rows)
	return msgs, total, err
}

func (r *MessageRepo) UpdateFlags(ctx context.Context, id uuid.UUID, flags repository.MessageFlags) error {
	var sets []string
	var args []interface{}
	argIdx := 2
	args = append(args, id)

	if flags.IsSeen != nil {
		sets = append(sets, fmt.Sprintf("is_seen = $%d", argIdx))
		args = append(args, *flags.IsSeen)
		argIdx++
	}
	if flags.IsAnswered != nil {
		sets = append(sets, fmt.Sprintf("is_answered = $%d", argIdx))
		args = append(args, *flags.IsAnswered)
		argIdx++
	}
	if flags.IsFlagged != nil {
		sets = append(sets, fmt.Sprintf("is_flagged = $%d", argIdx))
		args = append(args, *flags.IsFlagged)
		argIdx++
	}
	if flags.IsDeleted != nil {
		sets = append(sets, fmt.Sprintf("is_deleted = $%d", argIdx))
		args = append(args, *flags.IsDeleted)
		argIdx++
	}
	if flags.IsDraft != nil {
		sets = append(sets, fmt.Sprintf("is_draft = $%d", argIdx))
		args = append(args, *flags.IsDraft)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE messages SET %s WHERE id = $1", strings.Join(sets, ", "))
	_, err := r.db.Exec(ctx, query, args...)
	return err
}

func (r *MessageRepo) Move(ctx context.Context, id uuid.UUID, targetFolderID uuid.UUID, newUID int) error {
	query := `UPDATE messages SET folder_id = $2, uid = $3 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, targetFolderID, newUID)
	return err
}

func (r *MessageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM messages WHERE id = $1`, id)
	return err
}

func (r *MessageRepo) scan(row pgx.Row) (*model.Message, error) {
	m := &model.Message{}
	err := row.Scan(&m.ID, &m.FolderID, &m.MailboxID, &m.UID, &m.MessageID, &m.InReplyTo, &m.Subject,
		&m.FromAddress, &m.ToAddresses, &m.CCAddresses, &m.Date, &m.SizeBytes, &m.HasAttachments,
		&m.IsSeen, &m.IsAnswered, &m.IsFlagged, &m.IsDeleted, &m.IsDraft, &m.StorageKey, &m.SpamScore, &m.IsSpam, &m.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	return m, err
}

func (r *MessageRepo) scanRows(rows pgx.Rows) ([]*model.Message, error) {
	var msgs []*model.Message
	for rows.Next() {
		m := &model.Message{}
		if err := rows.Scan(&m.ID, &m.FolderID, &m.MailboxID, &m.UID, &m.MessageID, &m.InReplyTo, &m.Subject,
			&m.FromAddress, &m.ToAddresses, &m.CCAddresses, &m.Date, &m.SizeBytes, &m.HasAttachments,
			&m.IsSeen, &m.IsAnswered, &m.IsFlagged, &m.IsDeleted, &m.IsDraft, &m.StorageKey, &m.SpamScore, &m.IsSpam, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
