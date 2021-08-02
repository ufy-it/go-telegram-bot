package conversation

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"ufygobot/handlers"
	"ufygobot/logger"
	"ufygobot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// InputConversation is an interface that allows to push new update into a conversation and manage conversation lifetime
type InputConversation interface {
	IsClosed() bool                           // check whether the conversation is closed
	Timeout() (bool, error)                   // check for a timeout, close convesation in case of it
	Kill()                                    // force close conversation
	CancelByUser() error                      // cancel conversation on user request
	PushUpdate(update *tgbotapi.Update) error // push a new update from a user
}

// SendBot interface declares methods needed from Bot by a conversation
type SendBot interface {
	// Send sends a message through the bot
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	AnswerCallbackQuery(config tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error)
}

// Config is struct with configuration parameters for a conversation
type Config struct {
	MaxMessageQueue int                       // the maximum size of message queue for a conversation
	TimeoutMinutes  int                       // Lifetime of an open conversation in minutes
	FinishSeconds   int                       // Lifetime of a finished conversation in seconds
	Handlers        *handlers.CommandHandlers // list of handlers for command handling

	CloseByBotMessage     string // message to send to a user if the conversation is closed from the bot side
	CloseByUserMessage    string // message to send to a user if they decided to close the conversation
	ToManyMessagesMessage string // message to send to a user in case to too many unprocessed messages
}

// NewConversation createActionActions a new conversation struct and starts handler in a separate thread
func NewConversation(chatID int64, bot SendBot, botState state.BotState, wg *sync.WaitGroup, config Config) (InputConversation, error) {
	if wg == nil {
		return nil, errors.New("cannot create conversation, wait group should not be nil")
	}
	if bot == nil {
		return nil, errors.New("cannot create conversation, bot object should not be nil")
	}
	result := &conversation{
		exit:    make(chan struct{}),
		updates: make(chan *tgbotapi.Update, config.MaxMessageQueue),

		chatID: chatID,

		bot:             bot,
		maxMessageQueue: config.MaxMessageQueue,
		timeoutMinutes:  config.TimeoutMinutes,
		finshSeconds:    config.FinishSeconds,
		closeTime:       time.Now().Add(time.Duration(config.TimeoutMinutes) * time.Minute),
		closed:          false,
		active:          false,

		closeByBotMessage:     config.CloseByBotMessage,
		closeByUserMessage:    config.CloseByUserMessage,
		toManyMessagesMessage: config.ToManyMessagesMessage,
	}

	// start conversation handler in a separate thread
	go func(wg *sync.WaitGroup, conv *conversation) {
		wg.Add(1)
		defer wg.Done()
		handlers.HandleConversation(result, botState, config.Handlers)
	}(wg, result)

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

type conversation struct {
	exit    chan struct{}         // chanel that tells handler to exit
	updates chan *tgbotapi.Update //channel with incoming messages from a user

	chatID int64 //id of the telegram chat

	maxMessageQueue int        // max size of messages buffer
	timeoutMinutes  int        // timeout from the latest message from a user in minutes before closing the conversation due to a long inactivity
	finshSeconds    int        // timeout in seconds before closing the conversation because it is complete
	bot             SendBot    // pointer to the bot
	closed          bool       // flag to indicate whether the conversation is already closed
	active          bool       // flag to indicate that there is an ongoing conversation with a user. So it will require a notification in case of interruption
	closeTime       time.Time  // timepoint when the conversation should be closed
	mu              sync.Mutex // mutex for synchronizing conversation closing

	closeByBotMessage     string // message to send to a user if the conversation is closed from the bot side
	closeByUserMessage    string // message to send to a user if they decided to close the conversation
	toManyMessagesMessage string // message to send to a user in case to too many unprocessed messages
}

func (c *conversation) IsClosed() bool {
	return c.closed
}

// Timeout() checks if the conversation should be closed due a timeout from the last interraction
func (c *conversation) Timeout() (bool, error) {
	if c.IsClosed() {
		return true, errors.New("the conversaton is already closed")
	}
	if time.Now().After(c.closeTime) {
		c.Kill()
		return true, nil
	}
	return false, nil
}

// tryClose returns false if the conversation was already closed, or closes the conversation and returns true
func (c *conversation) tryClose() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.IsClosed() {
		return false // already closed
	}
	c.closed = true
	close(c.exit) // this will notify the handler that it should exit. Should be called AFTER `c.closed = true`
	return true
}

// Kill() closes the conversation and sends signal to a handler to finish
func (c *conversation) Kill() {
	if !c.tryClose() {
		return
	}
	if c.active {
		msg := tgbotapi.NewMessage(c.chatID, c.closeByBotMessage)
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		_, err := c.bot.Send(msg)
		if err != nil {
			logger.Error("error while sending terminate mesage to chat %d: %v", c.chatID, err)
		}
	}
}

func (c *conversation) CancelByUser() error {
	if !c.tryClose() {
		return nil
	}
	if c.active {
		msg := tgbotapi.NewMessage(c.chatID, c.closeByUserMessage)
		msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		_, err := c.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("Error while sending cancel-by-user message to chat %d: %v", c.chatID, err)
		}
	}
	return nil
}

func (c *conversation) FinishConversation() {
	if len(c.updates) == 0 && !c.IsClosed() {
		c.closeTime = time.Now().Add(time.Duration(c.finshSeconds) * time.Second)
		c.active = false
	}
}

// PushMessage checks whether a conversation can accept one more message, and forwards message to handler
func (c *conversation) PushUpdate(update *tgbotapi.Update) error {
	if c.IsClosed() {
		return errors.New("the conversation is closed")
	}
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
	c.closeTime = time.Now().Add(time.Duration(c.timeoutMinutes) * time.Minute)
	c.active = true
	c.updates <- update
	return nil
}

// ReadMessage waits for the next message from a user, and returns pointer to the message and a flag that indicates that conversation is over
func (c *conversation) GetUpdateFromUser() (*tgbotapi.Update, bool) {
	if c.IsClosed() {
		return nil, true
	}
	select {
	case u := <-c.updates:
		if c.IsClosed() {
			return nil, true
		}
		return u, false
	case <-c.exit:
		return nil, true
	}
}

func (c *conversation) ChatID() int64 {
	return c.chatID
}

// Send message using the Bot
func (c *conversation) SendGeneralMessage(msg tgbotapi.Chattable) (int, error) {
	if c.IsClosed() {
		return 0, errors.New("the conversation is closed")
	}
	// ToDo: check thy we are sending message to the same chatID
	message, err := c.bot.Send(msg)
	return message.MessageID, err
}

// SendText creates a mesage with HTML parse mode from the input text and sends it through the bot
func (c *conversation) SendText(text string) (int, error) {
	return c.SendGeneralMessage(c.NewMessage(text))
}

// ReplyWithText replies to an existing message with a simple text message (HTML parse mode)
func (c *conversation) ReplyWithText(text string, msgID int) (int, error) {
	msg := c.NewMessage(text)
	msg.ReplyToMessageID = msgID
	return c.SendGeneralMessage(msg)
}

// DeleteMessage removes a message with msgID
func (c *conversation) DeleteMessage(msgID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(c.chatID, msgID)
	_, err := c.SendGeneralMessage(deleteMsg)
	return err
}

// RemoveReplyMarkup removes inline button from a message
func (c *conversation) RemoveReplyMarkup(msgID int) error {
	return c.EditReplyMarkup(msgID, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0)})
}

// EditReplyMarkup changes reply buttons in an existing message
func (c *conversation) EditReplyMarkup(msgID int, markup tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageReplyMarkup(c.chatID, msgID, markup)
	_, err := c.SendGeneralMessage(msg)
	return err
}

// NewMessage creates a new message for the conversation with HTML parse mode
func (c *conversation) NewMessage(text string) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(c.chatID, text)
	msg.ParseMode = "HTML"
	return msg
}

// NewPhotoShare creates a PhotoConfig for this conversation with HTML parse mode
func (c *conversation) NewPhotoShare(photoFileID string, caption string) tgbotapi.PhotoConfig {
	msg := tgbotapi.NewPhotoShare(c.chatID, photoFileID)
	msg.Caption = caption
	msg.ParseMode = "HTML"
	return msg
}

// EditMessageText changes text of an existing message (ReplyMarkup will be deleted)
func (c *conversation) EditMessageText(messageID int, text string) error {
	msg := tgbotapi.NewEditMessageText(c.chatID, messageID, text)
	msg.ParseMode = "HTML"
	_, err := c.SendGeneralMessage(msg)
	return err
}

// EditMessageTextAndInlineMarkup changes both text and ReplyMarkup of an existing message
func (c *conversation) EditMessageTextAndInlineMarkup(messageID int, text string, markup tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewEditMessageText(c.chatID, messageID, text)
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = &markup
	_, err := c.SendGeneralMessage(msg)
	return err
}

func (c *conversation) AnswerButton(callbackQueryID string) error {
	msg := tgbotapi.NewCallback(callbackQueryID, "")
	_, err := c.bot.AnswerCallbackQuery(msg)
	return err
}
