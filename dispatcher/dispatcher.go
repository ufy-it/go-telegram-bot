package dispatcher

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	. "ufygobot/conversation"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

type Dispatcher struct {
	maxOpenConversations int
	closed               bool
	closeTimeoutSeconds  int

	mu sync.Mutex     //mutex to sync operations over the conversations map between main thread and cleaning routing
	wg sync.WaitGroup // wait group to finish all routins

	conversations              map[int64]InputConversation
	maxConversationQueue       int
	conversationTimeoutMinutes int
	conversationFinishSeconds  int

	bot *tgbotapi.BotAPI
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
			isTimeout, err := conversation.Timeout()
			if err != nil {
				log.Println("Error during cleaning conversations: %v", err)
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
	d.closed = true
	d.mu.Lock()
	defer d.mu.Unlock()
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
		return errors.New("Some threads were not finished in time")
	}
}

// NewDispatcher creates a new Dispatcher objects and starts a separate thread to clear old conversations
func NewDispatcher(maxOpenConversations int,
	maxConversationQueue int,
	clearJobInterval int,
	conversationTimeoutMinutes int,
	conversationFinishSeconds int,
	bot *tgbotapi.BotAPI) *Dispatcher {
	disp := &Dispatcher{
		conversations:              make(map[int64]InputConversation),
		maxOpenConversations:       maxOpenConversations,
		maxConversationQueue:       maxConversationQueue,
		conversationTimeoutMinutes: conversationTimeoutMinutes,
		conversationFinishSeconds:  conversationFinishSeconds,
		bot:                        bot,
		mu:                         sync.Mutex{},
		wg:                         sync.WaitGroup{},
		closed:                     false,
	}
	go disp.clearOldConversations(clearJobInterval) //start cleaning in a separate thread
	return disp
}

func (d *Dispatcher) IsClosed() bool {
	return d.closed
}

func (d *Dispatcher) DispatchMessage(message *tgbotapi.Message) error {
	if d.IsClosed() {
		return errors.New("Dispatcher is closed")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if conv, ok := d.conversations[message.Chat.ID]; ok {
		return conv.PushMessage(message)
	} else if len(d.conversations) < d.maxOpenConversations {
		d.conversations[message.Chat.ID] = NewConversation(d.maxConversationQueue,
			d.conversationTimeoutMinutes,
			d.conversationFinishSeconds,
			message.Chat.ID,
			d.bot,
			&d.wg)
		if conv, ok := d.conversations[message.Chat.ID]; ok {
			return conv.PushMessage(message)
		} else {
			return errors.New("cannot create new conversation")
		}
	} else {
		msg := tgbotapi.NewMessage(message.Chat.ID,
			"Бот сейчас сильно згружен. Повторите свой запрос позже")
		_, err := d.bot.Send(msg)
		return errors.New(fmt.Sprintf("to many open conversations (%v)", err))
	}
}
