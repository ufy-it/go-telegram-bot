package dispatcher

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"ufygobot/conversation"
	"ufygobot/logger"
	"ufygobot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// Dispatcher is an object that manages conversations and routers input messsages to needed conversations
type Dispatcher struct {
	maxOpenConversations int
	closed               bool
	closeTimeoutSeconds  int

	mu sync.Mutex     //mutex to sync operations over the conversations map between main thread and cleaning routing
	wg sync.WaitGroup // wait group to finish all routins

	conversations      map[int64]conversation.InputConversation
	conversationConfig conversation.Config

	singleMessageTrySendInterval   int
	cancelCommand                  string // command that cancels any conversation
	toManyOpenConversationsMessage string

	bot   *tgbotapi.BotAPI
	state state.BotState
}

// Config contains configuration parameters for a new dispatcher
type Config struct {
	MaxOpenConversations           int                 // the maximum number of open conversations
	CloseTimeoutSeconds            int                 // timeout in seconds for closing the dispatcher
	ClearJobInterval               int                 // interval in seconds between Cleaning Job runs
	SingleMessageTrySendInterval   int                 // interval between several tries of send single message to a user
	ConversationConfig             conversation.Config // configuration for a conversation
	CancelCommand                  string              // the command that a user can use to cancel any conversation
	ToManyOpenConversationsMessage string              // message to send to a user if a conversation cannot be started
}

// clearOldConversation should run in a separate routine
// and periodically remove outdated conversations
func (d *Dispatcher) clearOldConversations(intervalSeconds int) {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		if d.IsClosed() {
			return
		}
		d.mu.Lock()
		idsToDelete := make([]int64, 0)
		for id, conversation := range d.conversations {
			if conversation == nil {
				idsToDelete = append(idsToDelete, id)
				continue
			}
			isTimeout, err := conversation.Timeout()
			if err != nil {
				logger.Warning("error during cleaning conversations: %v", err)
			}
			if isTimeout {
				idsToDelete = append(idsToDelete, id)
			}
		}
		for _, id := range idsToDelete {
			delete(d.conversations, id)
		}
		d.mu.Unlock()
		time.Sleep(time.Duration(time.Duration(intervalSeconds) * time.Second))
	}
}

// Close kills all conversations
func (d *Dispatcher) Close() error {
	if d.IsClosed() {
		return errors.New("cannot close Dispatcher that is already closed")
	}
	d.closed = true // the order of exiting is important. Keep multythreading in mind before changing anything
	d.mu.Lock()
	defer d.mu.Unlock()
	err := d.state.Close()
	if err != nil {
		logger.Warning("could not save state to file: %v", err)
	}
	for _, conv := range d.conversations {
		conv.Kill()
	}
	for id := range d.conversations {
		delete(d.conversations, id)
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		d.wg.Wait()
	}()

	select {
	case <-c: // everything finished successfully
		return nil
	case <-time.After(time.Duration(d.closeTimeoutSeconds) * time.Second):
		return errors.New("some threads were not finished in time")
	}
}

// NewDispatcher creates a new Dispatcher objects and starts a separate thread to clear old conversations
func NewDispatcher(config Config, bot *tgbotapi.BotAPI, stateFile string) *Dispatcher {
	d := &Dispatcher{
		conversations:                  make(map[int64]conversation.InputConversation),
		conversationConfig:             config.ConversationConfig,
		maxOpenConversations:           config.MaxOpenConversations,
		singleMessageTrySendInterval:   config.SingleMessageTrySendInterval,
		bot:                            bot,
		mu:                             sync.Mutex{},
		wg:                             sync.WaitGroup{},
		closed:                         false,
		closeTimeoutSeconds:            config.CloseTimeoutSeconds,
		cancelCommand:                  config.CancelCommand,
		toManyOpenConversationsMessage: config.ToManyOpenConversationsMessage,
		state:                          state.NewBotState(),
	}
	if stateFile != "" { //load previouse state from file
		err := d.state.LoadState(stateFile)
		if err != nil {
			logger.Warning("cannot load previouse state: %v. Will start from blank", err)
		}
	}
	// resume conversations from the state
	for _, chatID := range d.state.GetConversations() {
		conv, err := conversation.NewConversation(chatID, d.bot, d.state, &d.wg, d.conversationConfig)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		d.conversations[chatID] = conv
		if _, ok := d.conversations[chatID]; !ok {
			logger.Warning("cannot start conversation with %d from state", chatID)
		}
	}
	go d.clearOldConversations(config.ClearJobInterval) //start cleaning in a separate thread
	return d
}

// IsClosed shows whether the dispatcher has been closed, or is active
func (d *Dispatcher) IsClosed() bool {
	return d.closed
}

// isCancelCommand returns true if a user entered cancel command
func (d *Dispatcher) isCancelCommand(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Text == d.cancelCommand
}

// SendSingleMessage waits until the conversation with chatID is closed and sends a single message from bot to a user
func (d *Dispatcher) SendSingleMessage(chatID int64, text string) error {
	closedErr := errors.New("the dispatcher is closed")
	msg := tgbotapi.NewMessage(chatID, text)
	for {
		if d.IsClosed() {
			return closedErr
		}
		d.mu.Lock()
		if conversation, ongoingConversation := d.conversations[chatID]; !ongoingConversation || conversation.IsClosed() {
			if d.IsClosed() {
				d.mu.Unlock()
				return closedErr
			}
			_, err := d.bot.Send(msg)
			d.mu.Unlock()
			return err
		}
		d.mu.Unlock()
		time.Sleep(time.Duration(time.Duration(d.singleMessageTrySendInterval) * time.Second))
	}
}

// DispatchUpdate routes an update to the target conversation, or creates a new conversation
func (d *Dispatcher) DispatchUpdate(update *tgbotapi.Update) error {
	if d.IsClosed() {
		return errors.New("Dispatcher is closed")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	chatID, err := conversation.GetUpdateChatID(update)
	if err != nil {
		return err
	}
	if d.isCancelCommand(update) {
		if conv, ok := d.conversations[chatID]; ok {
			err = conv.CancelByUser()
			delete(d.conversations, chatID)
		}
		return err
	}
	if conv, ok := d.conversations[chatID]; ok {
		return conv.PushUpdate(update)
	} else if len(d.conversations) < d.maxOpenConversations {
		conv, err := conversation.NewConversation(chatID, d.bot, d.state, &d.wg, d.conversationConfig)
		if err != nil {
			return err
		}
		d.conversations[chatID] = conv
		if conv, ok := d.conversations[chatID]; ok {
			return conv.PushUpdate(update)
		}
		return errors.New("cannot create new conversation")
	} else {
		msg := tgbotapi.NewMessage(chatID, d.toManyOpenConversationsMessage)
		_, err := d.bot.Send(msg)
		return fmt.Errorf("to many open conversations (%v)", err)
	}
}
