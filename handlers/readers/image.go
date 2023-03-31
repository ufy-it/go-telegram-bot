package readers

import (
	"context"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UserImageAndDataReply contains image input from a user
type UserImageAndDataReply struct {
	Exit  bool
	Data  string
	Image []tgbotapi.PhotoSize
}

// GetImage asks a user to send image
func GetImage(ctx context.Context, conversation BotConversation, text string, navigation buttons.ButtonSet, textOnIncorrect string) (UserImageAndDataReply, error) {
	msg := conversation.NewMessage(text)
	validator := func(update *tgbotapi.Update) (bool, string) {
		if update != nil && update.Message != nil && update.Message.Photo != nil && len(update.Message.Photo) > 0 {
			return true, ""
		}
		return false, textOnIncorrect
	}
	reply, exit, err := AskGenericMessageReplyWithValidation(ctx, conversation, msg, navigation, validator, true)
	result := UserImageAndDataReply{Exit: exit}
	if reply != nil && reply.CallbackQuery != nil {
		result.Data = reply.CallbackQuery.Data
	}
	if reply != nil && reply.Message != nil && reply.Message.Photo != nil && len(reply.Message.Photo) > 0 {
		result.Image = reply.Message.Photo
	}
	return result, err
}
