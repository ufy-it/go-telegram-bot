package handlers

import (
	"context"
	"errors"

	"github.com/ufy-it/go-telegram-bot/handlers/readers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
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
		firstMessage, ok := ctx.Value(FirstMessageVariable).(*tgbotapi.Message)
		if !ok {
			return errors.New("expected first message in the context")
		}
		_, err := conversation.ReplyWithText(message, firstMessage.MessageID)
		return err
	})
}
