package postgres

import (
	"context"
	"fmt"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type adminMessageRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewAdminMessageRepo creates a new admin message repository
func NewAdminMessageRepo(db *pgxpool.Pool, log logger.LoggerI) storage.AdminMessageRepoI {
	return &adminMessageRepo{
		db:  db,
		log: log,
	}
}

// Upsert creates or updates an admin message for a job
func (r *adminMessageRepo) Upsert(ctx context.Context, adminMsg *models.AdminJobMessage) error {
	query := `
		INSERT INTO admin_job_messages (job_id, admin_id, message_id, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (job_id, admin_id)
		DO UPDATE SET message_id = $3, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query, adminMsg.JobID, adminMsg.AdminID, adminMsg.MessageID).
		Scan(&adminMsg.ID, &adminMsg.CreatedAt, &adminMsg.UpdatedAt)
	if err != nil {
		r.log.Error("Failed to upsert admin message", logger.Error(err))
		return fmt.Errorf("failed to upsert admin message: %w", err)
	}

	return nil
}

// Get retrieves an admin message by job and admin ID
func (r *adminMessageRepo) Get(ctx context.Context, jobID, adminID int64) (*models.AdminJobMessage, error) {
	query := `
		SELECT id, job_id, admin_id, message_id, created_at, updated_at
		FROM admin_job_messages
		WHERE job_id = $1 AND admin_id = $2
	`

	adminMsg := &models.AdminJobMessage{}
	err := r.db.QueryRow(ctx, query, jobID, adminID).Scan(
		&adminMsg.ID,
		&adminMsg.JobID,
		&adminMsg.AdminID,
		&adminMsg.MessageID,
		&adminMsg.CreatedAt,
		&adminMsg.UpdatedAt,
	)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get admin message", logger.Error(err))
		return nil, fmt.Errorf("failed to get admin message: %w", err)
	}

	return adminMsg, nil
}

// GetAllByJobID retrieves all admin messages for a job
func (r *adminMessageRepo) GetAllByJobID(ctx context.Context, jobID int64) ([]*models.AdminJobMessage, error) {
	query := `
		SELECT id, job_id, admin_id, message_id, created_at, updated_at
		FROM admin_job_messages
		WHERE job_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		r.log.Error("Failed to get admin messages for job", logger.Error(err))
		return nil, fmt.Errorf("failed to get admin messages for job: %w", err)
	}
	defer rows.Close()

	var messages []*models.AdminJobMessage
	for rows.Next() {
		adminMsg := &models.AdminJobMessage{}
		if err := rows.Scan(
			&adminMsg.ID,
			&adminMsg.JobID,
			&adminMsg.AdminID,
			&adminMsg.MessageID,
			&adminMsg.CreatedAt,
			&adminMsg.UpdatedAt,
		); err != nil {
			r.log.Error("Failed to scan admin message", logger.Error(err))
			return nil, fmt.Errorf("failed to scan admin message: %w", err)
		}
		messages = append(messages, adminMsg)
	}

	return messages, nil
}

// Delete deletes an admin message
func (r *adminMessageRepo) Delete(ctx context.Context, jobID, adminID int64) error {
	query := `DELETE FROM admin_job_messages WHERE job_id = $1 AND admin_id = $2`
	_, err := r.db.Exec(ctx, query, jobID, adminID)
	if err != nil {
		r.log.Error("Failed to delete admin message", logger.Error(err))
		return fmt.Errorf("failed to delete admin message: %w", err)
	}
	return nil
}

// DeleteAllByJobID deletes all admin messages for a job
func (r *adminMessageRepo) DeleteAllByJobID(ctx context.Context, jobID int64) error {
	query := `DELETE FROM admin_job_messages WHERE job_id = $1`
	_, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		r.log.Error("Failed to delete all admin messages for job", logger.Error(err))
		return fmt.Errorf("failed to delete all admin messages for job: %w", err)
	}
	return nil
}
