AIM : The task of the bot is to send the created job to the telegram channel,
 to receive the payment check from the users who have passed through the telegram channel to book the job, 
 and send it to the admin, and if the admin approves it, then attach the job to this user

Role : 
 - admin 
 - user
 - system


Admin : 
1. Admin ish yaratadi :
  - Ish haqqi (string)
  - Ovqat (string)
  - Vaqt (string)
  - Manzil (string)
  - Xizmat haqqi (int)
  - Avtobuslar (string)
  - Qo'shimcha (string)
  - Ish kuni (string)
  - Status (enum, automatic)
  - Order number (auto increment)
  - Kerakli ishchilar soni
  - Band qilgan ishchilar soni
2. Admin ishlarning ro'yhatini ko'ra oladi, ularni statuslarini o'zgartira oladi.
3. Admin bot orqali ish yaratgandan keyin , bot telegram kanalga ishni quyidagi formatda yuborishi kerak :

💰 Ish haqqi: Soatiga 20 000 so’m
🍛 Ovqat: kiritilmagan
⏰ Vaqt: 10:30 dan - kamida 5/6 soat ish
📱 Manzil: Yunusobod Amir Temur xiyoboniga yaqin
🌟 Xizmat haqi: 9 990 so'm
📝 Qo'shimcha: Ish yengil 3 -4 soatlik ish, 3-4 soatda bitsa ham 5 soatni puli beriladi

“Tahlash ishlari”

“Turgan narsani boshqa joyga”

“4 ta kere ekan aniq keladiganlari aka”

“Lekn aniq kelishsin aka 5 ta bolsa xam bolaadi”

🔴 Holat: To'ldi
📅 Ertaga
№ 3851
 
PC: qo'shimcha tarzda ishga yozilish uchun button bo'lishi kerak. foydalanuvchi ishga yozilish buttoni bosganda botga o'tishi kerak

