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

// CallbackCommandSelector creates CommandSelector that accepts callbeck data if it matches a regular expression
func CallbackCommandSelector(callbackRe string) CommandSelectorType {
	re := regexp.MustCompile(callbackRe)
	return func(firstUpdate *tgbotapi.Update) bool {
		return firstUpdate != nil && firstUpdate.CallbackQuery != nil && re.MatchString(firstUpdate.CallbackQuery.Data)
	}
}

// CommandOrCallbackRegExpCommand creates CommandSelector that accepts command text or callback data if it matches a regular expression
func CommandOrCallbackRegExpCommand(commandRe, callbackRe string) CommandSelectorType {
	command := RegExpCommandSelector(commandRe)
	callback := CallbackCommandSelector(callbackRe)
	return func(firstUpdate *tgbotapi.Update) bool {
		return command(firstUpdate) || callback(firstUpdate)
	}
}
