package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// ConversationState reflects the current state of a conversation handler with "step" granularity.
// The exact handler should be deduced from the firstMessage
type ConversationState struct {
	FirstUpdate *tgbotapi.Update `json:"first_update"` // initial message for a handler
	Step        int              `json:"step"`         // the index of the stap that should be processed
	Data        interface{}      `json:"data"`         // user-data
}

type botState struct {
	ConversationStates map[int64]*ConversationState `json:"conversations"` // map of all active conversation states
	filename           string                       // file to store state on disk
	closed             bool                         // flag that indicates that state is closed and should not do any saves
	mu                 sync.RWMutex                 // mutex to synchronize read-write operations to the map of conversation states
	muIO               sync.Mutex                   // mutex to synchronize saving to a IO
	io                 StateIO                      // abstraction for reading and writing states
}

// BotState is an interface for object that records current state of all ongoing conversations
type BotState interface {
	RemoveConverastionState(chatID int64) error // removes record about a current conversation. Should be called after a high-level handler is done
	LoadState() error                           // Load state from a file
	Close() error                               // Forbid furter savings

	GetConversations() []int64                                  // get list of conversations
	GetConversatonFirstUpdate(chatID int64) *tgbotapi.Update    // get first update of the conversation
	GetConversationStepAndData(chatID int64) (int, interface{}) // get data and state of the conversation

	StartConversationWithUpdate(chatID int64, firstUpdate *tgbotapi.Update) error // create state for a conversation with first update
	SaveConversationStepAndData(chatID int64, step int, data interface{}) error   // save new conversation step and data, save all states to a file
}

// NewBotState method constructs a new BotState object
func NewBotState(io StateIO) BotState {
	return &botState{
		ConversationStates: make(map[int64]*ConversationState),
		closed:             false,
		io:                 io,
	}
}

func (bs *botState) RemoveConverastionState(chatID int64) error {
	bs.mu.Lock()
	if _, ok := bs.ConversationStates[chatID]; !ok {
		return fmt.Errorf("no record about conversation with %d in the BotState", chatID)
	}
	delete(bs.ConversationStates, chatID)
	bs.mu.Unlock()
	return bs.saveState()
}

func (bs *botState) LoadState() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if bs.io == nil {
		return errors.New("state io is not defined")
	}
	file, err := bs.io.Load()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(file), bs)
	if err != nil {
		return fmt.Errorf("cannot load state: %v", err)
	}
	return nil
}

func (bs *botState) Close() error {
	if bs.closed {
		return errors.New("state is already closed")
	}
	bs.closed = true
	return nil
}

// saveState first saves the state to temporary file and then renames the file to a target filename
func (bs *botState) saveState() error {
	if bs.closed {
		return errors.New("the state is closed for saving")
	}
	bs.muIO.Lock()
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	defer bs.muIO.Unlock()

	content, err := json.MarshalIndent(bs, "", " ")
	if err != nil {
		return fmt.Errorf("cannot marshal state to json: %v", err)
	}

	err = bs.io.Save(content)
	if err != nil {
		return fmt.Errorf("cannot save state to io: %v", err)
	}

	return nil
}

func (bs *botState) GetConversations() []int64 {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	keys := make([]int64, len(bs.ConversationStates))
	i := 0
	for k := range bs.ConversationStates {
		keys[i] = k
		i++
	}
	return keys
}

func (bs *botState) getConversatonState(chatID int64) *ConversationState {
	bs.mu.RLock()
	if state, ok := bs.ConversationStates[chatID]; ok {
		defer bs.mu.RUnlock()
		return state
	}
	bs.mu.RUnlock()
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if state, ok := bs.ConversationStates[chatID]; ok { //now do the same thing under write Lock
		return state
	}
	bs.ConversationStates[chatID] = &ConversationState{
		FirstUpdate: nil,
		Step:        0,
		Data:        nil,
	}
	state := bs.ConversationStates[chatID]
	return state
}

func (bs *botState) GetConversatonFirstUpdate(chatID int64) *tgbotapi.Update {
	return bs.getConversatonState(chatID).FirstUpdate
}

func (bs *botState) GetConversationStepAndData(chatID int64) (int, interface{}) {
	state := bs.getConversatonState(chatID)
	return state.Step, state.Data
}

func (bs *botState) StartConversationWithUpdate(chatID int64, firstUpdate *tgbotapi.Update) error {
	bs.getConversatonState(chatID).FirstUpdate = firstUpdate
	return bs.saveState()
}

func (bs *botState) SaveConversationStepAndData(chatID int64, step int, data interface{}) error {
	state := bs.getConversatonState(chatID)
	bs.muIO.Lock() //forbid saving while updating
	state.Data = data
	state.Step = step
	bs.muIO.Unlock()
	return bs.saveState()
}
