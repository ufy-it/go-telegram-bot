package dispatcher

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
)

// GlobalKeyboardType is a alias for interface{}
type GlobalKeyboardType interface{}

// GlobalKeyboardFuncType is a type for function that generates a global keyboard for a specific chat
type GlobalKeyboardFuncType func(chatID int64) GlobalKeyboardType

// RemoveKeyboard is a Global Keyboard function that removes an permanent keyboard
func RemoveKeyboard() GlobalKeyboardType {
	return buttons.RemoveKeyboard()
}

// SingleRowGlobalKeyboard generates a single-row Global Keyboard from slice of strings
func SingleRowGlobalKeyboard(resize bool, rowText ...string) GlobalKeyboardType {
	return NewGlobalKeyboard(resize, [][]string{rowText})
}

// SingleColumnGlobalKeyboard generates a single-column Global Keyboard from slice of strings
func SingleColumnGlobalKeyboard(resize bool, columnText ...string) GlobalKeyboardType {
	buttonText := make([][]string, len(columnText))
	for i, text := range columnText {
		buttonText[i] = make([]string, 1)
		buttonText[i][0] = text
	}
	return NewGlobalKeyboard(resize, buttonText)
}

// NewGlobalKeyboard generates a Global Keyboard from a two dimentional array of strings
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
