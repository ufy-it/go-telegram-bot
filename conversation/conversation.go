package conversation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SendBot interface declares methods needed from Bot by a conversation
type SendBot interface {
	// Send sends a message through the bot
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
}

// Config is struct with configuration parameters for a conversation
type Config struct {
	MaxMessageQueue int // the maximum size of message queue for a conversation
	TimeoutMinutes  int // timeout for a user's input

	CloseByBotMessage     string // message to send to a user if the conversation is closed from the bot side
	CloseByUserMessage    string // message to send to a user if they decided to close the conversation
	ToManyMessagesMessage string // message to send to a user in case to too many unprocessed messages
}

// NewConversation createActionActions a new conversation struct and starts handler in a separate thread
func NewConversation(chatID int64, bot SendBot, botState state.BotState, config Config) (*BotConversation, error) {
	if bot == nil {
		return nil, errors.New("cannot create conversation, bot object should not be nil")
	}
	result := &BotConversation{
		updates: make(chan *tgbotapi.Update, config.MaxMessageQueue),

		chatID: chatID,

		bot:             bot,
		maxMessageQueue: config.MaxMessageQueue,
		timeoutMinutes:  config.TimeoutMinutes,

		closeByBotMessage:     config.CloseByBotMessage,
		closeByUserMessage:    config.CloseByUserMessage,
		toManyMessagesMessage: config.ToManyMessagesMessage,

		canceledByUser: false,
	}

	return result, nil
}

// GetUpdateChatID returns target ChatID from an update
func GetUpdateChatID(update *tgbotapi.Update) (int64, error) {
	if update == nil {
		return 0, errors.New("update is nil")
	}
	if update.Message != nil {
		if update.Message.Chat == nil {
			return 0, errors.New("expected chat field in update, got nil")
		}
		return update.Message.Chat.ID, nil
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message == nil {
			return 0, errors.New("expected message field in update, got nil")
		}
		if update.CallbackQuery.Message.Chat == nil {
			return 0, errors.New("expected chat field in update, got nil")
		}
		return update.CallbackQuery.Message.Chat.ID, nil
	}
	return 0, errors.New("usupported query type")
}

type BotConversation struct {
	updates chan *tgbotapi.Update //channel with incoming messages from a user

	chatID int64 //id of the telegram chat

	maxMessageQueue int     // max size of messages buffer
	timeoutMinutes  int     // timeout from the latest message from a user in minutes before closing the conversation due to a long inactivity
	bot             SendBot // pointer to the bot

	closeByBotMessage     string // message to send to a user if the conversation is closed from the bot side
	closeByUserMessage    string // message to send to a user if they decided to close the conversation
	toManyMessagesMessage string // message to send to a user in case to too many unprocessed messages

	canceledByUser bool // flag that indicates that the conversation was canceled by the user
}

// Kill() closes the conversation and sends signal to a handler to finish
func (c *BotConversation) CancelByBot() error {
	msg := tgbotapi.NewMessage(c.chatID, c.closeByBotMessage)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error while sending terminate mesage to chat %d: %v", c.chatID, err)
	}
	return nil
}

func (c *BotConversation) CancelByUser() error {
	c.canceledByUser = true
	msg := tgbotapi.NewMessage(c.chatID, c.closeByUserMessage)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, err := c.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("error while sending cancel-by-user message to chat %d: %v", c.chatID, err)
	}
	return nil
}

// PushMessage checks whether a conversation can accept one more message, and forwards message to handler
func (c *BotConversation) PushUpdate(update *tgbotapi.Update) error {
	chatID, err := GetUpdateChatID(update)
	if err != nil {
		return err
	}
	if c.chatID != chatID {
		return fmt.Errorf("tried to process update from UserID %d in the conversation with UserID %d", chatID, c.chatID)
	}
	if len(c.updates) >= c.maxMessageQueue {
		msg := tgbotapi.NewMessage(c.chatID, c.toManyMessagesMessage)
		_, err := c.bot.Send(msg)
		return fmt.Errorf("to many open unprocessed messages in the conversation (%v)", err)
	}
	c.updates <- update
	return nil
}

// GetFirstUpdateFromUser reads the first message in the conversation, the message should start a new handler.
// If context is closed, or conversation queue is empty, returns false
func (c *BotConversation) GetFirstUpdateFromUser(ctx context.Context) (*tgbotapi.Update, bool) {
	select {
	case <-ctx.Done():
		return nil, true // exit the conversation, as context is closed
	case update := <-c.updates:
		return update, false
	default:
		return nil, true // exit as there is no messages
	}
}

// ReadMessage waits for the next message from a user, and returns pointer to the message and a flag that indicates that conversation is over
func (c *BotConversation) GetUpdateFromUser(ctx context.Context) (*tgbotapi.Update, bool) {
	select {
	case update := <-c.updates:
		return update, false
	case <-ctx.Done():
		if !c.canceledByUser {
			err := c.CancelByBot()
			if err != nil {
				logger.Error(err.Error())
			}
		}
		return nil, true
	case <-time.After(time.Duration(c.timeoutMinutes) * time.Minute):
		err := c.CancelByBot()
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, true
	}
}

func (c *BotConversation) ChatID() int64 {
	return c.chatID
}

// Send message using the Bot
func (c *BotConversation) SendGeneralMessage(msg tgbotapi.Chattable) (int, error) {
	// ToDo: check thy we are sending message to the same chatID
	message, err := c.bot.Send(msg)
	return message.MessageID, err
}

// SendText creates a mesage with HTML parse mode from the input text and sends it through the bot
func (c *BotConversation) SendText(text string) (int, error) {
	return c.SendGeneralMessage(c.NewMessage(text))
}

// ReplyWithText replies to an existing message with a simple text message (HTML parse mode)
func (c *BotConversation) ReplyWithText(text string, msgID int) (int, error) {
	msg := c.NewMessage(text)
	msg.ReplyToMessageID = msgID
	return c.SendGeneralMessage(msg)
}

// DeleteMessage removes a message with msgID
func (c *BotConversation) DeleteMessage(msgID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(c.chatID, msgID)
	_, err := c.SendGeneralMessage(deleteMsg)
	if err.Error() == "json: cannot unmarshal bool into Go value of type tgbotapi.Message" { // bug in api
		err = nil
	}
	return err
}

// RemoveReplyMarkup removes inline button from a message
func (c *BotConversation) RemoveReplyMarkup(msgID int) error {
	return c.EditReplyMarkup(msgID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0)})
}

// EditReplyMarkup changes reply buttons in an existing message
func (c *BotConversation) EditReplyMarkup(msgID int, markup tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageReplyMarkup(c.chatID, msgID, markup)
	_, err := c.SendGeneralMessage(msg)
	return err
}

// NewMessage creates a new message for the conversation with HTML parse mode
func (c *BotConversation) NewMessage(text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(c.chatID, text)
	msg.ParseMode = "HTML"
	return msg
}

// NewPhotoShare creates a PhotoConfig for this conversation with HTML parse mode
func (c *BotConversation) NewPhotoShare(photoFileID string, caption string) tgbotapi.PhotoConfig {
	msg := tgbotapi.PhotoConfig{
		Caption: caption,
		BaseFile: tgbotapi.BaseFile{
			File: tgbotapi.FileID(photoFileID),
			BaseChat: tgbotapi.BaseChat{
				ChatID: c.ChatID(),
			},
		},
		ParseMode: "HTML",
	}
	return msg
}

// EditMessageText changes text of an existing message (ReplyMarkup will be deleted)
func (c *BotConversation) EditMessageText(messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(c.chatID, messageID, text)
	msg.ParseMode = "HTML"
	_, err := c.SendGeneralMessage(msg)
	return err
}

// EditMessageTextAndInlineMarkup changes both text and ReplyMarkup of an existing message
func (c *BotConversation) EditMessageTextAndInlineMarkup(messageID int, text string, markup tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageText(c.chatID, messageID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &markup
	_, err := c.SendGeneralMessage(msg)
	return err
}

func (c *BotConversation) AnswerButton(callbackQueryID string) error {
	msg := tgbotapi.NewCallback(callbackQueryID, "")
	_, err := c.bot.Send(msg)
	return err
}
