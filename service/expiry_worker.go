package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
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
	w.processExpiredBookings()

	for {
		select {
		case <-ticker.C:
			w.processExpiredBookings()
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

// processExpiredBookings finds and processes all expired bookings
func (w *ExpiryWorker) processExpiredBookings() {
	ctx := context.Background()

	// Get expired bookings (with FOR UPDATE SKIP LOCKED to prevent conflicts)
	expiredBookings, err := w.storage.Booking().GetExpiredBookings(ctx, 100)
	if err != nil {
		w.log.Error("Failed to get expired bookings", logger.Error(err))
		return
	}

	if len(expiredBookings) == 0 {
		return
	}

	w.log.Info("Processing expired bookings", logger.Any("count", len(expiredBookings)))

	// Process each expired booking
	for _, booking := range expiredBookings {
		if err := w.processExpiredBooking(ctx, booking); err != nil {
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
func (w *ExpiryWorker) processExpiredBooking(ctx context.Context, booking *models.JobBooking) error {
	// Start transaction
	tx, err := w.storage.Transaction().Begin(ctx)
	if err != nil {
		return err
	}

	// Rollback helper
	rollback := func() error {
		if rbErr := w.storage.Transaction().Rollback(ctx, tx); rbErr != nil {
			w.log.Error("Failed to rollback transaction", logger.Error(rbErr))
			return rbErr
		}
		return nil
	}

	// Mark booking as expired
	if err := w.storage.Booking().MarkAsExpired(ctx, tx, booking.ID); err != nil {
		rollback()
		return err
	}

	// Release the reserved slot (decrement reserved_slots)
	if err := w.storage.Job().DecrementReservedSlots(ctx, tx, booking.JobID); err != nil {
		rollback()
		return err
	}

	// Commit transaction
	if err := w.storage.Transaction().Commit(ctx, tx); err != nil {
		return err
	}

	// Get job details for notification
	job, err := w.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		w.log.Error("Failed to get job for notification", logger.Error(err))
		// Don't fail the expiry process if notification fails
		return nil
	}

	// Send notification to user
	w.notifyUserExpired(booking, job)

	return nil
}

// notifyUserExpired sends a notification to the user about expired booking
func (w *ExpiryWorker) notifyUserExpired(booking *models.JobBooking, job *models.Job) {
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
