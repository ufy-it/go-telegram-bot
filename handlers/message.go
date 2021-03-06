package handlers

import (
	"context"
	"errors"

	"github.com/ufy-it/go-telegram-bot/handlers/readers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MessageHandlerCreator returns handler that prints a text message to a user
func MessageHandlerCreator(message string) HandlerCreatorType {
	return OneStepHandlerCreator(func(ctx context.Context, conversation readers.BotConversation) error {
		_, err := conversation.SendText(message)
		return err
	})
}

// ReplyMessageHandlerCreator returns handler that replies to user's message with a text message
func ReplyMessageHandlerCreator(message string) HandlerCreatorType {
	return OneStepHandlerCreator(func(ctx context.Context, conversation readers.BotConversation) error {
		firstUpdate, ok := ctx.Value(FirstUpdateVariable).(*tgbotapi.Update)
		if !ok {
			return errors.New("expected first update in the context")
		}

		var err error
		if firstUpdate.Message != nil {
			_, err = conversation.ReplyWithText(message, firstUpdate.Message.MessageID)
		} else {
			_, err = conversation.SendText(message)
		}
		return err
	})
}
