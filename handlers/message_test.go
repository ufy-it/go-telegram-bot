package handlers_test

import (
	"testing"

	"github.com/ufy-it/go-telegram-bot/handlers"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

func TestMessageCreator(t *testing.T) {

	handler := handlers.MessageHandlerCreator("my message")

	if handler == nil {
		t.Errorf("MessageHandlerCreator returned nil")
	}

	conversation := mockConversation{}
	handlerStruct := handler(&conversation, nil)

	err := handlerStruct.Execute(state.NewBotState())

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
	handlerStruct := handler(&conversation, &tgbotapi.Message{MessageID: 133})

	err := handlerStruct.Execute(state.NewBotState())

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
