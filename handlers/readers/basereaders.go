package readers

import (
	"context"
	"time"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// BotConversation implements interface for reading and writing messages from the bot side
type BotConversation interface {
	ChatID() int64 // get current chatID

	GetUpdateFromUser(ctx context.Context) (*tgbotapi.Update, bool) // read update from a user (will hang until a user sends new mupdate, or conversation is closed)

	NewPhotoShare(photoFileID string, caption string) tgbotapi.PhotoConfig // create a message with a Photo (should be uploaded to the telegram an caption)
	NewMessage(text string) tgbotapi.MessageConfig                         // Create a new Text message with HTML parsing

	SendGeneralMessage(msg tgbotapi.Chattable) (int, error) // send general message to a user. This method is not safe, so use it as less as possible
	SendText(text string) (int, error)                      // send text with HTML parsing
	ReplyWithText(text string, messageID int) (int, error)  // reply with text message to the existing message
	AnswerButton(callbackQueryID string) error              // record answer to an inline button press
	DeleteMessage(messageID int) error                      // delete an existing message

	RemoveReplyMarkup(messageID int) error                                                                 // remove reply markup from the existing message
	EditReplyMarkup(messageID int, markup tgbotapi.InlineKeyboardMarkup) error                             // replace reply markup in the existing message
	EditMessageText(messageID int, text string) error                                                      // replace text in the existing message
	EditMessageTextAndInlineMarkup(messageID int, text string, markup tgbotapi.InlineKeyboardMarkup) error // replace both text and reply markup in the existing message
}

// UserTextAndDataReply handles simplified information from the update
type UserTextAndDataReply struct {
	MessageID       int
	CallbackQueryID string
	Text            string
	Data            string
	Exit            bool
}

// UserTimeAndDataReply handles user input a Time (or Date)
type UserTimeAndDataReply struct {
	MessageID       int
	CallbackQueryID string
	Time            time.Time
	Data            string
	Exit            bool
}

// ParseUserTextAndDataReply parses text message and button data from the user's input
func ParseUserTextAndDataReply(update *tgbotapi.Update, exit bool) UserTextAndDataReply {
	result := UserTextAndDataReply{
		Exit: exit,
	}
	if exit {
		return result
	}
	if update != nil && update.Message != nil {
		result.Text = update.Message.Text
		result.MessageID = update.Message.MessageID
	}
	if update != nil && update.CallbackQuery != nil {
		result.CallbackQueryID = update.CallbackQuery.ID
		result.Data = update.CallbackQuery.Data
	}
	return result
}

// ParseUserTextDataAndErrorReply creates pair (UserTextAndDataReply, error) from triple (update *tgbotapi.Update, exit bool, err error)
func ParseUserTextDataAndErrorReply(update *tgbotapi.Update, exit bool, err error) (UserTextAndDataReply, error) {
	return ParseUserTextAndDataReply(update, exit), err
}

// ReadRawTextAndDataResult waits for an update fro a user, and parses the update to UserTextAndDataReply struct
func ReadRawTextAndDataResult(ctx context.Context, conversation BotConversation) UserTextAndDataReply {
	update, exit := conversation.GetUpdateFromUser(ctx)
	return ParseUserTextAndDataReply(update, exit)
}
