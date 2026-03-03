package service

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
)

const (
	// expiryDBTimeout is the max time for any single DB operation in the expiry worker.
	expiryDBTimeout = 10 * time.Second
	// expiryNotifyTimeout is the max time for sending a Telegram notification.
	expiryNotifyTimeout = 15 * time.Second
)

// ExpiryWorker handles automatic expiration of reserved bookings
type ExpiryWorker struct {
	storage  storage.StorageI
	log      logger.LoggerI
	bot      *tele.Bot
	interval time.Duration
	stopChan chan struct{}
}

// NewExpiryWorker creates a new expiry worker
func NewExpiryWorker(storage storage.StorageI, log logger.LoggerI, bot *tele.Bot) *ExpiryWorker {
	return &ExpiryWorker{
		storage:  storage,
		log:      log,
		bot:      bot,
		interval: 10 * time.Second, // Check every 10 seconds
		stopChan: make(chan struct{}),
	}
}

// Start begins the expiry worker background process
func (w *ExpiryWorker) Start() {
	w.log.Info("Expiry worker started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.safeProcessExpiredBookings()

	for {
		select {
		case <-ticker.C:
			w.safeProcessExpiredBookings()
		case <-w.stopChan:
			w.log.Info("Expiry worker stopped")
			return
		}
	}
}

// Stop gracefully stops the expiry worker
func (w *ExpiryWorker) Stop() {
	close(w.stopChan)
}

// safeProcessExpiredBookings wraps processExpiredBookings with panic recovery.
// Without this, an unrecovered panic would crash the entire bot process
// (container stays up, bot stops responding).
func (w *ExpiryWorker) safeProcessExpiredBookings() {
	defer func() {
		if r := recover(); r != nil {
			w.log.Error("PANIC in expiry worker recovered",
				logger.Any("panic", fmt.Sprintf("%v", r)),
				logger.Any("stack", string(debug.Stack())),
			)
		}
	}()
	w.processExpiredBookings()
}

// processExpiredBookings finds and processes all expired bookings
func (w *ExpiryWorker) processExpiredBookings() {
	ctx, cancel := context.WithTimeout(context.Background(), expiryDBTimeout)
	defer cancel()

	expiredBookings, err := w.storage.Booking().GetExpiredBookings(ctx, 100)
	if err != nil {
		w.log.Error("Failed to get expired bookings", logger.Error(err))
		return
	}

	if len(expiredBookings) == 0 {
		return
	}

	w.log.Info("Processing expired bookings", logger.Any("count", len(expiredBookings)))

	// Process each expired booking with its own timeout
	for _, booking := range expiredBookings {
		if err := w.processExpiredBooking(booking); err != nil {
			w.log.Error("Failed to process expired booking",
				logger.Error(err),
				logger.Any("booking_id", booking.ID),
				logger.Any("user_id", booking.UserID),
				logger.Any("job_id", booking.JobID),
			)
			continue
		}

		w.log.Info("Released expired booking",
			logger.Any("booking_id", booking.ID),
			logger.Any("user_id", booking.UserID),
			logger.Any("job_id", booking.JobID),
		)
	}
}

// processExpiredBooking releases a single expired booking
func (w *ExpiryWorker) processExpiredBooking(booking *models.JobBooking) error {
	// Use a dedicated context with timeout for the DB transaction
	ctx, cancel := context.WithTimeout(context.Background(), expiryDBTimeout)
	defer cancel()

	// Start transaction
	tx, err := w.storage.Transaction().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	// Always rollback on failure — calling Rollback after Commit is a harmless no-op in pgx.
	// The previous code had a bug: commit failure skipped rollback, leaking the connection.
	defer func() {
		if rbErr := w.storage.Transaction().Rollback(ctx, tx); rbErr != nil {
			// Ignore "tx is closed" errors (expected after successful commit)
			w.log.Debug("Rollback after processExpiredBooking (expected if committed)",
				logger.Error(rbErr))
		}
	}()

	// Mark booking as expired
	if err := w.storage.Booking().MarkAsExpired(ctx, tx, booking.ID); err != nil {
		return fmt.Errorf("mark expired: %w", err)
	}

	// Release the reserved slot (decrement reserved_slots)
	if err := w.storage.Job().DecrementReservedSlots(ctx, tx, booking.JobID); err != nil {
		return fmt.Errorf("decrement slots: %w", err)
	}

	// Commit transaction
	if err := w.storage.Transaction().Commit(ctx, tx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Notification is best-effort — don't fail the expiry if it doesn't work
	w.notifyUserExpiredSafe(booking)

	return nil
}

// notifyUserExpiredSafe wraps notifyUserExpired with a timeout so a hung
// Telegram API call can't block the worker goroutine forever.
func (w *ExpiryWorker) notifyUserExpiredSafe(booking *models.JobBooking) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				w.log.Error("PANIC in notifyUserExpired recovered",
					logger.Any("panic", fmt.Sprintf("%v", r)),
					logger.Any("booking_id", booking.ID),
				)
			}
		}()
		w.notifyUserExpired(booking)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(expiryNotifyTimeout):
		w.log.Error("Timeout sending expiry notification",
			logger.Any("booking_id", booking.ID),
			logger.Any("user_id", booking.UserID),
		)
	}
}

// notifyUserExpired sends a notification to the user about expired booking
func (w *ExpiryWorker) notifyUserExpired(booking *models.JobBooking) {
	// Get job details for notification
	ctx, cancel := context.WithTimeout(context.Background(), expiryDBTimeout)
	defer cancel()

	job, err := w.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		w.log.Error("Failed to get job for notification", logger.Error(err))
		return
	}

	// Try to delete or edit the original payment instruction message
	if booking.PaymentInstructionMsgID != 0 {
		expiredMsg := fmt.Sprintf(`
⏰ <b>VAQT TUGADI</b>

Sizning band qilgan joyingiz muddati tugadi, chunki 3 daqiqa ichida to'lov qilmadingiz.

📋 <b>Ish:</b> №%d
💰 %s
📅 %s

Yana yozilish uchun kanal orqali ishga qaytadan o'tishingiz mumkin.
`, job.OrderNumber, job.Salary, job.WorkDate)

		msg := &tele.StoredMessage{
			MessageID: strconv.FormatInt(booking.PaymentInstructionMsgID, 10),
			ChatID:    booking.UserID,
		}

		// Try to edit the message
		recipient := &tele.User{ID: booking.UserID}
		if _, err := w.bot.Edit(msg, expiredMsg, tele.ModeHTML); err != nil {
			// If edit fails, try to delete and send new message
			w.log.Error("Failed to edit expiry message, trying delete",
				logger.Error(err),
				logger.Any("user_id", booking.UserID),
			)

			if err := w.bot.Delete(&tele.Message{
				ID:   int(booking.PaymentInstructionMsgID),
				Chat: &tele.Chat{ID: booking.UserID},
			}); err != nil {
				w.log.Error("Failed to delete payment message", logger.Error(err))
			}

			// Send new notification
			w.bot.Send(recipient, expiredMsg, tele.ModeHTML)
		}
	} else {
		// No message ID stored, just send a new notification
		msg := fmt.Sprintf(`
⏰ <b>VAQT TUGADI</b>

Sizning №%d raqamli ishga band qilgan joyingiz muddati tugadi, chunki 3 daqiqa ichida to'lov qilmadingiz.

📋 <b>Ish:</b>
💰 %s
📅 %s

Yana yozilish uchun kanal orqali ishga qaytadan o'tishingiz mumkin.
`, job.OrderNumber, job.Salary, job.WorkDate)

		recipient := &tele.User{ID: booking.UserID}
		if _, err := w.bot.Send(recipient, msg, tele.ModeHTML); err != nil {
			w.log.Error("Failed to send expiry notification",
				logger.Error(err),
				logger.Any("user_id", booking.UserID),
				logger.Any("booking_id", booking.ID),
			)
		}
	}
}
