package readers

import (
	"context"
	"regexp"
	"time"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
	"github.com/ufy-it/go-telegram-bot/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// InputTextValidation is a type for function that validates messaged recieved from a user
type InputTextValidation func(text string) bool // predicat that accepts only valid strings

// BotUpdateValidator validates an update recieved from a user.
// If the update does not pass the criteria, the validator can return a string with the clarification
type BotUpdateValidator func(update *tgbotapi.Update) (bool, string)

// AskGenericMessageReplyWithValidation continuously asks a user for reply until reply passes validation functor
func AskGenericMessageReplyWithValidation(ctx context.Context,
	conversation BotConversation,
	message tgbotapi.Chattable,
	bs buttons.ButtonSet,
	validator BotUpdateValidator,
	repeatOriginalOnIncorrect bool) (*tgbotapi.Update, bool, error) {
	sendOriginalMessage := func() (int, error) {
		if message != nil { //do nothing on empty message
			msgID, err := conversation.SendGeneralMessage(message)
			if err != nil {
				return -1, err
			}
			if !bs.IsEmpty() {
				err = conversation.EditReplyMarkup(msgID, bs.GetInlineKeyboard())
			}
			return msgID, err
		}
		return -1, nil
	}

	sentID, err := sendOriginalMessage()

	defer func() { //hide buttons before exiting
		if bs.IsEmpty() {
			return
		}
		err := conversation.RemoveReplyMarkup(sentID)
		if err != nil {
			logger.Warning("failed to hide reply markup in message: %v", err)
		}
	}()

	for {
		if err != nil {
			return nil, false, err
		}

		reply, exit := conversation.GetUpdateFromUser(ctx) // get reply from user
		if exit {
			return nil, true, nil
		}
		if reply.CallbackQuery != nil && reply.CallbackQuery.Data != "" {
			var berr error
			reply.CallbackQuery.Data, berr = bs.FindButtonData(reply.CallbackQuery.Data) // convert data from guid to an original
			if berr == nil {
				return reply, false, nil
			}
			logger.Warning("unknown button pressed: %v", berr) //log unknown button press
			continue
		}
		if isValid, messageOnIncorrect := validator(reply); !isValid {
			if messageOnIncorrect != "" {
				if reply.Message != nil {
					_, err = conversation.ReplyWithText(messageOnIncorrect, reply.Message.MessageID)
				} else {
					_, err = conversation.SendText(messageOnIncorrect)
				}
				if err != nil {
					return nil, false, err
				}
			}
		} else {
			return reply, false, nil
		}
		if sentID != -1 && repeatOriginalOnIncorrect { // no need to delete a message
			err = conversation.DeleteMessage(sentID) // delete and resend original message
			if err != nil {
				logger.Warning("failed to delete old mesage: %v", err)
			}
			sentID, err = sendOriginalMessage()
		}
	}
}

// AskTextMessageReplyWithValidation is a complex conversation that keep asking a question to user
// until they give information that pass the validation or press a button
func AskTextMessageReplyWithValidation(ctx context.Context,
	conversation BotConversation,
	message string, bs buttons.ButtonSet,
	validator InputTextValidation, messageOnIncorrect string) (UserTextAndDataReply, error) {
	outerValidator := func(update *tgbotapi.Update) (bool, string) {
		if update != nil && update.Message != nil && validator(update.Message.Text) {
			return true, ""
		}
		return false, messageOnIncorrect
	}
	reply, exit, err := AskGenericMessageReplyWithValidation(ctx, conversation, conversation.NewMessage(message), bs, outerValidator, true)
	return ParseUserTextAndDataReply(reply, exit), err
}

//AskOnlyButtonReply sends message to a user and accepts only buttons
func AskOnlyButtonReply(ctx context.Context, conversation BotConversation, message tgbotapi.Chattable, bs buttons.ButtonSet, messageOnText string) (UserTextAndDataReply, error) {
	validator := func(update *tgbotapi.Update) (bool, string) {
		return false, messageOnText
	}
	return ParseUserTextDataAndErrorReply(AskGenericMessageReplyWithValidation(ctx, conversation, message, bs, validator, true))
}

// AskReplyDate asks a user to enter date in format dd.mm.yyyy
func AskReplyDate(ctx context.Context, conversation BotConversation, message string, bs buttons.ButtonSet, messageOnIncorrectInput string, location *time.Location) (UserTimeAndDataReply, error) {
	validator := func(text string) bool {
		_, err := time.ParseInLocation("02.01.2006", text, location)
		return err == nil
	}
	preResult, err := AskTextMessageReplyWithValidation(ctx, conversation, message, bs, validator, messageOnIncorrectInput)
	result := UserTimeAndDataReply{
		MessageID: preResult.MessageID,
		Data:      preResult.Data,
		Exit:      preResult.Exit,
	}
	if err != nil {
		return result, err
	}
	if preResult.Text != "" {
		result.Time, err = time.ParseInLocation("02.01.2006", preResult.Text, location)
	}
	return result, err
}

// AskReplyEmail asks user to enter a vaid email address
func AskReplyEmail(ctx context.Context, conversation BotConversation, message string, bs buttons.ButtonSet, messageOnIncorrectInput string) (UserTextAndDataReply, error) {
	validator := func(text string) bool {
		if len(text) < 3 || len(text) > 254 {
			return false
		}
		// regexp by W3C
		return regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$").MatchString(text)
	}
	return AskTextMessageReplyWithValidation(ctx, conversation, message, bs, validator, messageOnIncorrectInput)
}
