package buttons

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

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
