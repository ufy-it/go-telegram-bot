package dispatcher

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
)

type GlobalKeyboardType interface{}

type GlobalKeyboardFuncType func(chatID int64) GlobalKeyboardType

func RemoveKeyboard() GlobalKeyboardType {
	return buttons.RemoveKeyboard()
}

func SingleRowGlobalKeyboard(resize bool, rowText ...string) GlobalKeyboardType {
	return NewGlobalKeyboard(resize, [][]string{rowText})
}

func SingleColumnGlobalKeyboard(resize bool, columnText ...string) GlobalKeyboardType {
	buttonText := make([][]string, len(columnText))
	for i, text := range columnText {
		buttonText[i] = make([]string, 1)
		buttonText[i][0] = text
	}
	return NewGlobalKeyboard(resize, buttonText)
}

func NewGlobalKeyboard(resize bool, buttonText [][]string) GlobalKeyboardType {
	bu := make([][]tgbotapi.KeyboardButton, len(buttonText))
	for i, row := range buttonText {
		bu[i] = make([]tgbotapi.KeyboardButton, len(row))
		for j := range row {
			bu[i][j] = tgbotapi.NewKeyboardButton(row[j])
		}
	}
	return tgbotapi.ReplyKeyboardMarkup{
		ResizeKeyboard: resize,
		Keyboard:       bu,
	}
}
