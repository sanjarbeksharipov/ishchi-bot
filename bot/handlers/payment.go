package handlers

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/helper"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// HandlePaymentSubmission handles when user sends payment receipt photo
func (h *Handler) HandlePaymentSubmission(c tele.Context) error {
	// Get photo
	photo := c.Message().Photo
	if photo == nil {
		return c.Send("❌ Iltimos, to'lov chekini rasm sifatida yuboring.")
	}

	// Check if user has a pending booking (SLOT_RESERVED status)
	// We need to find their most recent reserved booking
	// For now, we'll need user to specify which booking or store it in session
	// Let's check if there's a caption with booking ID or find latest reserved booking

	// TODO: Implement logic to find user's pending booking
	// For now, let's assume we can find it by user_id and status

	return c.Send("📸 To'lov cheki qabul qilindi. Iltimos, qaysi ish uchun to'lov qilganingizni belgilang.")
}

// ForwardPaymentToAdminGroup forwards payment receipt to admin group with approval buttons
func (h *Handler) ForwardPaymentToAdminGroup(ctx context.Context, booking *models.JobBooking, receiptFileID string) error {
	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for admin notification", logger.Error(err))
		return err
	}

	// Get user details from registered_users
	registeredUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, booking.UserID)
	if err != nil {
		h.log.Error("Failed to get registered user", logger.Error(err))
		return err
	}

	// Get telegram user info
	telegramUser, err := h.storage.User().GetByID(ctx, booking.UserID)
	if err != nil {
		h.log.Error("Failed to get telegram user", logger.Error(err))
		return err
	}

	// Format message for admin group
	message := fmt.Sprintf(`🆕 <b>YANGI TO'LOV CHEKI</b>

👤 <b>Foydalanuvchi:</b>
• Ism: %s
• Telefon: %s
• Telegram: @%s (ID: <code>%d</code>)
• Yosh: %d
• Vazn: %d kg
• Bo'y: %d sm

💼 <b>Ish ma'lumotlari:</b>
• Tartib raqami: #%d
• Ish haqqi: %s
• Ish kuni: %s
• Vaqt: %s
• Manzil: %s
• Ovqat: %s
• Xizmat haqqi: %s so'm

📋 <b>Booking ID:</b> #%d
⏰ <b>Yuborilgan vaqt:</b> %s

👇 <b>To'lov cheki:</b>`,
		registeredUser.FullName,
		registeredUser.Phone,
		telegramUser.Username,
		booking.UserID,
		registeredUser.Age,
		registeredUser.Weight,
		registeredUser.Height,
		job.OrderNumber,
		job.Salary,
		job.WorkDate,
		job.WorkTime,
		job.Address,
		job.Food,
		helper.FormatMoney(job.ServiceFee),
		booking.ID,
		time.Now().Add(time.Hour*5).Format("02.01.2006 15:04"),
	)

	// Create photo message
	photo := &tele.Photo{
		File: tele.File{
			FileID: receiptFileID,
		},
		Caption: message,
	}

	// Create inline keyboard with approval buttons
	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("✅ Tasdiqlash", fmt.Sprintf("approve_payment_%d", booking.ID)),
			keyboard.Data("❌ Rad etish", fmt.Sprintf("reject_payment_%d", booking.ID)),
		),
		keyboard.Row(
			keyboard.Data("🚫 Foydalanuvchini bloklash", fmt.Sprintf("block_user_%d_%d", booking.UserID, booking.ID)),
		),
	)

	// Send to admin group
	_, err = h.bot.Send(
		&tele.Chat{ID: h.cfg.Bot.AdminGroupID},
		photo,
		keyboard,
		tele.ModeHTML,
	)

	if err != nil {
		h.log.Error("Failed to send payment to admin group", logger.Error(err))
		return fmt.Errorf("failed to send to admin group: %w", err)
	}

	h.log.Info("Payment receipt forwarded to admin group",
		logger.Any("booking_id", booking.ID),
		logger.Any("user_id", booking.UserID),
	)

	return nil
}

// HandleApprovePayment handles admin approval of payment
func (h *Handler) HandleApprovePayment(c tele.Context, params string) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Get booking ID from callback data (format: approve_payment_bookingID)
	bookingID, err := strconv.ParseInt(params, 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri booking ID.", ShowAlert: true})
	}

	// Approve payment through service
	booking, err := h.services.Payment().ApprovePayment(ctx, bookingID, c.Sender().ID)
	if err != nil {
		h.log.Error("Failed to approve payment", logger.Error(err))

		if err.Error() == "booking not found" {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Booking topilmadi.",
				ShowAlert: true,
			})
		}
		if strings.HasPrefix(err.Error(), "payment already processed") {
			return c.Respond(&tele.CallbackResponse{
				Text:      fmt.Sprintf("⚠️ Bu to'lov allaqachon qayta ishlangan: %s", booking.Status),
				ShowAlert: true,
			})
		}

		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user
	go h.notifyUserPaymentApproved(booking)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n✅ <b>TASDIQLANDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s",
		adminUsername,
		time.Now().Add(time.Hour*5).Format("02.01.2006 15:04"),
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err))
	}

	return c.Respond(&tele.CallbackResponse{
		Text: "✅ To'lov tasdiqlandi!",
	})
}

// HandleRejectPayment handles admin rejection of payment
func (h *Handler) HandleRejectPayment(c tele.Context, params string) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Get booking ID from callback data (format: reject_payment_bookingID)
	bookingID, err := strconv.ParseInt(params, 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Noto'g'ri booking ID.",
			ShowAlert: true,
		})
	}

	// Reject payment through service
	reason := "To'lov cheki noto'g'ri yoki aniq emas"
	booking, err := h.services.Payment().RejectPayment(ctx, bookingID, c.Sender().ID, reason)
	if err != nil {
		h.log.Error("Failed to reject payment", logger.Error(err))

		if err.Error() == "booking not found" {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Booking topilmadi.",
				ShowAlert: true,
			})
		}
		if strings.HasPrefix(err.Error(), "payment already processed") {
			return c.Respond(&tele.CallbackResponse{
				Text:      fmt.Sprintf("⚠️ Bu to'lov allaqachon qayta ishlangan: %s", booking.Status),
				ShowAlert: true,
			})
		}

		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user
	go h.notifyUserPaymentRejected(booking)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n❌ <b>RAD ETILDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s\n💬 Sabab: %s",
		adminUsername,
		time.Now().Add(time.Hour*5).Format("02.01.2006 15:04"),
		booking.RejectionReason,
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err), logger.Any("message", updatedCaption))
	}

	return c.Respond(&tele.CallbackResponse{
		Text: "❌ To'lov rad etildi.",
	})
}

// HandleBlockUser handles blocking a user
func (h *Handler) HandleBlockUser(c tele.Context, params string) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}
	// Get booking ID,user ID from callback data : block_user_userID_bookingID
	callbackDataSl := strings.Split(params, "_")
	if len(callbackDataSl) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri booking ID.", ShowAlert: true})
	}

	userID, err := strconv.ParseInt(callbackDataSl[0], 10, 64)
	if err != nil {
		h.log.Error("Failed to parse user ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri user ID.", ShowAlert: true})
	}

	bookingID, err := strconv.ParseInt(callbackDataSl[1], 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri booking ID.", ShowAlert: true})
	}

	// Block user and reject payment through service
	booking, err := h.services.Payment().BlockUserAndRejectPayment(ctx, bookingID, userID, c.Sender().ID)
	if err != nil {
		h.log.Error("Failed to block user", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Get violation count to determine notification type
	violationCount, err := h.storage.User().GetViolationCount(ctx, nil, userID)
	if err != nil {
		h.log.Error("Failed to get violation count", logger.Error(err))
		violationCount = 0 // fallback
	}
	// Get job
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for violation notification", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user based on violation count
	go h.notifyUserViolation(userID, int64(job.OrderNumber), violationCount)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n🚫 <b>FOYDALANUVCHI BLOKLANDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s",
		adminUsername,
		time.Now().Add(time.Hour*5).Format("02.01.2006 15:04"),
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err))
	}

	h.log.Warn("User blocked by admin",
		logger.Any("user_id", userID),
		logger.Any("admin_id", c.Sender().ID),
		logger.Any("booking_id", bookingID),
	)

	return c.Respond(&tele.CallbackResponse{
		Text: "🚫 Foydalanuvchi bloklandi.",
	})
}

// notifyUserPaymentApproved sends notification to user about approved payment
func (h *Handler) notifyUserPaymentApproved(booking *models.JobBooking) {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for notification", logger.Error(err))
		return
	}

	// Build full job details
	var sb strings.Builder
	sb.WriteString("✅ <b>TO'LOVINGIZ TASDIQLANDI!</b>\n\n")
	sb.WriteString("🎉 Tabriklaymiz! Sizning to'lovingiz admin tomonidan tasdiqlandi.\n\n")
	sb.WriteString("💼 <b>ISH MA'LUMOTLARI:</b>\n")
	fmt.Fprintf(&sb, "📋 Tartib raqami: #%d\n", job.OrderNumber)
	fmt.Fprintf(&sb, "📅 Ish kuni: %s\n", job.WorkDate)
	fmt.Fprintf(&sb, "💰 Ish haqqi: %s\n", job.Salary)
	fmt.Fprintf(&sb, "⏰ Ish vaqti: %s\n", job.WorkTime)
	fmt.Fprintf(&sb, "📍 Manzil: %s\n", job.Address)

	if job.Food != "" {
		fmt.Fprintf(&sb, "🍛 Ovqat: %s\n", job.Food)
	} else {
		sb.WriteString("🍛 Ovqat: Berilmaydi\n")
	}

	if job.Buses != "" {
		fmt.Fprintf(&sb, "🚌 Avtobuslar: %s\n", job.Buses)
	}

	fmt.Fprintf(&sb, "💳 Xizmat haqi: %s so'm\n", helper.FormatMoney(job.ServiceFee))

	if job.AdditionalInfo != "" {
		fmt.Fprintf(&sb, "📝 Qo'shimcha: %s\n", job.AdditionalInfo)
	}

	sb.WriteString("\n� <b>ISH BERUVCHI MA'LUMOTLARI:</b>\n")
	if job.EmployerPhone != "" {
		fmt.Fprintf(&sb, "📱 Telefon: <code>%s</code>\n", job.EmployerPhone)
		sb.WriteString("(Zararuri savollar uchun ish beruvchi bilan bog'laning)\n")
	}

	sb.WriteString("\n�📋 <b>KEYINGI QADAMLAR:</b>\n")
	sb.WriteString("1️⃣ Ishga tayyor bo'ling\n")
	sb.WriteString("2️⃣ Belgilangan vaqtda kelib turing\n")
	sb.WriteString("3️⃣ Ish haqqi ish tugagandan keyin to'lanadi\n\n")
	sb.WriteString("✨ Omad tilaymiz!")

	message := sb.String()

	_, err = h.bot.Send(&tele.User{ID: booking.UserID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify user", logger.Error(err))
	}

	// Send location as a separate message if available
	if job.Location != "" {
		// Parse location string (format: "lat,lng")
		parts := strings.Split(job.Location, ",")
		if len(parts) == 2 {
			lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)

			if err1 == nil && err2 == nil {
				location := &tele.Location{
					Lat: float32(lat),
					Lng: float32(lng),
				}

				_, err = h.bot.Send(&tele.User{ID: booking.UserID}, location)
				if err != nil {
					h.log.Error("Failed to send location", logger.Error(err))
				} else {
					// Send explanation message after location
					_, err = h.bot.Send(&tele.User{ID: booking.UserID}, "📌 <b>Ishga borish uchun aniq manzil yuqorida ko'rsatilgan</b>", tele.ModeHTML)
					if err != nil {
						h.log.Error("Failed to send location explanation", logger.Error(err))
					}
				}
			}
		}
	}
}

// notifyUserPaymentRejected sends notification to user about rejected payment
func (h *Handler) notifyUserPaymentRejected(booking *models.JobBooking) {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for notification", logger.Error(err))
		return
	}

	message := fmt.Sprintf(`❌ <b>TO'LOV RAD ETILDI</b>

Afsuski, sizning to'lov chekingiz admin tomonidan rad etildi.

💼 <b>Ish:</b> №%d
💬 <b>Sabab:</b> %s

📝 <b>Nima qilish kerak:</b>
1️⃣ To'lov chekini tekshiring
2️⃣ Agar to'lov qilgan bo'lsangiz, aniq va to'liq rasm yuboring
3️⃣ Agar to'lov qilmagan bo'lsangiz, qaytadan to'lov qiling va chekni yuboring

💡 <b>Maslahat:</b>
• Chek aniq va o'qilishi kerak
• Summa to'g'ri ko'rsatilgan bo'lishi kerak
• Sana bugungi kunni ko'rsatishi kerak

Agar joylar to'lgan bo'lsa, keyingi ishlar e'lon qilinishini kuting.`,
		job.OrderNumber,
		booking.RejectionReason,
	)

	_, err = h.bot.Send(&tele.User{ID: booking.UserID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify user", logger.Error(err))
	}
}

// notifyUserViolation sends progressive violation notifications
func (h *Handler) notifyUserViolation(userID, jobID int64, violationCount int) {
	var message string

	switch violationCount {
	case 1:
		// First strike - warning
		message = fmt.Sprintf(`⚠️ <b>OGOHLANTIRISH</b>

Sizning to'lov kvitansiyangiz №%d ish uchun soxta yoki noto'g'ri deb topildi.

❗️ <b>Muhim:</b>
• Faqat haqiqiy to'lov chekini yuboring
• To'lov cheki aniq va to'liq bo'lishi kerak
• To'lov summasi va sanasi to'g'ri bo'lishi kerak

⚠️ <b>Ogohlantirish:</b>
Bu sizning birinchi ogohlantirishingiz.

Yana 1 marta soxta to'lov yuborilsa - 24 soat bloklanasiz.
Yana 2 marta soxta to'lov yuborilsa - doimiy bloklanasiz!

📞 Savol bo'lsa admin bilan bog'laning.`,
			jobID,
		)
	case 2:
		// Second strike - 24h block
		message = fmt.Sprintf(`🚫 <b>24 SOAT BLOKLANGANSIZ</b>

Sizning to'lov kvitansiyangiz №%d ish uchun ikkinchi marta soxta deb topildi.

⏰ <b>Bloklash muddati:</b> 24 soat

❌ Siz 24 soat davomida:
• Ish bron qila olmaysiz
• To'lov yubora olmaysiz
• Ishlar ro'yxatini ko'rishingiz mumkin

⚠️ <b>OXIRGI OGOHLANTIRISH:</b>
Yana 1 marta soxta to'lov yuborilsa, doimiy bloklanasiz va endi ish bandlash imkoniyatiga ega bo'lmaysiz!

⏳ 24 soatdan keyin qaytadan urinib ko'rishingiz mumkin.`,
			jobID,
		)
	default:
		// Third strike - permanent block
		message = fmt.Sprintf(`🚫 <b>DOIMIY BLOKLANGANSIZ</b>

Sizning to'lov kvitansiyangiz №%d ish uchun uchinchi marta soxta deb topildi.

❌ <b>Hisobingiz doimiy bloklandi.</b>

Siz endi:
• Ish bron qila olmaysiz
• To'lov yubora olmaysiz
• Tizimdan foydalana olmaysiz

3 marta soxta to'lov kvitansiyasi yuborish tizimdan doimiy chiqarilishga olib keladi.

📞 <b>Apellyatsiya:</b>
Agar bu xato deb hisoblasangiz, admin bilan bog'laning.
Ammo soxta to'lov aniq isbot bo'lsa, bloklash olib tashlanmaydi.`,
			jobID,
		)
	}

	_, err := h.bot.Send(&tele.User{ID: userID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify user about violation", logger.Error(err))
	}
}

// notifyUserBlocked sends notification to blocked user (legacy, kept for backward compatibility)
func (h *Handler) notifyUserBlocked(userID int64) {
	message := `🚫 <b>SIZNING HISOBINGIZ BLOKLANDI</b>

Afsuski, qoidabuzarlik sababli sizning hisobingiz bloklandi.

❌ Siz endi ish bandlash imkoniyatiga ega emassiz.

📞 Agar bu xato deb hisoblasangiz, admin bilan bog'laning.`

	_, err := h.bot.Send(&tele.User{ID: userID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify blocked user", logger.Error(err))
	}
}

// isAdmin checks if user is admin
func (h *Handler) isAdmin(userID int64) bool {
	return slices.Contains(h.cfg.Bot.AdminIDs, userID)
}
