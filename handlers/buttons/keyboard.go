package buttons

import tgbotapi "github.com/Syfaro/telegram-bot-api"

// RequestContactButton returns keyboard with request contact button
func RequestContactButton(text string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{
				tgbotapi.NewKeyboardButtonContact(text),
			},
		},
	}
}

// RemoveKeyboard return markup that hides custom keyboard
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(false)
}
