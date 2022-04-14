package handlers_test

import (
	"context"
	"testing"

	"github.com/ufy-it/go-telegram-bot/handlers"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestMessageCreator(t *testing.T) {

	handler := handlers.MessageHandlerCreator("my message")

	if handler == nil {
		t.Errorf("MessageHandlerCreator returned nil")
	}

	conversation := mockConversation{}
	handlerStruct := handler(context.Background(), &conversation)

	err := handlerStruct.Execute(0, state.NewBotState(state.NewFileState("")))

	if err != nil {
		t.Errorf("Unexpacted error: %v", err)
	}

	if conversation.sentText != "my message" {
		t.Errorf("Did not recieved expected message")
	}
}

func TestReplyMessageCreator(t *testing.T) {

	handler := handlers.ReplyMessageHandlerCreator("my reply")

	if handler == nil {
		t.Errorf("ReplyMessageHandlerCreator returned nil")
	}

	conversation := mockConversation{}
	handlerStruct := handler(context.WithValue(context.Background(), handlers.FirstUpdateVariable, &tgbotapi.Update{Message: &tgbotapi.Message{MessageID: 133}}), &conversation)

	err := handlerStruct.Execute(0, state.NewBotState(state.NewFileState("")))

	if err != nil {
		t.Errorf("Unexpacted error: %v", err)
	}

	if conversation.replyText != "my reply" {
		t.Errorf("Did not recieve expected message")
	}
	if conversation.replyMessageID != 133 {
		t.Errorf("Did not recieve expected reply Message ID")
	}
}
