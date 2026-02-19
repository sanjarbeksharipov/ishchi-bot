package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type jobRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewJobRepo creates a new job repository
func NewJobRepo(db *pgxpool.Pool, log logger.LoggerI) storage.JobRepoI {
	return &jobRepo{
		db:  db,
		log: log,
	}
}

// Create creates a new job
func (r *jobRepo) Create(ctx context.Context, job *models.Job) (*models.Job, error) {
	query := `
		INSERT INTO jobs (
			order_number, salary, food, work_time, address, location, service_fee, buses,
			additional_info, work_date, status, required_workers, reserved_slots, 
			confirmed_slots, channel_message_id, admin_message_id, created_by_admin_id, employer_phone
		) VALUES (nextval('job_order_number_seq'), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, order_number, created_at, updated_at
	`

	err := r.db.QueryRow(ctx, query,
		job.Salary,
		job.Food,
		job.WorkTime,
		job.Address,
		job.Location,
		job.ServiceFee,
		job.Buses,
		job.AdditionalInfo,
		job.WorkDate,
		job.Status,
		job.RequiredWorkers,
		job.ReservedSlots,
		job.ConfirmedSlots,
		job.ChannelMessageID,
		job.AdminMessageID,
		job.CreatedByAdminID,
		job.EmployerPhone,
	).Scan(&job.ID, &job.OrderNumber, &job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to create job", logger.Error(err))
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	return job, nil
}

// GetByID retrieves a job by ID
func (r *jobRepo) GetByID(ctx context.Context, id int64) (*models.Job, error) {
	query := `
		SELECT id, order_number, salary, food, work_time, address, location, service_fee,
			buses, additional_info, work_date, status, required_workers,
			reserved_slots, confirmed_slots, channel_message_id, admin_message_id,
			created_by_admin_id, employer_phone, created_at, updated_at
		FROM jobs
		WHERE id = $1
	`

	job := &models.Job{}
	var food, buses, additionalInfo, employerPhone, location sql.NullString
	var channelMessageID, adminMessageID sql.NullInt64

	err := r.db.QueryRow(ctx, query, id).Scan(
		&job.ID,
		&job.OrderNumber,
		&job.Salary,
		&food,
		&job.WorkTime,
		&job.Address,
		&location,
		&job.ServiceFee,
		&buses,
		&additionalInfo,
		&job.WorkDate,
		&job.Status,
		&job.RequiredWorkers,
		&job.ReservedSlots,
		&job.ConfirmedSlots,
		&channelMessageID,
		&adminMessageID,
		&job.CreatedByAdminID,
		&employerPhone,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get job", logger.Error(err))
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	// Handle nullable fields
	if food.Valid {
		job.Food = food.String
	}
	if buses.Valid {
		job.Buses = buses.String
	}
	if additionalInfo.Valid {
		job.AdditionalInfo = additionalInfo.String
	}
	if location.Valid {
		job.Location = location.String
	}
	if channelMessageID.Valid {
		job.ChannelMessageID = channelMessageID.Int64
	}
	if adminMessageID.Valid {
		job.AdminMessageID = adminMessageID.Int64
	}
	if employerPhone.Valid {
		job.EmployerPhone = employerPhone.String
	}

	return job, nil
}

// GetByIDForUpdate retrieves a job with row lock (FOR UPDATE)
func (r *jobRepo) GetByIDForUpdate(ctx context.Context, tx any, id int64) (*models.Job, error) {
	query := `
		SELECT id, order_number, salary, food, work_time, address, location, service_fee,
			buses, additional_info, work_date, status, required_workers,
			reserved_slots, confirmed_slots, channel_message_id, admin_message_id,
			created_by_admin_id, employer_phone, created_at, updated_at
		FROM jobs
		WHERE id = $1
		FOR UPDATE
	`

	job := &models.Job{}
	var food, buses, additionalInfo, employerPhone, location sql.NullString
	var channelMessageID, adminMessageID sql.NullInt64

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		err = pgxTx.QueryRow(ctx, query, id).Scan(
			&job.ID, &job.OrderNumber, &job.Salary, &food,
			&job.WorkTime, &job.Address, &location, &job.ServiceFee, &buses,
			&additionalInfo, &job.WorkDate, &job.Status, &job.RequiredWorkers,
			&job.ReservedSlots, &job.ConfirmedSlots, &channelMessageID, &adminMessageID,
			&job.CreatedByAdminID, &employerPhone, &job.CreatedAt, &job.UpdatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, query, id).Scan(
			&job.ID, &job.OrderNumber, &job.Salary, &food,
			&job.WorkTime, &job.Address, &location, &job.ServiceFee, &buses,
			&additionalInfo, &job.WorkDate, &job.Status, &job.RequiredWorkers,
			&job.ReservedSlots, &job.ConfirmedSlots, &channelMessageID, &adminMessageID,
			&job.CreatedByAdminID, &employerPhone, &job.CreatedAt, &job.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get job for update: %w", err)
	}

	// Handle nullable fields
	if food.Valid {
		job.Food = food.String
	}
	if buses.Valid {
		job.Buses = buses.String
	}
	if additionalInfo.Valid {
		job.AdditionalInfo = additionalInfo.String
	}
	if location.Valid {
		job.Location = location.String
	}
	if channelMessageID.Valid {
		job.ChannelMessageID = channelMessageID.Int64
	}
	if adminMessageID.Valid {
		job.AdminMessageID = adminMessageID.Int64
	}
	if employerPhone.Valid {
		job.EmployerPhone = employerPhone.String
	}

	return job, nil
}

// GetAll retrieves all jobs with optional status filter
func (r *jobRepo) GetAll(ctx context.Context, status *models.JobStatus) ([]*models.Job, error) {
	query := `
		SELECT id, order_number, salary, food, work_time, address, location, service_fee,
			buses, additional_info, work_date, status, required_workers,
			reserved_slots, confirmed_slots, channel_message_id, admin_message_id,
			created_by_admin_id, employer_phone, created_at, updated_at
		FROM jobs
	`
	args := []any{}

	if status != nil {
		query += " WHERE status = $1"
		args = append(args, *status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		r.log.Error("Failed to get all jobs", logger.Error(err))
		return nil, fmt.Errorf("failed to get all jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		var food, buses, additionalInfo, employerPhone, location sql.NullString
		var channelMessageID, adminMessageID sql.NullInt64

		err := rows.Scan(
			&job.ID, &job.OrderNumber, &job.Salary, &food,
			&job.WorkTime, &job.Address, &location, &job.ServiceFee, &buses,
			&additionalInfo, &job.WorkDate, &job.Status, &job.RequiredWorkers,
			&job.ReservedSlots, &job.ConfirmedSlots, &channelMessageID, &adminMessageID,
			&job.CreatedByAdminID, &employerPhone, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan job", logger.Error(err))
			continue
		}

		// Handle nullable fields
		if food.Valid {
			job.Food = food.String
		}
		if buses.Valid {
			job.Buses = buses.String
		}
		if additionalInfo.Valid {
			job.AdditionalInfo = additionalInfo.String
		}
		if location.Valid {
			job.Location = location.String
		}
		if channelMessageID.Valid {
			job.ChannelMessageID = channelMessageID.Int64
		}
		if adminMessageID.Valid {
			job.AdminMessageID = adminMessageID.Int64
		}
		if employerPhone.Valid {
			job.EmployerPhone = employerPhone.String
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// Update updates a job
func (r *jobRepo) Update(ctx context.Context, job *models.Job) error {
	query := `
		UPDATE jobs
		SET salary = $2, food = $3, work_time = $4, address = $5, location = $6, service_fee = $7,
			buses = $8, additional_info = $9, work_date = $10, status = $11,
			required_workers = $12, reserved_slots = $13, confirmed_slots = $14,
			channel_message_id = $15, admin_message_id = $16, employer_phone = $17, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		job.ID,
		job.Salary,
		toNullString(job.Food),
		job.WorkTime,
		job.Address,
		toNullString(job.Location),
		job.ServiceFee,
		toNullString(job.Buses),
		toNullString(job.AdditionalInfo),
		job.WorkDate,
		job.Status,
		job.RequiredWorkers,
		job.ReservedSlots,
		job.ConfirmedSlots,
		toNullInt64(job.ChannelMessageID),
		toNullInt64(job.AdminMessageID),
		toNullString(job.EmployerPhone),
	)

	if err != nil {
		r.log.Error("Failed to update job", logger.Error(err))
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// UpdateStatus updates only the job status
func (r *jobRepo) UpdateStatus(ctx context.Context, id int64, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, status)
	if err != nil {
		r.log.Error("Failed to update job status", logger.Error(err))
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// UpdateStatusInTx updates only the job status within a transaction
func (r *jobRepo) UpdateStatusInTx(ctx context.Context, tx any, id int64, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $2, updated_at = NOW() WHERE id = $1`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, id, status)
	} else {
		_, err = r.db.Exec(ctx, query, id, status)
	}

	if err != nil {
		r.log.Error("Failed to update job status in transaction", logger.Error(err))
		return fmt.Errorf("failed to update job status: %w", err)
	}
	return nil
}

// UpdateChannelMessageID updates the channel message ID for a job
func (r *jobRepo) UpdateChannelMessageID(ctx context.Context, id int64, messageID int64) error {
	query := `UPDATE jobs SET channel_message_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, messageID)
	if err != nil {
		r.log.Error("Failed to update channel message ID", logger.Error(err))
		return fmt.Errorf("failed to update channel message ID: %w", err)
	}
	return nil
}

// UpdateAdminMessageID updates the admin message ID for a job
func (r *jobRepo) UpdateAdminMessageID(ctx context.Context, id int64, messageID int64) error {
	query := `UPDATE jobs SET admin_message_id = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id, messageID)
	if err != nil {
		r.log.Error("Failed to update admin message ID", logger.Error(err))
		return fmt.Errorf("failed to update admin message ID: %w", err)
	}
	return nil
}

// Delete deletes a job by ID
func (r *jobRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM jobs WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.log.Error("Failed to delete job", logger.Error(err))
		return fmt.Errorf("failed to delete job: %w", err)
	}
	return nil
}

// IncrementReservedSlots atomically increments reserved_slots with validation
func (r *jobRepo) IncrementReservedSlots(ctx context.Context, tx any, jobID int64) error {
	query := `
		UPDATE jobs
		SET reserved_slots = reserved_slots + 1,
			updated_at = NOW()
		WHERE id = $1
		  AND (reserved_slots + confirmed_slots) < required_workers
	`

	var result pgconn.CommandTag
	var err error

	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		result, err = pgxTx.Exec(ctx, query, jobID)
	} else {
		result, err = r.db.Exec(ctx, query, jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to increment reserved slots: %w", err)
	}

	if result.RowsAffected() == 0 {
		return storage.ErrNotFound // Job full or not found
	}

	return nil
}

// DecrementReservedSlots atomically decrements reserved_slots
func (r *jobRepo) DecrementReservedSlots(ctx context.Context, tx any, jobID int64) error {
	query := `
		UPDATE jobs
		SET reserved_slots = GREATEST(reserved_slots - 1, 0),
			updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, jobID)
	} else {
		_, err = r.db.Exec(ctx, query, jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to decrement reserved slots: %w", err)
	}

	return nil
}

// MoveReservedToConfirmed atomically moves slot from reserved to confirmed
func (r *jobRepo) MoveReservedToConfirmed(ctx context.Context, tx any, jobID int64) error {
	query := `
		UPDATE jobs
		SET reserved_slots = GREATEST(reserved_slots - 1, 0),
			confirmed_slots = confirmed_slots + 1,
			updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, jobID)
	} else {
		_, err = r.db.Exec(ctx, query, jobID)
	}

	if err != nil {
		return fmt.Errorf("failed to move reserved to confirmed: %w", err)
	}

	return nil
}

// GetAvailableSlots returns how many slots are available
func (r *jobRepo) GetAvailableSlots(ctx context.Context, jobID int64) (int, error) {
	query := `
		SELECT required_workers - (reserved_slots + confirmed_slots) as available
		FROM jobs
		WHERE id = $1
	`

	var available int
	err := r.db.QueryRow(ctx, query, jobID).Scan(&available)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, storage.ErrNotFound
		}
		return 0, fmt.Errorf("failed to get available slots: %w", err)
	}

	if available < 0 {
		available = 0
	}

	return available, nil
}

// GetTotalCount returns the total number of jobs
func (r *jobRepo) GetTotalCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs`).Scan(&count)
	if err != nil {
		r.log.Error("Failed to get total job count: " + err.Error())
		return 0, fmt.Errorf("failed to get total job count: %w", err)
	}
	return count, nil
}

// GetCountByStatus returns the number of jobs with a given status
func (r *jobRepo) GetCountByStatus(ctx context.Context, status models.JobStatus) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM jobs WHERE status = $1`, status).Scan(&count)
	if err != nil {
		r.log.Error("Failed to get job count by status: " + err.Error())
		return 0, fmt.Errorf("failed to get job count by status: %w", err)
	}
	return count, nil
}
