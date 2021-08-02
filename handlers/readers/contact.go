package readers

import (
	"github.com/ufy-it/go-telegram-bot/handlers/buttons"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// RequestUserContact requests user to share it's contact.
func RequestUserContact(conversation BotConversation, message string, buttonText string, messageOnIncorrect string, finalMessage string) (tgbotapi.Contact, bool, error) {
	validator := func(update *tgbotapi.Update) (bool, string) {
		if update == nil || update.Message == nil || update.Message.Contact == nil {
			return false, messageOnIncorrect
		}
		return true, ""
	}
	msg := conversation.NewMessage(message)
	msg.ReplyMarkup = buttons.RequestContactButton(buttonText)
	reply, exit, err := AskGenericMessageReplyWithValidation(conversation, msg, buttons.EmptyButtonSet(), validator, false)
	contact := tgbotapi.Contact{}
	if reply != nil && reply.Message != nil && reply.Message.Contact != nil {
		contact = *reply.Message.Contact
	}
	if !exit && err == nil {
		finalMsg := conversation.NewMessage(finalMessage)
		finalMsg.ReplyMarkup = buttons.RemoveKeyboard()
		_, err = conversation.SendGeneralMessage(finalMsg)
	}
	return contact, exit, err
}
