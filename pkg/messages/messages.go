package messages

// Common bot messages
const (
	MsgWelcome = `👋 Welcome to the Bot!

I'm here to help you. Use /help to see available commands.`

	MsgHelp = `📖 Available Commands:

/start - Start the bot
/help - Show this help message
/about - About this bot
/settings - Bot settings

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
)
