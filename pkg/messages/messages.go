package messages

import (
	"fmt"
	"strings"

	"telegram-bot-starter/bot/models"
)

// Common bot messages
const (
	MsgWelcome = `👋 Welcome to the Bot!

I'm here to help you. Use /help to see available commands.`

	MsgHelp = `📖 Available Commands:

/start - Start the bot
/help - Show this help message
/about - About this bot
/settings - Bot settings
/admin - Admin panel (only for admins)

Feel free to send me any message!`

	MsgAbout = `ℹ️ About This Bot

This is a Telegram bot built with Go and Telebot.
Clean architecture ensures maintainability and scalability.

Version: 1.0.0`

	MsgSettings = `⚙️ Settings

Configure your preferences here.
(Settings coming soon!)`

	MsgUnknownCommand = "❓ Unknown command. Type /help to see available commands."

	MsgError = "⚠️ Something went wrong. Please try again later."

	// Admin messages
	MsgAdminPanel = `👨‍💼 Admin Panel

Ishlarni boshqarish uchun quyidagi tugmalardan foydalaning:`

	// Job creation prompts
	MsgEnterIshHaqqi         = "💰 Ish haqqini kiriting:\n\nMasalan: Soatiga 20 000 so'm"
	MsgEnterOvqat            = "🍛 Ovqat haqida ma'lumot kiriting:\n\nMasalan: Tushlik bilan yoki kiritilmagan"
	MsgEnterVaqt             = "⏰ Ish vaqtini kiriting:\n\nMasalan: 10:30 dan - kamida 5/6 soat ish"
	MsgEnterManzil           = "📍 Manzilni kiriting:\n\nMasalan: Yunusobod Amir Temur xiyoboniga yaqin"
	MsgEnterXizmatHaqqi      = "🌟 Xizmat haqqini kiriting (faqat raqam):\n\nMasalan: 9990"
	MsgEnterAvtobuslar       = "🚌 Avtobuslar haqida ma'lumot kiriting:\n\nMasalan: 45, 67, 89 avtobuslar"
	MsgEnterQoshimcha        = "📝 Qo'shimcha ma'lumot kiriting:\n\nMasalan: Ish yengil 3-4 soatlik ish"
	MsgEnterIshKuni          = "📅 Ish kunini kiriting:\n\nMasalan: Ertaga yoki 25-yanvar"
	MsgEnterKerakliIshchilar = "👥 Kerakli ishchilar sonini kiriting:\n\nMasalan: 5"

	// Registration messages
	MsgRegistrationWelcome = `👋 Xush kelibsiz!

Ishga yozilish uchun avval ro'yxatdan o'tishingiz kerak.

Quyidagi shartlar bilan tanishib chiqing:`

	MsgRegistrationRequired = `⚠️ Ro'yxatdan o'tish talab qilinadi

Ishga yozilish uchun avval ro'yxatdan o'tishingiz kerak.
Ro'yxatdan o'tish uchun /start buyrug'ini yuboring.`

	MsgRegistrationContinue = `📝 Sizda tugallanmagan ro'yxatdan o'tish jarayoni mavjud.

Davom ettirish yoki qaytadan boshlash uchun tanlang:`

	MsgWelcomeRegistered = `👋 Xush kelibsiz, %s!

Siz muvaffaqiyatli ro'yxatdan o'tgansiz.

Quyidagi imkoniyatlardan foydalanishingiz mumkin:`

	MsgPhoneRequestManualInput = `❌ Iltimos, qo'l bilan yozmang!

Telefon raqamingizni yuborish uchun pastdagi "📞 Telefon raqamni yuborish" tugmasini bosing.

⚠️ Bu raqam orqali ish beruvchi siz bilan bog'lanadi!`

	MsgPublicOfferDeclined = `❌ Siz ofertani qabul qilmadingiz.

Ro'yxatdan o'tish bekor qilindi.

Qayta ro'yxatdan o'tish uchun /start buyrug'ini yuboring.`

	MsgRegistrationCancelled = `❌ Ro'yxatdan o'tish bekor qilindi.

Qayta boshlash uchun /start buyrug'ini yuboring.`

	MsgRegistrationComplete = `🎉 Tabriklaymiz!

Siz muvaffaqiyatli ro'yxatdan o'tdingiz!

Endi siz ishlarni ko'rishingiz va ishga yozilishingiz mumkin.`

	MsgSelectEditField = `✏️ Qaysi ma'lumotni o'zgartirmoqchisiz?

Kerakli tugmani tanlang:`

	MsgEnterFullName = `👤 To'liq ism-familiyangizni kiriting (pasportdagidek):

Masalan: Abdullayev Abdulloh

⚠️ Faqat harflar va bo'sh joy, raqamsiz va emojisiz`

	MsgEnterPhone = `📱 Telefon raqamingizni yuborish uchun pastdagi tugmani bosing.

⚠️ Diqqat: Bu raqam orqali ish beruvchi siz bilan bog'lanadi!`

	MsgEnterAge = `🎂 Yoshingizni kiriting (faqat raqam):

Masalan: 25

⚠️ Yosh 16 dan 65 gacha bo'lishi kerak`

	MsgEnterBodyParams = `📏 Vazningiz (kg) va bo'yingizni (sm) kiriting:

Masalan: 70 175

⚠️ Vazn: 30-200 kg, Bo'y: 100-250 sm`

	MsgEnterPassportPhoto = `📸 Pasport rasmingizni yuboring:

⚠️ Faqat rasm formatida yuboring (fayl emas)`
)

// FormatWelcomeRegistered formats welcome message for registered user
func FormatWelcomeRegistered(fullName string) string {
	return fmt.Sprintf(MsgWelcomeRegistered, fullName)
}

// FormatJobForChannel formats a job for posting to the channel
func FormatJobForChannel(job *models.Job) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("💰 Ish haqqi: %s\n", job.IshHaqqi))

	if job.Ovqat != "" {
		sb.WriteString(fmt.Sprintf("🍛 Ovqat: %s\n", job.Ovqat))
	} else {
		sb.WriteString("🍛 Ovqat: kiritilmagan\n")
	}

	sb.WriteString(fmt.Sprintf("⏰ Vaqt: %s\n", job.Vaqt))
	sb.WriteString(fmt.Sprintf("📍 Manzil: %s\n", job.Manzil))
	sb.WriteString(fmt.Sprintf("🌟 Xizmat haqi: %d so'm\n", job.XizmatHaqqi))

	if job.Avtobuslar != "" {
		sb.WriteString(fmt.Sprintf("🚌 Avtobuslar: %s\n", job.Avtobuslar))
	}

	if job.Qoshimcha != "" {
		sb.WriteString(fmt.Sprintf("📝 Qo'shimcha: %s\n", job.Qoshimcha))
	}

	sb.WriteString("\n")

	// Status
	switch job.Status {
	case models.JobStatusOpen:
		sb.WriteString("🟢 Holat: Ochiq\n")
	case models.JobStatusToldi:
		sb.WriteString("🔴 Holat: To'ldi\n")
	case models.JobStatusClosed:
		sb.WriteString("⚫ Holat: Yopilgan\n")
	}

	sb.WriteString(fmt.Sprintf("👥 Ishchilar: %d/%d\n", job.BandIshchilar, job.KerakliIshchilar))
	sb.WriteString(fmt.Sprintf("📅 %s\n", job.IshKuni))
	sb.WriteString(fmt.Sprintf("№ %d", job.OrderNumber))

	return sb.String()
}

// FormatJobDetailAdmin formats a job for admin detail view
func FormatJobDetailAdmin(job *models.Job) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>№ %d</b>\n\n", job.OrderNumber))
	sb.WriteString(fmt.Sprintf("💰 <b>Ish haqqi:</b> %s\n", job.IshHaqqi))
	sb.WriteString(fmt.Sprintf("🍛 <b>Ovqat:</b> %s\n", valueOrEmpty(job.Ovqat)))
	sb.WriteString(fmt.Sprintf("⏰ <b>Vaqt:</b> %s\n", job.Vaqt))
	sb.WriteString(fmt.Sprintf("📍 <b>Manzil:</b> %s\n", job.Manzil))
	sb.WriteString(fmt.Sprintf("🌟 <b>Xizmat haqqi:</b> %d so'm\n", job.XizmatHaqqi))
	sb.WriteString(fmt.Sprintf("🚌 <b>Avtobuslar:</b> %s\n", valueOrEmpty(job.Avtobuslar)))
	sb.WriteString(fmt.Sprintf("📝 <b>Qo'shimcha:</b> %s\n", valueOrEmpty(job.Qoshimcha)))
	sb.WriteString(fmt.Sprintf("📅 <b>Ish kuni:</b> %s\n", job.IshKuni))
	sb.WriteString(fmt.Sprintf("👥 <b>Ishchilar:</b> %d/%d\n", job.BandIshchilar, job.KerakliIshchilar))
	sb.WriteString(fmt.Sprintf("\n<b>Status:</b> %s\n", job.Status.Display()))

	if job.ChannelMessageID != 0 {
		sb.WriteString("\n✅ <i>Kanalga yuborilgan</i>")
	} else {
		sb.WriteString("\n⚠️ <i>Kanalga yuborilmagan</i>")
	}

	return sb.String()
}

func valueOrEmpty(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
