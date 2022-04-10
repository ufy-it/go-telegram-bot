package conversation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var currentConversationID int64 = 0 // global conversation ID counter

// SendBot interface declares methods needed from Bot by a conversation
type SendBot interface {
	// Send sends a message through the bot
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
}

// SpecialMessageFuncType is a function type for dunction that should send messages in a special cases
type SpecialMessageFuncType func() error

// GlobalKeyboardFuncType is a function type that generates global keyboard for the conversation
// the keyboard could be used in the conversation
type GlobalKeyboardFuncType func() interface{}

// NewConversation creates a new conversation struct and assigns an incremental ID to it
func NewConversation(
	chatID int64,
	bot SendBot,
	botState state.BotState,
	tooManyMessages, cancelByBot, cancelByUser SpecialMessageFuncType,
	globalKeyboardFunc GlobalKeyboardFuncType,
	config Config) (*BotConversation, error) {
	return NewConversationWithID(
		currentConversationID,
		chatID,
		bot,
		botState,
		tooManyMessages,
		cancelByBot,
		cancelByUser,
		globalKeyboardFunc,
		config)
}

// NewConversationWithID creates a new conversation struct with pre-defined conversation ID.
// Should be used for starting conversations from a saved state
func NewConversationWithID(
	converationID, chatID int64,
	bot SendBot,
	botState state.BotState,
	tooManyMessages, cancelByBot, cancelByUser SpecialMessageFuncType,
	globalKeyboardFunc GlobalKeyboardFuncType,
	config Config) (*BotConversation, error) {
	if bot == nil {
		return nil, errors.New("cannot create conversation, bot object should not be nil")
	}
	if currentConversationID <= converationID {
		currentConversationID = converationID + 1
	}
	result := &BotConversation{
		updates: make(chan *tgbotapi.Update, config.MaxMessageQueue),

		chatID:         chatID,
		conversationID: converationID,

		bot:             bot,
		maxMessageQueue: config.MaxMessageQueue,
		timeoutMinutes:  config.TimeoutMinutes,

		cancelByBotMessage:     cancelByBot,
		cancelByUserMessage:    cancelByUser,
		tooManyMessagesMessage: tooManyMessages,

		globalKeyboardFunc: globalKeyboardFunc,

		canceled: false,

		mu: sync.Mutex{},
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

// BotConversation struct handlers all conversation-related data
// and should manage all interactions between a user and a command handler
type BotConversation struct {
	updates chan *tgbotapi.Update //channel with incoming messages from a user

	chatID         int64 // id of the telegram chat
	conversationID int64 // unique ID of the converation object

	maxMessageQueue int     // max size of messages buffer
	timeoutMinutes  int     // timeout from the latest message from a user in minutes before closing the conversation due to a long inactivity
	bot             SendBot // pointer to the bot

	cancelByBotMessage     SpecialMessageFuncType // message to send to a user if the conversation is closed from the bot side
	cancelByUserMessage    SpecialMessageFuncType // message to send to a user if they closed by user
	tooManyMessagesMessage SpecialMessageFuncType // message to send to a user in case to too many unprocessed messages

	globalKeyboardFunc GlobalKeyboardFuncType // function to generate global keyboard

	canceled bool // flag that indicates that the conversation was canceled by the user

	mu sync.Mutex // mutex to ensure that conversation will not send any messages after close
}

// cancelConversation cancels the conversation and sends special message to the chat
func (c *BotConversation) cancelConversation(cancelMessage SpecialMessageFuncType) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.canceled {
		return fmt.Errorf("the conversation with chat %d is already canceled", c.chatID)
	}
	c.canceled = true
	err := cancelMessage()
	if err != nil {
		return fmt.Errorf("error while sending cancel message to chat %d: %v", c.chatID, err)
	}
	return nil
}

// cancelByBot closes the conversation and sends "Cancel by bot message"
func (c *BotConversation) cancelByBot() error {
	return c.cancelConversation(c.cancelByBotMessage)
}

// CancelByUser tolds conversation object that user switched to another conversation
// that will be handled by a different conversation object
func (c *BotConversation) CancelByUser() error {
	return c.cancelConversation(c.cancelByUserMessage)
}

// IsCanceled indicates that the conversation was canceled by a user
func (c *BotConversation) IsCanceled() bool {
	return c.canceled
}

// PushUpdate checks whether a conversation can accept one more message, and forwards message to handler
func (c *BotConversation) PushUpdate(update *tgbotapi.Update) error {
	chatID, err := GetUpdateChatID(update)
	if err != nil {
		return err
	}
	if c.chatID != chatID {
		return fmt.Errorf("tried to process update from UserID %d in the conversation with UserID %d", chatID, c.chatID)
	}
	if len(c.updates) >= c.maxMessageQueue {
		err := c.tooManyMessagesMessage()
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

// GetUpdateFromUser waits for the next message from a user,
// and returns pointer to the message and a flag that indicates that conversation is over
func (c *BotConversation) GetUpdateFromUser(ctx context.Context) (*tgbotapi.Update, bool) {
	select {
	case update := <-c.updates:
		return update, false
	case <-ctx.Done():
		if !c.canceled {
			err := c.cancelByBot()
			if err != nil {
				logger.Error(err.Error())
			}
		}
		return nil, true
	case <-time.After(time.Duration(c.timeoutMinutes) * time.Minute):
		err := c.cancelByBot()
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, true
	}
}

// ChatID returns chat ID of the conversation
func (c *BotConversation) ChatID() int64 {
	return c.chatID
}

// ConverationID returns unique id of the conversation object
func (c *BotConversation) ConversationID() int64 {
	return c.conversationID
}

// SendGeneralMessage sends a general tgbotapi message using the Bot
// it returns messageID of the sent message end error if occured
func (c *BotConversation) SendGeneralMessage(msg tgbotapi.Chattable) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.canceled {
		return 0, fmt.Errorf("the conversation with chat %d is already canceled", c.chatID)
	}
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

// AnswerButton answer callback query
func (c *BotConversation) AnswerButton(callbackQueryID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.canceled {
		return fmt.Errorf("the conversation with chat %d is already canceled", c.chatID)
	}
	msg := tgbotapi.NewCallback(callbackQueryID, "")
	_, err := c.bot.Send(msg)
	return err
}

// GlobalKeyboard returns global keyboard struct generated for the chat
func (c *BotConversation) GlobalKeyboard() interface{} {
	return c.globalKeyboardFunc()
}
