package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ufy-it/go-telegram-bot/conversation"
	"github.com/ufy-it/go-telegram-bot/handlers"
	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// conversationWithCancel contains a conversation object and CancelFunction for the conversation context
type conversatonWithCancel struct {
	c      *conversation.BotConversation
	cancel context.CancelFunc
}

// Dispatcher is an object that manages conversations and routers input messsages to needed conversations
type Dispatcher struct {
	maxOpenConversations int

	mu sync.Mutex //mutex to sync operations over the conversations map between main thread and handler routins

	conversations      map[int64]conversatonWithCancel
	conversationConfig conversation.Config
	commandHandlers    *handlers.CommandHandlers // list of command handlers

	singleMessageTrySendInterval   int
	cancelCommand                  string // command that cancels any conversation
	toManyOpenConversationsMessage string

	bot   *tgbotapi.BotAPI
	state state.BotState

	incomeCh chan *tgbotapi.Update
}

// Config contains configuration parameters for a new dispatcher
type Config struct {
	MaxOpenConversations           int                       // the maximum number of open conversations
	SingleMessageTrySendInterval   int                       // interval between several tries of send single message to a user
	ConversationConfig             conversation.Config       // configuration for a conversation
	CancelCommand                  string                    // the command that a user can use to cancel any conversation
	ToManyOpenConversationsMessage string                    // message to send to a user if a conversation cannot be started
	Handlers                       *handlers.CommandHandlers // list of handlers for command handling
}

// start conversation handling
func (d *Dispatcher) handleConversation(ctx context.Context, conv *conversation.BotConversation) {
	var exit bool = false
	for {
		update := d.state.GetConversatonFirstUpdate(conv.ChatID())
		if update == nil { // if the conversation is not started from the state
			d.mu.Lock() // to make sure that no new messagess will arrive to this conversation
			update, exit = conv.GetFirstUpdateFromUser(ctx)
			if exit {
				delete(d.conversations, conv.ChatID())                // all new messages will go to a new go-routine
				err := d.state.RemoveConverastionState(conv.ChatID()) // this is still under the lock to prevent starting a new go-routine that uses the same state
				d.mu.Unlock()
				if err != nil {
					logger.Error("cannot remove conversation state: %v", err)
				}
				return // exit handling loop as there is no active messages, or the parent context is closed
			} else {
				d.mu.Unlock()
				err := d.state.StartConversationWithUpdate(conv.ChatID(), update)
				if err != nil {
					logger.Error("cannot add conversation to state: %v", err)
				}
			}
		}
		if update.Message == nil {
			logger.Warning("cannot process non-message high-level command in conversation with %d", conv.ChatID())
			d.state.RemoveConverastionState(conv.ChatID())
			continue
		}
		message := update.Message
		var handler handlers.Handler = nil
		if update.Message.Photo != nil && len(*update.Message.Photo) > 0 && d.commandHandlers.Image != nil {
			handler = d.commandHandlers.Image(conv, message) // use image handler
		} else {
			for _, creator := range d.commandHandlers.List { // find corresponding handler for the first message
				if creator.CommandRe.MatchString(message.Text) || creator.CommandRe.MatchString(message.Command()) {
					handler = creator.HandlerCreator(conv, message)
					break
				}
			}
		}
		if handler == nil {
			handler = d.commandHandlers.Default(conv, message) // use default handler if there is no suitable
		}
		err := handler.Execute(d.state) // execute handler
		if err != nil {
			logger.Error("in conversation with %d got error: %v", conv.ChatID(), err)
			_, err = conv.SendText(d.commandHandlers.UserErrorMessage)
			if err != nil {
				logger.Warning("cannot send error notification to %d", conv.ChatID())
			}
		}
		err = d.state.RemoveConverastionState(conv.ChatID()) // clear state for the conversation
		if err != nil {
			logger.Error("cannot remove conversation state: %v", err)
		}
	}
}

func (d *Dispatcher) dispatchUpdate(ctx context.Context, update *tgbotapi.Update) error {
	select {
	case <-ctx.Done():
		return errors.New("cannot dispatch update, context is closed")
	default:
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	chatID, err := conversation.GetUpdateChatID(update)
	if err != nil {
		return err
	}
	if conv, ok := d.conversations[chatID]; ok {
		if d.isCancelCommand(update) {
			err := conv.c.CancelByUser()
			conv.cancel()
			delete(d.conversations, chatID)
			return err
		}
		return conv.c.PushUpdate(update)
	} else if len(d.conversations) < d.maxOpenConversations {
		conv, err := conversation.NewConversation(chatID, d.bot, d.state, d.conversationConfig)
		if err != nil {
			return err
		}

		err = conv.PushUpdate(update)
		if err != nil {
			return err
		}

		convCtx, cancel := context.WithCancel(ctx)
		d.conversations[chatID] = conversatonWithCancel{conv, cancel}
		go d.handleConversation(convCtx, conv)
		return nil
	} else {
		msg := tgbotapi.NewMessage(chatID, d.toManyOpenConversationsMessage)
		_, err := d.bot.Send(msg)
		return fmt.Errorf("to many open conversations (%v)", err)
	}
}

// dispatchLoop waits for a new update in incomeCh and dispatch it
func (d *Dispatcher) dispatchLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-d.incomeCh:
			err := d.dispatchUpdate(ctx, update)
			if err != nil {
				logger.Error("cannot dispatch an update: %v", err)
			}
		}
	}
}

// NewDispatcher creates a new Dispatcher objects and starts a separate thread to clear old conversations
func NewDispatcher(ctx context.Context, config Config, bot *tgbotapi.BotAPI, stateFile string) *Dispatcher {
	d := &Dispatcher{
		conversations:                  make(map[int64]conversatonWithCancel),
		conversationConfig:             config.ConversationConfig,
		maxOpenConversations:           config.MaxOpenConversations,
		singleMessageTrySendInterval:   config.SingleMessageTrySendInterval,
		bot:                            bot,
		mu:                             sync.Mutex{},
		cancelCommand:                  config.CancelCommand,
		toManyOpenConversationsMessage: config.ToManyOpenConversationsMessage,
		state:                          state.NewBotState(),
		incomeCh:                       make(chan *tgbotapi.Update),
	}
	if stateFile != "" { //load previouse state from file
		err := d.state.LoadState(stateFile)
		if err != nil {
			logger.Warning("cannot load previouse state: %v. Will start from blank", err)
		}
	}
	// resume conversations from the state
	for _, chatID := range d.state.GetConversations() {
		conv, err := conversation.NewConversation(chatID, d.bot, d.state, d.conversationConfig)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		convCtx, cancel := context.WithCancel(ctx)
		d.conversations[chatID] = conversatonWithCancel{conv, cancel}
		if _, ok := d.conversations[chatID]; !ok {
			logger.Warning("cannot start conversation with %d from state", chatID)
		} else {
			go d.handleConversation(convCtx, conv)
		}
	}
	go d.dispatchLoop(ctx) // start the dispaching loop
	return d
}

// isCancelCommand returns true if a user entered cancel command
func (d *Dispatcher) isCancelCommand(update *tgbotapi.Update) bool {
	return update.Message != nil && update.Message.Text == d.cancelCommand
}

// SendSingleMessage waits until the conversation with chatID is closed and sends a single message from bot to a user
func (d *Dispatcher) SendSingleMessage(ctx context.Context, chatID int64, text string) error {
	closedErr := errors.New("Cannot send a message, context closed")
	msg := tgbotapi.NewMessage(chatID, text)
	for {
		select {
		case <-ctx.Done():
			return closedErr
		default:
		}
		d.mu.Lock()
		if _, ongoingConversation := d.conversations[chatID]; !ongoingConversation {
			_, err := d.bot.Send(msg)
			d.mu.Unlock()
			return err
		}
		d.mu.Unlock()
		time.Sleep(time.Duration(time.Duration(d.singleMessageTrySendInterval) * time.Second))
	}
}

// DispatchUpdate routes an update to the target conversation, or creates a new conversation
func (d *Dispatcher) DispatchUpdate(update *tgbotapi.Update) {
	d.incomeCh <- update
}
