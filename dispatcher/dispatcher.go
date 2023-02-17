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

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

	conversations          map[int64]conversatonWithCancel
	chatIDtoConversationID map[int64]int64 // map from chatID to conversationID
	conversationConfig     conversation.Config
	commandHandlers        *handlers.CommandHandlers // list of command handlers
	globalCommandHandlers  []handlers.CommandHandler // list of commands that can be started at any point of conversation

	singleMessageTrySendInterval int

	globalMessagesFunc TechnicalMessageFuncType
	globalKeyboardFunc GlobalKeyboardFuncType

	bot   *tgbotapi.BotAPI
	state state.BotState

	incomeCh chan *tgbotapi.Update
}

// start conversation handling
func (d *Dispatcher) handleConversation(ctx context.Context, conv *conversation.BotConversation) {
	var exit bool = false
	for {
		update := d.state.GetConversatonFirstUpdate(conv.ConversationID())
		if update == nil { // if the conversation is not started from the state
			d.mu.Lock() // to make sure that no new messagess will arrive to this conversation
			update, exit = conv.GetFirstUpdateFromUser(ctx)
			if exit {
				delete(d.conversations, conv.ConversationID())                                                    // all new messages will go to a new go-routine
				if convID, ok := d.chatIDtoConversationID[conv.ChatID()]; ok && convID == conv.ConversationID() { // remove chatID to conversationID mapping
					delete(d.chatIDtoConversationID, conv.ChatID())
				}
				err := d.state.RemoveConverastionState(conv.ConversationID()) // this is still under the lock to prevent starting a new go-routine that uses the same state
				d.mu.Unlock()
				if err != nil {
					logger.Error("cannot remove conversation state: %v", err)
				}
				return // exit handling loop as there is no active messages, or the parent context is closed
			} else {
				d.mu.Unlock()
				err := d.state.StartConversationWithUpdate(conv.ConversationID(), conv.ChatID(), update)
				if err != nil {
					logger.Error("cannot add conversation to state: %v", err)
				}
				if update.CallbackQuery != nil && update.CallbackQuery.ID != "" {
					err = conv.AnswerButton(update.CallbackQuery.ID)
					if err != nil {
						logger.Error("cannot answer button: %v", err)
					}
				}
			}
		}

		selectHandlerFromList := func(list []handlers.CommandHandler, firstUpdate *tgbotapi.Update) handlers.Handler {
			for _, creator := range list {
				if creator.CommandSelector(ctx, update) {
					return creator.HandlerCreator(context.WithValue(ctx, handlers.FirstUpdateVariable, update), conv)
				}
			}
			return nil
		}

		handler := selectHandlerFromList(d.globalCommandHandlers, update)
		if handler == nil {
			handler = selectHandlerFromList(d.commandHandlers.List, update)
		}
		if handler == nil {
			handler = d.commandHandlers.Default(context.WithValue(ctx, handlers.FirstUpdateVariable, update), conv) // use default handler if there is no suitable
		}

		err := handler.Execute(conv.ConversationID(), d.state) // execute handler
		if err != nil {
			logger.Error("in conversation with %d got error: %v", conv.ChatID(), err)
			if !conv.IsCanceled() {
				err = d.sendGlobalMessage(conv.ChatID(), UserError)
				if err != nil {
					logger.Warning("cannot send error notification to %d", conv.ChatID())
				}
			}
		}
		err = d.state.RemoveConverastionState(conv.ConversationID()) // clear state for the conversation
		if err != nil {
			logger.Error("cannot remove conversation state: %v", err)
		}
		if !conv.IsCanceled() {
			err = d.sendGlobalMessage(conv.ChatID(), ConversationEnded)
			if err != nil {
				logger.Error("cannot send conversation ended message: %v", err)
			}
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

	startNewConversation := func() error {
		conv, err := conversation.NewConversation(chatID,
			d.bot,
			d.state,
			d.generateSpecialMesaageFunc(chatID, TooManyMessages),
			d.generateSpecialMesaageFunc(chatID, ConversationClosedByBot),
			d.generateSpecialMesaageFunc(chatID, ConversationClosedByUser),
			d.generateGlobalKeyboardFunc(chatID),
			d.conversationConfig)
		if err != nil {
			return err
		}

		err = conv.PushUpdate(update)
		if err != nil {
			return err
		}

		convCtx, cancel := context.WithCancel(ctx)
		d.conversations[conv.ConversationID()] = conversatonWithCancel{conv, cancel}
		d.chatIDtoConversationID[chatID] = conv.ConversationID()
		go d.handleConversation(convCtx, conv)
		return nil
	}
	if convID, ok := d.chatIDtoConversationID[chatID]; ok {
		if conv, ok := d.conversations[convID]; ok {
			if d.isGlobalCommand(ctx, update) {
				err := conv.c.CancelByUser()
				conv.cancel()
				if err != nil {
					return err
				}
				return startNewConversation()
			}
			return conv.c.PushUpdate(update)
		}
	}
	if len(d.conversations) < d.maxOpenConversations {
		return startNewConversation()
	} else {
		err := d.sendGlobalMessage(chatID, TooManyConversations)
		return fmt.Errorf("to many open conversations (%v)", err)
	}
}

// sendGlobalMessage sends a message dirrectly through API
// this message does not handled by a conversation object
func (d *Dispatcher) sendGlobalMessage(chatID int64, messageID MessageIDType) error {
	text := d.globalMessagesFunc(chatID, messageID)
	if text == "" {
		return nil // skip empty message
	}
	msg := tgbotapi.NewMessage(chatID, text)
	if d.globalKeyboardFunc != nil {
		msg.ReplyMarkup = d.globalKeyboardFunc(chatID)
	}
	_, err := d.bot.Send(msg)
	return err
}

// generateSpecialMesaageFunc creates a function that sends selected general message to the specified chat
func (d *Dispatcher) generateSpecialMesaageFunc(chatID int64, messageID MessageIDType) conversation.SpecialMessageFuncType {
	return func() error {
		return d.sendGlobalMessage(chatID, messageID)
	}
}

// generateGlobalKeyboardFunc creates a function that generates gloabal keyboard for the specified chat
func (d *Dispatcher) generateGlobalKeyboardFunc(chatID int64) conversation.GlobalKeyboardFuncType {
	if d.globalKeyboardFunc == nil {
		return func() interface{} {
			return nil
		}
	}
	return func() interface{} {
		return d.globalKeyboardFunc(chatID)
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
func NewDispatcher(ctx context.Context, config Config, bot *tgbotapi.BotAPI, stateIO state.StateIO) (*Dispatcher, error) {
	d := &Dispatcher{
		conversations:                make(map[int64]conversatonWithCancel),
		chatIDtoConversationID:       make(map[int64]int64),
		conversationConfig:           config.ConversationConfig,
		maxOpenConversations:         config.MaxOpenConversations,
		singleMessageTrySendInterval: config.SingleMessageTrySendInterval,
		bot:                          bot,
		mu:                           sync.Mutex{},
		state:                        state.NewBotState(stateIO),
		incomeCh:                     make(chan *tgbotapi.Update),
		commandHandlers:              config.Handlers,
		globalCommandHandlers:        config.GlobalHandlers,
		globalMessagesFunc:           config.TechnicalMessageFunc,
		globalKeyboardFunc:           config.GloabalKeyboardFunc,
	}
	if d.commandHandlers == nil {
		return nil, errors.New("handlers cannot be nil")
	}

	if d.globalMessagesFunc == nil {
		d.globalMessagesFunc = EmptyTechnicalMessageFunc
		logger.Warning("GlobalMessageFunc was not set, will use EmptyGlobalMessageFunc")
	}

	err := d.state.LoadState()
	if err != nil {
		logger.Warning("cannot load previouse state: %v, will start from blank", err)
	}

	// resume conversations from the state
	for _, conversationID := range d.state.GetConversationIDs() {
		chatID := d.state.GetConversationChatID(conversationID)
		conv, err := conversation.NewConversationWithID(
			conversationID,
			chatID,
			d.bot,
			d.state,
			d.generateSpecialMesaageFunc(chatID, TooManyConversations),
			d.generateSpecialMesaageFunc(chatID, ConversationClosedByBot),
			d.generateSpecialMesaageFunc(chatID, ConversationClosedByUser),
			d.generateGlobalKeyboardFunc(chatID),
			d.conversationConfig)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		convCtx, cancel := context.WithCancel(ctx)
		d.conversations[conversationID] = conversatonWithCancel{conv, cancel}
		d.chatIDtoConversationID[chatID] = conversationID
		if _, ok := d.conversations[conversationID]; !ok {
			logger.Warning("cannot start conversation with %d from state", conversationID)
		} else {
			go d.handleConversation(convCtx, conv)
		}
	}
	go d.dispatchLoop(ctx) // start the dispaching loop
	return d, nil
}

// isGlobalCommand returns true if a user started a global command
func (d *Dispatcher) isGlobalCommand(ctx context.Context, update *tgbotapi.Update) bool {
	for _, creator := range d.globalCommandHandlers {
		if creator.CommandSelector(ctx, update) {
			return true
		}
	}
	return false
}

// SendSingleMessage waits until the conversation with chatID is closed and sends a single message from bot to a user
func (d *Dispatcher) SendSingleMessage(ctx context.Context, chatID int64, text string) error {
	closedErr := errors.New("cannot send a message, context closed")
	msg := tgbotapi.NewMessage(chatID, text)
	for {
		select {
		case <-ctx.Done():
			return closedErr
		default:
		}
		d.mu.Lock()
		if _, ongoingConversation := d.chatIDtoConversationID[chatID]; !ongoingConversation {
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
