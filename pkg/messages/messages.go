package messages

import (
	"fmt"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/helper"
)

// Common bot messages
const (
	MsgWelcome = `👋 Welcome to the Bot!

I'm here to help you. Use /help to see available commands.`

	MsgHelp = `📖 <b>YORDAM</b>

<b>Bot haqida:</b>
Bu bot orqali siz kunlik ish topishingiz va ish uchun yozilishingiz mumkin.

<b>Qanday ishlaydi?</b>
1️⃣ Avval ro'yxatdan o'ting
2️⃣ Kanalimizdan ish e'lonlarini ko'ring
3️⃣ Yoqqan ishga yoziling
4️⃣ To'lov qiling
5️⃣ Admin tasdiqlashi kutilsin
6️⃣ Ish ma'lumotlarini oling

<b>Buyruqlar:</b>
/start - Botni ishga tushirish
/help - Yordam

<b>Profil:</b>
Profilingizni ko'rish va tahrirlash uchun "👤 Profil" tugmasini bosing.

❓ Savollar bo'lsa @ArzonBepul bilan bog'laning.`

	MsgAbout = `ℹ️ Bot haqida

Bu bot orqali kunlik ishlarni topish va yozilish mumkin.

Versiya: 1.0.0`

	MsgSettings = `⚙️ Sozlamalar

Sozlamalar tez orada qo'shiladi.`

	MsgUnknownCommand = "❓ Noma'lum buyruq. Mavjud buyruqlarni ko'rish uchun /help ni bosing."

	MsgError = "❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring."

	// Admin messages
	MsgAdminPanel = `👨‍💼 Admin Panel

Ishlarni boshqarish uchun quyidagi tugmalardan foydalaning:`

	// Job creation prompts
	MsgEnterIshHaqqi         = "💰 Ish haqqini kiriting:\n\nMasalan: Soatiga 20 000 so'm"
	MsgEnterOvqat            = "🍛 Ovqat haqida ma'lumot kiriting:\n\nMasalan: Tushlik bilan yoki kiritilmagan"
	MsgEnterVaqt             = "⏰ Ish vaqtini kiriting:\n\nMasalan: 10:30 dan - kamida 5/6 soat ish"
	MsgEnterManzil           = "📍 Manzilni kiriting:\n\nMasalan: Yunusobod Amir Temur xiyoboniga yaqin"
	MsgEnterLocation         = "📌 Aniq joylashuvni yuboring (faqat to'lov tasdiqlangan foydalanuvchilar uchun):\n\n📍 Telegram orqali joylashuvni (location) yuboring.\n\n⚠️ Matnli xabar emas, balki Telegram location funksiyasidan foydalaning."
	MsgEnterXizmatHaqqi      = "🌟 Xizmat haqqini kiriting (faqat raqam):\n\nMasalan: 9990"
	MsgEnterAvtobuslar       = "🚌 Avtobuslar haqida ma'lumot kiriting:\n\nMasalan: 45, 67, 89 avtobuslar"
	MsgEnterIshTavsifi       = "📝 Ish tavsifi va talablarni kiriting:\n\nMasalan: Ish yengil, 3-4 soatlik. Kiyim: Qora kiyim talab qilinadi"
	MsgEnterIshKuni          = "📅 Ish kunini kiriting:\n\nMasalan: Ertaga yoki 25-yanvar"
	MsgEnterKerakliIshchilar = "👥 Kerakli ishchilar sonini kiriting:\n\nMasalan: 5"
	MsgEnterConfirmedSlots   = "✅ Qabul qilingan ishchilar sonini kiriting:\n\nMasalan: 3\n\n⚠️ Qabul qilingan soni kerakli sondan oshmasligi kerak."
	MsgEnterEmployerPhone    = "📞 Ish beruvchining telefon raqamini kiriting:\n\nMasalan: +998901234567 yoki 901234567\n\n⚠️ Bu raqam faqat to'lov tasdiqlangan foydalanuvchilar uchun ko'rinadi."

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

func FormatJobForChannel(job *models.Job) string {
	var sb strings.Builder

	// Header with Order Number
	fmt.Fprintf(&sb, "📋 №%d\n\n", job.OrderNumber)
	// Main Details
	fmt.Fprintf(&sb, "📅Sana: %s\n", job.WorkDate)
	fmt.Fprintf(&sb, "💰Maosh: %s\n", job.Salary)
	fmt.Fprintf(&sb, "⏰Ish vaqti: %s\n", job.WorkTime)

	// Conditional Food Info
	if job.Food == "" {
		fmt.Fprintf(&sb, "🍛Ovqat: Berilmaydi\n")
	} else {
		fmt.Fprintf(&sb, "🍛Ovqat: %s\n", job.Food)
	}

	fmt.Fprintf(&sb, "📍Manzil: %s\n", job.Address)

	// Transport
	if job.Buses != "" {
		fmt.Fprintf(&sb, "🚌Avtobuslar: %s\n", job.Buses)
	}

	// Money matters
	fmt.Fprintf(&sb, "💳Xizmat haqqi: %s so'm\n", helper.FormatMoney(job.ServiceFee))
	if job.AdditionalInfo != "" {
		fmt.Fprintf(&sb, "📝Batafsil: %s \n\n", job.AdditionalInfo)
	}

	// Progress Bar and Status
	statusEmoji := "🟢"
	statusText := "FAOL"
	switch job.Status {
	case models.JobStatusFull:
		statusEmoji = "🔴"
		statusText = "TO'LDI"
	case models.JobStatusCompleted:
		statusEmoji = "⚫"
		statusText = "YOPILGAN"
	}

	// Visual Capacity Bar
	fmt.Fprintf(&sb, "%sHolat: %s\n", statusEmoji, statusText)
	fmt.Fprintf(
		&sb,
		"👥 Ishchilar: %d/%d (Bo‘sh: %d ta)\n",
		job.ConfirmedSlots,
		job.RequiredWorkers,
		job.RequiredWorkers-job.ConfirmedSlots,
	)
	return sb.String()
}

// FormatJobDetailAdmin formats a job for admin detail view
func FormatJobDetailAdmin(job *models.Job) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("<b>№ %d</b>\n\n", job.OrderNumber))
	sb.WriteString(fmt.Sprintf("💰 <b>Ish haqqi:</b> %s\n", job.Salary))
	sb.WriteString(fmt.Sprintf("🍛 <b>Ovqat:</b> %s\n", valueOrEmpty(job.Food)))
	sb.WriteString(fmt.Sprintf("⏰ <b>Vaqt:</b> %s\n", job.WorkTime))
	sb.WriteString(fmt.Sprintf("📍 <b>Manzil:</b> %s\n", job.Address))
	sb.WriteString(fmt.Sprintf("📌 <b>Aniq joylashuv:</b> %s\n", valueOrEmpty(job.Location)))
	sb.WriteString(fmt.Sprintf("🌟 <b>Xizmat haqqi:</b> %s so'm\n", helper.FormatMoney(job.ServiceFee)))
	sb.WriteString(fmt.Sprintf("🚌 <b>Avtobuslar:</b> %s\n", valueOrEmpty(job.Buses)))
	sb.WriteString(fmt.Sprintf("📝 <b>Ish tavsifi:</b> %s\n", valueOrEmpty(job.AdditionalInfo)))
	sb.WriteString(fmt.Sprintf("📅 <b>Ish kuni:</b> %s\n", job.WorkDate))
	sb.WriteString(fmt.Sprintf("👥 <b>Ishchilar:</b> %d/%d\n", job.ConfirmedSlots, job.RequiredWorkers))
	sb.WriteString(fmt.Sprintf("📞 <b>Ish beruvchi telefon:</b> %s\n", valueOrEmpty(job.EmployerPhone)))
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
func FormatNoAvailableSlots(job *models.Job) string {
	msg := fmt.Sprintf(`
⏳ <b>Hozircha bo'sh joylar qolmadi</b>

📊 <b>Bandlik holati:</b>
- Jami o'rinlar: <b>%d</b> ta
- Tasdiqlangan: <b>%d</b> ta
- To'lov kutilmoqda: <b>%d</b> ta
💡 <b>Eslatma:</b>
Ayrim foydalanuvchilar to'lovni o'z vaqtida amalga oshirmasliklari mumkin. Bunday holda, band qilingan joylar <b>3 daqiqa ichida</b> qayta ochiladi.

⏰ Bir necha daqiqadan so'ng qaytadan urinib ko'ring!
`, job.RequiredWorkers, job.ConfirmedSlots, job.ReservedSlots)
	return msg
}

func FormatJobDetailUser(job *models.Job) string {
	msg := fmt.Sprintf(`
<b>ISH HAQIDA MA'LUMOT</b>

📋 <b>№:</b> %d
💰 <b>Ish haqqi:</b> %s
🍛 <b>Ovqat:</b> %s
⏰ <b>Vaqt:</b> %s
📍 <b>Manzil:</b> %s
🌟 <b>Xizmat haqqi:</b> %s so'm
📅 <b>Ish kuni:</b> %s

👥 <b>Bo'sh joylar:</b> %d

Ishga yozilishni tasdiqlaysizmi?
`,
		job.OrderNumber,
		job.Salary,
		helper.ValueOrDefault(job.Food, "ko'rsatilmagan"),
		job.WorkTime,
		job.Address,
		helper.FormatMoney(job.ServiceFee),
		job.WorkDate,
		job.AvailableSlots(),
	)
	return msg
}
func FormatPaymentInstructions(job *models.Job, cardNumber, cardHolderName string) string {
	msg := fmt.Sprintf(`
✅ <b>JOY BAND QILINDI!</b>

Sizga 3 daqiqa vaqt berildi. Iltimos, quyidagi ma'lumotlarga to'lovni amalga oshiring va to'lov chekini yuboring.

<b>To'lov ma'lumotlari:</b>
💳 Karta: <code>%s</code>
👤 Ism: %s

<b>To'lov summasi:</b> %s so'm (Xizmat haqqi)

⏰ Vaqt: 3 daqiqa

To'lov chekini yuboring (screenshot):
`, cardNumber, cardHolderName, helper.FormatMoney(job.ServiceFee))
	return msg
}
