package postgres

import (
	"context"
	"errors"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
)

type jobRepo struct {
	db *Store
}

// NewJobRepo creates a new job repository
func NewJobRepo(db *Store) storage.JobRepoI {
	return &jobRepo{db: db}
}

// Create creates a new job
func (r *jobRepo) Create(ctx context.Context, job *models.Job) error {
	query := `
		INSERT INTO jobs (
			ish_haqqi, ovqat, vaqt, manzil, xizmat_haqqi, avtobuslar,
			qoshimcha, ish_kuni, status, kerakli_ishchilar, band_ishchilar,
			channel_message_id, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, order_number, created_at, updated_at
	`

	err := r.db.db.QueryRow(ctx, query,
		job.IshHaqqi,
		job.Ovqat,
		job.Vaqt,
		job.Manzil,
		job.XizmatHaqqi,
		job.Avtobuslar,
		job.Qoshimcha,
		job.IshKuni,
		job.Status,
		job.KerakliIshchilar,
		job.BandIshchilar,
		job.ChannelMessageID,
		job.CreatedBy,
	).Scan(&job.ID, &job.OrderNumber, &job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		return err
	}

	return nil
}

// GetByID retrieves a job by ID
func (r *jobRepo) GetByID(ctx context.Context, id int64) (*models.Job, error) {
	query := `
		SELECT id, order_number, ish_haqqi, ovqat, vaqt, manzil, xizmat_haqqi,
			avtobuslar, qoshimcha, ish_kuni, status, kerakli_ishchilar,
			band_ishchilar, channel_message_id, created_by, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	job := &models.Job{}
	err := r.db.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.OrderNumber,
		&job.IshHaqqi,
		&job.Ovqat,
		&job.Vaqt,
		&job.Manzil,
		&job.XizmatHaqqi,
		&job.Avtobuslar,
		&job.Qoshimcha,
		&job.IshKuni,
		&job.Status,
		&job.KerakliIshchilar,
		&job.BandIshchilar,
		&job.ChannelMessageID,
		&job.CreatedBy,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, err
	}

	return job, nil
}

// GetAll retrieves all jobs with optional status filter
func (r *jobRepo) GetAll(ctx context.Context, status *models.JobStatus) ([]*models.Job, error) {
	query := `
		SELECT id, order_number, ish_haqqi, ovqat, vaqt, manzil, xizmat_haqqi,
			avtobuslar, qoshimcha, ish_kuni, status, kerakli_ishchilar,
			band_ishchilar, channel_message_id, created_by, created_at, updated_at
		FROM jobs
	`
	args := []interface{}{}

	if status != nil {
		query += " WHERE status = $1"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		err := rows.Scan(
			&job.ID,
			&job.OrderNumber,
			&job.IshHaqqi,
			&job.Ovqat,
			&job.Vaqt,
			&job.Manzil,
			&job.XizmatHaqqi,
			&job.Avtobuslar,
			&job.Qoshimcha,
			&job.IshKuni,
			&job.Status,
			&job.KerakliIshchilar,
			&job.BandIshchilar,
			&job.ChannelMessageID,
			&job.CreatedBy,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Update updates a job
func (r *jobRepo) Update(ctx context.Context, job *models.Job) error {
	query := `
		UPDATE jobs SET
			ish_haqqi = $1,
			ovqat = $2,
			vaqt = $3,
			manzil = $4,
			xizmat_haqqi = $5,
			avtobuslar = $6,
			qoshimcha = $7,
			ish_kuni = $8,
			status = $9,
			kerakli_ishchilar = $10,
			band_ishchilar = $11,
			channel_message_id = $12,
			updated_at = $13
		WHERE id = $14
	`

	job.UpdatedAt = time.Now()
	result, err := r.db.db.Exec(ctx, query,
		job.IshHaqqi,
		job.Ovqat,
		job.Vaqt,
		job.Manzil,
		job.XizmatHaqqi,
		job.Avtobuslar,
		job.Qoshimcha,
		job.IshKuni,
		job.Status,
		job.KerakliIshchilar,
		job.BandIshchilar,
		job.ChannelMessageID,
		job.UpdatedAt,
		job.ID,
	)

	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateStatus updates only the job status
func (r *jobRepo) UpdateStatus(ctx context.Context, id int64, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.db.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateChannelMessageID updates the channel message ID for a job
func (r *jobRepo) UpdateChannelMessageID(ctx context.Context, id int64, messageID int64) error {
	query := `UPDATE jobs SET channel_message_id = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.db.Exec(ctx, query, messageID, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// IncrementBookedWorkers increments the booked workers count
func (r *jobRepo) IncrementBookedWorkers(ctx context.Context, id int64) error {
	query := `
		UPDATE jobs 
		SET band_ishchilar = band_ishchilar + 1, updated_at = NOW() 
		WHERE id = $1 AND band_ishchilar < kerakli_ishchilar
	`

	result, err := r.db.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// Delete deletes a job by ID
func (r *jobRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM jobs WHERE id = $1`

	result, err := r.db.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}
