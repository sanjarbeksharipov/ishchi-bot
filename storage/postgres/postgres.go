package postgres

import (
	"context"
	"strings"
	"time"

	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implements the storage.StorageI interface
type Store struct {
	db     *pgxpool.Pool
	logger logger.LoggerI
}

// NewPostgres creates a new PostgreSQL storage instance
func NewPostgres(ctx context.Context, cfg *config.Config, log logger.LoggerI) (storage.StorageI, error) {
	dsn := cfg.Database.DSN()
	log.Info("Connecting to database")

	parseConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Error("Error while parsing config: " + err.Error())
		return nil, err
	}

	// Configure connection pool for production use
	maxConns := cfg.Database.MaxConnections
	if maxConns < 20 {
		maxConns = 20 // Minimum for production
	}
	if maxConns > 200 {
		maxConns = 200 // PostgreSQL connection limit consideration
	}

	parseConfig.MaxConns = int32(maxConns)
	parseConfig.MinConns = int32(maxConns / 3)      // Keep 1/3 as minimum for quick response
	parseConfig.MaxConnLifetime = 2 * time.Hour     // Longer lifetime for stability
	parseConfig.MaxConnIdleTime = 30 * time.Minute  // Allow longer idle time
	parseConfig.HealthCheckPeriod = 1 * time.Minute // Regular health checks

	// Connection-level timeouts for reliability
	parseConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, parseConfig)
	if err != nil {
		log.Error("Error while creating pool: " + err.Error())
		return nil, err
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Error("Failed to ping database: " + err.Error())
		return nil, err
	}

	log.Info("Postgres connection established")

	// Run migrations
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		log.Error("Error while creating migration instance: " + err.Error())
		return nil, err
	}

	if err = m.Up(); err != nil {
		if !strings.Contains(err.Error(), "no change") {
			log.Error("Error while running migration up: " + err.Error())
			// checking version of migration
			ver, dir, err := m.Version()
			if err != nil {
				log.Error("Error while getting migration version: " + err.Error())
			}

			// checking isDirty of migration
			if dir {
				log.Warn("Migration is dirty, trying to force to previous version")
				// if migration is dirty, we need to force it to previous version
				if err = m.Force(int(ver - 1)); err != nil {
					log.Error("Error while forcing migration to previous version: " + err.Error())
					return nil, err
				}
			}
			return nil, err
		}
		log.Info("No new migrations to apply")
	} else {
		log.Info("Migrations applied successfully")
	}

	return &Store{
		db:     pool,
		logger: log,
	}, nil
}

// CloseDB closes the database connection pool
func (s *Store) CloseDB() {
	s.db.Close()
}

// User returns the user repository
func (s *Store) User() storage.UserRepoI {
	return NewUserRepo(s.db, s.logger)
}

// Job returns the job repository
func (s *Store) Job() storage.JobRepoI {
	return NewJobRepo(s.db, s.logger)
}

// Registration returns the registration repository
func (s *Store) Registration() storage.RegistrationRepoI {
	return NewRegistrationRepo(s.db, s.logger)
}

// Booking returns the booking repository
func (s *Store) Booking() storage.BookingRepoI {
	return NewBookingRepo(s.db, s.logger)
}

// AdminMessage returns the admin message repository
func (s *Store) AdminMessage() storage.AdminMessageRepoI {
	return NewAdminMessageRepo(s.db, s.logger)
}

// Transaction returns the transaction manager
func (s *Store) Transaction() storage.TransactionI {
	return NewTransactionManager(s.db, s.logger)
}
