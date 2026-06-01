package postgres

import (
	"context"
	"fmt"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type jobChannelMessageRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewJobChannelMessageRepo creates a new job channel message repository
func NewJobChannelMessageRepo(db *pgxpool.Pool, log logger.LoggerI) storage.JobChannelMessageRepoI {
	return &jobChannelMessageRepo{db: db, log: log}
}

// Upsert inserts or updates the message ID for a (job, channel) pair.
func (r *jobChannelMessageRepo) Upsert(ctx context.Context, jobID, channelID, messageID int64) error {
	query := `
		INSERT INTO job_channel_messages (job_id, channel_id, message_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (job_id, channel_id)
		DO UPDATE SET message_id = EXCLUDED.message_id
	`
	_, err := r.db.Exec(ctx, query, jobID, channelID, messageID)
	if err != nil {
		r.log.Error("Failed to upsert job channel message",
			logger.Error(err),
			logger.Any("job_id", jobID),
			logger.Any("channel_id", channelID),
		)
		return fmt.Errorf("failed to upsert job channel message: %w", err)
	}
	return nil
}

// GetAllByJobID returns all channel messages for a job.
func (r *jobChannelMessageRepo) GetAllByJobID(ctx context.Context, jobID int64) ([]*models.JobChannelMessage, error) {
	query := `
		SELECT id, job_id, channel_id, message_id, created_at
		FROM job_channel_messages
		WHERE job_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		r.log.Error("Failed to get job channel messages", logger.Error(err), logger.Any("job_id", jobID))
		return nil, fmt.Errorf("failed to get job channel messages: %w", err)
	}
	defer rows.Close()

	var result []*models.JobChannelMessage
	for rows.Next() {
		m := &models.JobChannelMessage{}
		if err := rows.Scan(&m.ID, &m.JobID, &m.ChannelID, &m.MessageID, &m.CreatedAt); err != nil {
			r.log.Error("Failed to scan job channel message", logger.Error(err))
			continue
		}
		result = append(result, m)
	}
	return result, nil
}

// Delete removes the record for a specific (job, channel) pair.
func (r *jobChannelMessageRepo) Delete(ctx context.Context, jobID, channelID int64) error {
	query := `DELETE FROM job_channel_messages WHERE job_id = $1 AND channel_id = $2`
	_, err := r.db.Exec(ctx, query, jobID, channelID)
	if err != nil {
		r.log.Error("Failed to delete job channel message",
			logger.Error(err),
			logger.Any("job_id", jobID),
			logger.Any("channel_id", channelID),
		)
		return fmt.Errorf("failed to delete job channel message: %w", err)
	}
	return nil
}

// DeleteAllByJobID removes all channel message records for a job.
func (r *jobChannelMessageRepo) DeleteAllByJobID(ctx context.Context, jobID int64) error {
	query := `DELETE FROM job_channel_messages WHERE job_id = $1`
	_, err := r.db.Exec(ctx, query, jobID)
	if err != nil {
		r.log.Error("Failed to delete all job channel messages",
			logger.Error(err),
			logger.Any("job_id", jobID),
		)
		return fmt.Errorf("failed to delete all job channel messages: %w", err)
	}
	return nil
}
