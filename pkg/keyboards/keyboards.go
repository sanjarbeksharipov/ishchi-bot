package keyboards

import (
	tele "gopkg.in/telebot.v4"
)

// MainMenuKeyboard returns the main menu inline keyboard
func MainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnHelp := menu.Data("📖 Help", "help")
	btnAbout := menu.Data("ℹ️ About", "about")
	btnSettings := menu.Data("⚙️ Settings", "settings")

	menu.Inline(
		menu.Row(btnHelp, btnAbout),
		menu.Row(btnSettings),
	)

	return menu
}

// ConfirmationKeyboard returns a yes/no confirmation keyboard
func ConfirmationKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnYes := menu.Data("✅ Yes", "confirm_yes")
	btnNo := menu.Data("❌ No", "confirm_no")

	menu.Inline(
		menu.Row(btnYes, btnNo),
	)

	return menu
}

// BackKeyboard returns a simple back button keyboard
func BackKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Back", "back")
	menu.Inline(menu.Row(btnBack))
	return menu
}
