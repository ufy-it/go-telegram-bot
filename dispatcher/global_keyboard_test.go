package dispatcher_test

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	. "github.com/ufy-it/go-telegram-bot/dispatcher"
)

func TestRemoveKeyboard(t *testing.T) {
	keyboard := RemoveKeyboard()
	if keyboard == nil {
		t.Error("RemoveKeyboard() returned nil")
	}
	if keyboard != tgbotapi.NewRemoveKeyboard(false) {
		t.Error("RemoveKeyboard() returned unexpected value")
	}
}

func buttonsEqual(a, b [][]tgbotapi.KeyboardButton) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := 0; j < len(a[i]); j++ {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}

func TestSingleRowGlobalKeyboard(t *testing.T) {
	keyboard := SingleRowGlobalKeyboard(true, "a", "b", "c")

	expected := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("a"), tgbotapi.NewKeyboardButton("b"), tgbotapi.NewKeyboardButton("c"),
		},
	}
	if keyboard == nil {
		t.Error("SingleRowGlobalKeyboard returned nil")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).ResizeKeyboard != true {
		t.Error("SingleRowGlobalKeyboard returned incorrect ResizeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).Selective != false {
		t.Error("SingleRowGlobalKeyboard returned incorrect Selective value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).OneTimeKeyboard != false {
		t.Error("SingleRowGlobalKeyboard returned incorrect OneTimeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).InputFieldPlaceholder != "" {
		t.Error("SingleRowGlobalKeyboard returned incorrect InputFieldPlaceholder value")
	}

	if !buttonsEqual(keyboard.(tgbotapi.ReplyKeyboardMarkup).Keyboard, expected) {
		t.Error("SingleRowGlobalKeyboard returned incorrect Keyboard value")
	}
}

func TestSingleColumnGlobalKeyboard(t *testing.T) {
	keyboard := SingleColumnGlobalKeyboard(false, "a", "b", "c")

	expected := [][]tgbotapi.KeyboardButton{
		{tgbotapi.NewKeyboardButton("a")}, {tgbotapi.NewKeyboardButton("b")}, {tgbotapi.NewKeyboardButton("c")},
	}
	if keyboard == nil {
		t.Error("SingleColumnGlobalKeyboard returned nil")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).ResizeKeyboard != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect ResizeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).Selective != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect Selective value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).OneTimeKeyboard != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect OneTimeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).InputFieldPlaceholder != "" {
		t.Error("SingleColumnGlobalKeyboard returned incorrect InputFieldPlaceholder value")
	}

	if !buttonsEqual(keyboard.(tgbotapi.ReplyKeyboardMarkup).Keyboard, expected) {
		t.Error("SingleColumnGlobalKeyboard returned incorrect Keyboard value")
	}
}

func TestNewGlobalKeyboard(t *testing.T) {
	keyboard := NewGlobalKeyboard(false, [][]string{{"a", "b", "c"}, {"d", "e"}})

	expected := [][]tgbotapi.KeyboardButton{
		{
			tgbotapi.NewKeyboardButton("a"), tgbotapi.NewKeyboardButton("b"), tgbotapi.NewKeyboardButton("c"),
		},
		{
			tgbotapi.NewKeyboardButton("d"), tgbotapi.NewKeyboardButton("e"),
		},
	}
	if keyboard == nil {
		t.Error("SingleColumnGlobalKeyboard returned nil")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).ResizeKeyboard != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect ResizeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).Selective != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect Selective value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).OneTimeKeyboard != false {
		t.Error("SingleColumnGlobalKeyboard returned incorrect OneTimeKeyboard value")
	}

	if keyboard.(tgbotapi.ReplyKeyboardMarkup).InputFieldPlaceholder != "" {
		t.Error("SingleColumnGlobalKeyboard returned incorrect InputFieldPlaceholder value")
	}

	if !buttonsEqual(keyboard.(tgbotapi.ReplyKeyboardMarkup).Keyboard, expected) {
		t.Error("SingleColumnGlobalKeyboard returned incorrect Keyboard value")
	}
}
