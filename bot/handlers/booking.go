package handlers

import (
	"context"
	"fmt"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// HandleJobBookingStart starts the job booking flow for a registered user
func (h *Handler) HandleJobBookingStart(c tele.Context, user *models.User, jobID int64) error {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send("❌ Ish topilmadi.")
	}

	// Check if job is still accepting bookings
	if job.Status != models.JobStatusActive {
		return c.Send("❌ Bu ish endi faol emas.")
	}

	// Check if job is full
	if job.IsFull() {
		// Check if there are reserved slots that might expire
		if job.ReservedSlots > 0 {
			msg := fmt.Sprintf(`
⏳ <b>Hozirda barcha joylar band</b>

📊 Holat:
• Jami joylar: %d
• Band qilingan: %d (3 daqiqa vaqt bor)
• Tasdiqlangan: %d

💡 <b>Maslahat:</b>
Ba'zi foydalanuvchilar to'lov qilmasalar, 3 daqiqadan so'ng joylar bo'shab qoladi. Iltimos, biroz kutib qaytadan urinib ko'ring!

⏰ Bir necha daqiqadan so'ng qaytadan tekshiring.
`, job.RequiredWorkers, job.ReservedSlots, job.ConfirmedSlots)
			return c.Send(msg, tele.ModeHTML)
		}
		return c.Send("❌ Bu ishga barcha joylar band.")
	}

	// Check if user already has a booking for this job
	// TODO: Add check for existing booking

	// Show job details with booking confirmation
	msg := fmt.Sprintf(`
📋 <b>ISH HAQIDA MA'LUMOT</b>

💰 <b>Ish haqqi:</b> %s
🍛 <b>Ovqat:</b> %s
⏰ <b>Vaqt:</b> %s
📍 <b>Manzil:</b> %s
🌟 <b>Xizmat haqi:</b> %d so'm
📅 <b>Ish kuni:</b> %s

👥 <b>Bo'sh joylar:</b> %d/%d

Ishga yozilishni tasdiqlaysizmi?
`,
		job.Salary,
		valueOrDefault(job.Food, "ko'rsatilmagan"),
		job.WorkTime,
		job.Address,
		job.ServiceFee,
		job.WorkDate,
		job.AvailableSlots(),
		job.RequiredWorkers,
	)

	// Create confirmation keyboard
	menu := &tele.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Ha, yozilaman", fmt.Sprintf("book_confirm_%d", jobID))
	btnCancel := menu.Data("❌ Yo'q, bekor qilish", "book_cancel")
	menu.Inline(
		menu.Row(btnConfirm),
		menu.Row(btnCancel),
	)

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleRegistrationStartWithJob starts registration flow while saving the target job ID
func (h *Handler) HandleRegistrationStartWithJob(c tele.Context, jobID int64) error {
	ctx := context.Background()

	// Get job to show what they're signing up for
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send("❌ Ish topilmadi.")
	}

	msg := fmt.Sprintf(`
👋 Salom!

Siz <b>№%d</b> raqamli ishga yozilmoqchisiz.

Avval ro'yxatdan o'tishingiz kerak. Ro'yxatdan o'tish bir necha daqiqani oladi.

Ro'yxatdan o'tgandan so'ng, ishga yozilish jarayonini davom ettirishingiz mumkin bo'ladi.

<b>Ish haqida qisqacha:</b>
💰 %s
📅 %s
📍 %s

Davom etamizmi?
`,
		job.OrderNumber,
		job.Salary,
		job.WorkDate,
		job.Address,
	)

	menu := &tele.ReplyMarkup{}
	btnStart := menu.Data("✅ Ro'yxatdan o'tish", fmt.Sprintf("start_reg_job_%d", jobID))
	btnCancel := menu.Data("❌ Bekor qilish", "book_cancel")
	menu.Inline(
		menu.Row(btnStart),
		menu.Row(btnCancel),
	)

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleStartRegistrationForJob starts the registration process and saves the job ID
func (h *Handler) HandleStartRegistrationForJob(c tele.Context, jobID int64) error {
	// TODO: Save job ID in session/storage to return to it after registration
	// For now, just start regular registration
	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return h.HandleAcceptOffer(c)
}

// HandleBookingConfirm handles the booking confirmation with atomic slot reservation
func (h *Handler) HandleBookingConfirm(c tele.Context, jobID int64) error {
	ctx := context.Background()
	userID := c.Sender().ID

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	// Check if user already has an active booking for this job
	idempotencyKey := models.GenerateIdempotencyKey(userID, jobID)
	existingBooking, _ := h.storage.Booking().GetByIdempotencyKey(ctx, nil, idempotencyKey)
	if existingBooking != nil {
		if existingBooking.Status == models.BookingStatusSlotReserved && !existingBooking.IsExpired() {
			// User already has a reservation, show remaining time
			remaining := existingBooking.TimeRemaining()
			minutes := int(remaining.Minutes())
			seconds := int(remaining.Seconds()) % 60
			return c.Edit(fmt.Sprintf("⚠️ Siz allaqachon bu ishga yozilgansiz!\n\nQolgan vaqt: %d daqiqa %d soniya", minutes, seconds))
		}
		if existingBooking.Status == models.BookingStatusPaymentSubmitted {
			return c.Edit("⚠️ Sizning to'lovingiz ko'rib chiqilmoqda. Iltimos, admin javobini kuting.")
		}
		if existingBooking.Status == models.BookingStatusConfirmed {
			return c.Edit("✅ Siz allaqachon tasdiqlangansiz!")
		}
	}

	// Start SERIALIZABLE transaction for atomic booking
	tx, err := h.storage.Transaction().Begin(ctx)
	if err != nil {
		h.log.Error("Failed to begin transaction", logger.Error(err))
		return c.Edit("❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring.")
	}

	// Rollback helper
	rollback := func() {
		if rbErr := h.storage.Transaction().Rollback(ctx, tx); rbErr != nil {
			h.log.Error("Failed to rollback transaction", logger.Error(rbErr))
		}
	}

	// Lock job row and get current state
	job, err := h.storage.Job().GetByIDForUpdate(ctx, tx, jobID)
	if err != nil {
		rollback()
		h.log.Error("Failed to lock job", logger.Error(err))
		return c.Edit("❌ Xatolik yuz berdi.")
	}

	// Check if job is still active
	if job.Status != models.JobStatusActive {
		rollback()
		return c.Edit("❌ Bu ish endi faol emas.")
	}

	// Check if slots are available (database will also enforce with CHECK constraint)
	if job.IsFull() {
		rollback()
		// Check if there are reserved slots that might expire
		if job.ReservedSlots > 0 {
			msg := fmt.Sprintf(`
⏳ <b>Hozirda barcha joylar band</b>

📊 Holat:
• Jami joylar: %d
• Band qilingan: %d (3 daqiqa vaqt bor)
• Tasdiqlangan: %d

💡 <b>Maslahat:</b>
Ba'zi foydalanuvchilar to'lov qilmasalar, 3 daqiqadan so'ng joylar bo'shab qoladi. Iltimos, biroz kutib qaytadan urinib ko'ring!

⏰ Bir necha daqiqadan so'ng qaytadan tekshiring.
`, job.RequiredWorkers, job.ReservedSlots, job.ConfirmedSlots)
			return c.Edit(msg, tele.ModeHTML)
		}
		return c.Edit("❌ Kechirasiz, barcha joylar band bo'lib qoldi! 😔")
	}

	// Atomically increment reserved_slots (includes validation)
	if err := h.storage.Job().IncrementReservedSlots(ctx, tx, jobID); err != nil {
		rollback()
		h.log.Error("Failed to increment reserved slots", logger.Error(err))
		return c.Edit("❌ Kechirasiz, barcha joylar band bo'lib qoldi! 😔")
	}

	// Create booking record with 3-minute expiry
	now := time.Now()
	booking := &models.JobBooking{
		JobID:          jobID,
		UserID:         userID,
		Status:         models.BookingStatusSlotReserved,
		ReservedAt:     now,
		ExpiresAt:      now.Add(3 * time.Minute),
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := h.storage.Booking().Create(ctx, tx, booking); err != nil {
		rollback()
		h.log.Error("Failed to create booking", logger.Error(err))
		return c.Edit("❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring.")
	}

	// Commit transaction
	if err := h.storage.Transaction().Commit(ctx, tx); err != nil {
		h.log.Error("Failed to commit transaction", logger.Error(err))
		return c.Edit("❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring.")
	}

	// Success! Send payment instructions
	msg := fmt.Sprintf(`
✅ <b>JOY BAND QILINDI!</b>

Sizga 3 daqiqa vaqt berildi. Iltimos, quyidagi ma'lumotlarga to'lovni amalga oshiring va to'lov chekini yuboring.

<b>To'lov ma'lumotlari:</b>
💳 Karta: 8600 1234 5678 9012
👤 Ism: ADMIN NAME

<b>To'lov summasi:</b> %d so'm (Xizmat haqi)

⏰ Vaqt: 3 daqiqa

To'lov chekini yuboring (screenshot):
`, job.ServiceFee)

	// Edit the message and get the message object to store its ID
	if err := c.Edit(msg, tele.ModeHTML); err != nil {
		h.log.Error("Failed to edit message", logger.Error(err))
		return c.Send(msg, tele.ModeHTML)
	}

	// Store the callback message ID in the booking for later deletion/editing
	// The message ID is from the callback query
	if c.Callback() != nil && c.Callback().Message != nil {
		messageID := int64(c.Callback().Message.ID)
		// Update booking with message ID in a separate transaction (non-critical)
		go func() {
			updateCtx := context.Background()
			tx, err := h.storage.Transaction().Begin(updateCtx)
			if err != nil {
				return
			}

			booking.PaymentInstructionMsgID = messageID
			if err := h.storage.Booking().Update(updateCtx, tx, booking); err != nil {
				h.storage.Transaction().Rollback(updateCtx, tx)
				return
			}
			h.storage.Transaction().Commit(updateCtx, tx)
		}()
	}

	return nil
}

// valueOrDefault returns the value if not empty, otherwise returns the default
func valueOrDefault(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}
