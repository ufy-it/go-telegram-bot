package handlers_test

import (
	"errors"
	"testing"

	"ufygobot/handlers"
	"ufygobot/handlers/readers"
	"ufygobot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

type mockConversation struct {
	sentText       string
	replyText      string
	replyMessageID int
}

func (mc *mockConversation) IsClosed() bool {
	return false
}

func (mc *mockConversation) FinishConversation() {

}

func (mc *mockConversation) ChatID() int64 {
	return 0
}

func (mc *mockConversation) GetUpdateFromUser() (*tgbotapi.Update, bool) {
	return nil, false
}

func (mc *mockConversation) NewPhotoShare(photoFileID string, caption string) tgbotapi.PhotoConfig {
	return tgbotapi.PhotoConfig{}
}

func (mc *mockConversation) NewMessage(text string) tgbotapi.MessageConfig {
	return tgbotapi.MessageConfig{}
}

func (mc *mockConversation) SendGeneralMessage(msg tgbotapi.Chattable) (int, error) {
	return 0, nil
}

func (mc *mockConversation) SendText(text string) (int, error) {
	mc.sentText = text
	return 0, nil
}

func (mc *mockConversation) ReplyWithText(text string, messageID int) (int, error) {
	mc.replyMessageID = messageID
	mc.replyText = text
	return 0, nil
}

func (mc *mockConversation) AnswerButton(callbackQueryID string) error {
	return nil
}

func (mc *mockConversation) DeleteMessage(messageID int) error {
	return nil
}

func (mc *mockConversation) RemoveReplyMarkup(messageID int) error {
	return nil
}

func (mc *mockConversation) EditReplyMarkup(messageID int, markup tgbotapi.InlineKeyboardMarkup) error {
	return nil
}

func (mc *mockConversation) EditMessageText(messageID int, text string) error {
	return nil
}

func (mc *mockConversation) EditMessageTextAndInlineMarkup(messageID int, text string, markup tgbotapi.InlineKeyboardMarkup) error {
	return nil
}

func TestOneStepCreator(t *testing.T) {
	var step handlers.OneStepCommandHandlerType = func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
		return errors.New("Some error")
	}
	handler := handlers.OneStepHandlerCreator(step)

	if handler == nil {
		t.Errorf("OneStepHandlerCreator returned nil")
	}

	handlerStruct := handler(&mockConversation{}, nil)

	err := handlerStruct.Execute(state.NewBotState())

	if err == nil || err.Error() != "Some error" {
		t.Errorf("Unexpacted error: %v", err)
	}
}
