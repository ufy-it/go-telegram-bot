package conversation

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ufygobot/handlers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

type InputConversation interface {
	IsClosed() bool                              // check whether the conversation is closed
	Timeout() (bool, error)                      // check for a timeout, close convesation in case of it
	Kill()                                       // force close conversation
	PushMessage(message *tgbotapi.Message) error // push a new message from a user
}

// NewConversation creates a new conversation struct and starts handler in a separate thread
func NewConversation(maxMessageQueue int, timeoutMinutes int, finishSeconds int, chatID int64, bot *tgbotapi.BotAPI, wg *sync.WaitGroup) InputConversation {
	result := &conversation{
		exit:     make(chan struct{}),
		messages: make(chan *tgbotapi.Message, maxMessageQueue),

		chatID: chatID,

		bot:             bot,
		maxMessageQueue: maxMessageQueue,
		timeoutMinutes:  timeoutMinutes,
		finshSeconds:    finishSeconds,
		closeTime:       time.Now().Add(time.Duration(timeoutMinutes) * time.Minute),
		closed:          false,
		active:          false,
	}

	// start conversation handler in a separate thread
	go func(wg *sync.WaitGroup, conv *conversation) {
		wg.Add(1)
		defer wg.Done()
		handlers.HandleConversation(result)
	}(wg, result)

	return result
}

type conversation struct {
	exit     chan struct{}          // chanel that tells handler to exit
	messages chan *tgbotapi.Message //channel with incoming messages from a user

	chatID int64 //id of the telegram chat

	maxMessageQueue int              // max size of messages buffer
	timeoutMinutes  int              // timeout from the latest message from a user in minutes before closing the conversation due to a long inactivity
	finshSeconds    int              // timeout in seconds before closing the conversation because it is complete
	bot             *tgbotapi.BotAPI // pointer to the bot
	closed          bool             // flag to indicate whether the conversation is already closed
	active          bool             // flag to indicate that there is an ongoing conversation with a user. So it will require a notification in case of interruption
	closeTime       time.Time        // timepoint when the conversation should be closed
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

// Kill() closes the conversation and sends signal to a handler to finish
func (c *conversation) Kill() {
	c.closed = true
	close(c.exit) // this will notify the handler that it should exit. Should be called AFTER `c.closed = true`
	if c.active {
		msg := tgbotapi.NewMessage(c.chatID, "Бот вынужден прервать диалог. Вам нужно будет еще раз повторить все действия с начала.")
		_, err := c.bot.Send(msg)
		if err != nil {
			log.Printf("Error: while sending terminate mesage to chat %d: %v", c.chatID, err)
		}
	}
}

func (c *conversation) Finish() {
	if len(c.messages) == 0 && !c.IsClosed() {
		c.closeTime = time.Now().Add(time.Duration(c.finshSeconds) * time.Second)
		c.active = false
	}
}

// PushMessage checks whether a conversation can accept one more message, and forwards message to handler
func (c *conversation) PushMessage(message *tgbotapi.Message) error {
	if c.IsClosed() {
		return errors.New("the conversation is closed")
	}
	if c.chatID != message.Chat.ID {
		return fmt.Errorf("tried to process message from UserID %d in the conversation with UserID", message.Chat.ID, c.chatID)
	}
	if len(c.messages) >= c.maxMessageQueue {
		msg := tgbotapi.NewMessage(c.chatID,
			"У бота слишком много необработанных сообщений от вас. Подождите пока он их обработает, пока что все дальнейшие сообщения будут игнорироваться.")
		_, err := c.bot.Send(msg)
		return fmt.Errorf("to many open unprocessed messages in th conversation (%v)", err)
	}
	c.closeTime = time.Now().Add(time.Duration(c.timeoutMinutes) * time.Minute)
	c.active = true
	c.messages <- message
	return nil
}

// ReadMessage waits for the next message from a user, and returns pointer to the message and a flag that indicates that conversation is over
func (c *conversation) ReadMessage() (*tgbotapi.Message, bool) {
	if c.IsClosed() {
		return nil, true
	}
	select {
	case m := <-c.messages:
		if c.IsClosed() {
			return nil, true
		}
		return m, false
	case <-c.exit:
		return nil, true
	}
}

func (c *conversation) ChatID() int64 {
	return c.chatID
}

// Send message using the Bot
func (c *conversation) SendBotMessage(msg tgbotapi.Chattable) error {
	if c.IsClosed() {
		return errors.New("the conversation is closed")
	}
	// ToDo: check thy we are sending message to the same chatID
	_, err := c.bot.Send(msg)
	return err
}
