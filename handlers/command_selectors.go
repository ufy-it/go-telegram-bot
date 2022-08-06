package handlers

import (
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// RegExpCommandSelector creates CommandSeletor that accepts command text if it matches a regular expression
func RegExpCommandSelector(commandRe string) CommandSelectorType {
	re := regexp.MustCompile(commandRe)
	return func(firstUpdate *tgbotapi.Update) bool {
		return firstUpdate != nil && firstUpdate.Message != nil && (re.MatchString(firstUpdate.Message.Text) || re.MatchString(firstUpdate.Message.Command()))
	}
}

// RegExCallbackSelector creates CommandSelector that accepts callbeck data if it matches a regular expression
func RegExCallbackSelector(callbackRe string) CommandSelectorType {
	re := regexp.MustCompile(callbackRe)
	return func(firstUpdate *tgbotapi.Update) bool {
		return firstUpdate != nil && firstUpdate.CallbackQuery != nil && re.MatchString(firstUpdate.CallbackQuery.Data)
	}
}

// CommandOrCallbackRegExpSelector creates CommandSelector that accepts command text or callback data if it matches a regular expression
func CommandOrCallbackRegExpSelector(commandRe, callbackRe string) CommandSelectorType {
	return OrSelector(RegExpCommandSelector(commandRe), RegExCallbackSelector(callbackRe))
}

// ImageSelector creates CommandSelector that accepts image
func ImageSelector() CommandSelectorType {
	return func(firstUpdate *tgbotapi.Update) bool {
		return firstUpdate != nil && firstUpdate.Message != nil && firstUpdate.Message.Photo != nil
	}
}

// OrSelecor creates CommandSelector that accepts command if any of inline selectors accept it
func OrSelector(selectors ...CommandSelectorType) CommandSelectorType {
	return func(firstUpdate *tgbotapi.Update) bool {
		for _, selector := range selectors {
			if selector(firstUpdate) {
				return true
			}
		}
		return false
	}
}

// AndSelector creates CommandSelector that accepts command if all of inline selectors accept it
func AndSelector(selectors ...CommandSelectorType) CommandSelectorType {
	return func(firstUpdate *tgbotapi.Update) bool {
		for _, selector := range selectors {
			if !selector(firstUpdate) {
				return false
			}
		}
		return true
	}
}

// NotSelector creates CommandSelector that accepts command if inline selector does not accept it
func NotSelector(selector CommandSelectorType) CommandSelectorType {
	return func(firstUpdate *tgbotapi.Update) bool {
		return !selector(firstUpdate)
	}
}
