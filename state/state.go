package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ConversationState reflects the current state of a conversation handler with "step" granularity.
// The exact handler should be deduced from the firstMessage
type ConversationState struct {
	FirstUpdate *tgbotapi.Update `json:"first_update"` // initial message for a handler
	Step        int              `json:"step"`         // the index of the stap that should be processed
	Data        interface{}      `json:"data"`         // user-data
	ChatID      int64            `json:"chat_id"`      // chat_id of the conversation
}

type botState struct {
	ConversationStates map[int64]*ConversationState `json:"conversations"` // map of all active conversation states
	closed             bool                         // flag that indicates that state is closed and should not do any saves
	mu                 sync.RWMutex                 // mutex to synchronize read-write operations to the map of conversation states
	muIO               sync.Mutex                   // mutex to synchronize saving to a IO
	io                 StateIO                      // abstraction for reading and writing states
}

// BotState is an interface for object that records current state of all ongoing conversations
type BotState interface {
	RemoveConverastionState(conversationID int64) error // removes record about a current conversation. Should be called after a high-level handler is done
	LoadState() error                                   // Load state from a file
	Close() error                                       // Forbid furter savings

	GetConversationIDs() []int64                                        // get list of conversationsIDs, if several IDs have the same ChatID, only the latest conversationID will be listed
	GetConversatonFirstUpdate(conversationID int64) *tgbotapi.Update    // get first update of the conversation
	GetConversationStepAndData(conversationID int64) (int, interface{}) // get data and state of the conversation
	GetConversationChatID(conversationID int64) int64                   // get ChatID of the conversation

	StartConversationWithUpdate(conversationID int64, chatID int64, firstUpdate *tgbotapi.Update) error // create state for a conversation with first update
	SaveConversationStepAndData(conversationID int64, step int, data interface{}) error                 // save new conversation step and data, save all states to a file
}

// NewBotState method constructs a new BotState object
func NewBotState(io StateIO) BotState {
	return &botState{
		ConversationStates: make(map[int64]*ConversationState),
		closed:             false,
		io:                 io,
	}
}

func (bs *botState) RemoveConverastionState(converationID int64) error {
	bs.mu.Lock()
	if _, ok := bs.ConversationStates[converationID]; !ok {
		return fmt.Errorf("no record about conversation with %d in the BotState", converationID)
	}
	delete(bs.ConversationStates, converationID)
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

func (bs *botState) GetConversationIDs() []int64 {
	min := func(a, b int64) int64 {
		if a < b {
			return a
		}
		return b
	}
	max := func(a, b int64) int64 {
		if a > b {
			return a
		}
		return b
	}

	bs.mu.RLock()
	defer bs.mu.RUnlock()

	chatToConversationID := make(map[int64]int64)
	statesToDelete := make([]int64, 0)
	for conversationID, state := range bs.ConversationStates {
		if prevID, ok := chatToConversationID[state.ChatID]; ok {
			statesToDelete = append(statesToDelete, min(prevID, conversationID))
			chatToConversationID[state.ChatID] = max(prevID, conversationID)
		} else {
			chatToConversationID[state.ChatID] = conversationID
		}
	}
	for _, id := range statesToDelete {
		delete(bs.ConversationStates, id)
	}

	keys := make([]int64, len(bs.ConversationStates))
	i := 0
	for k := range bs.ConversationStates {
		keys[i] = k
		i++
	}
	return keys
}

func (bs *botState) getConversatonState(converationID int64) *ConversationState {
	bs.mu.RLock()
	if state, ok := bs.ConversationStates[converationID]; ok {
		defer bs.mu.RUnlock()
		return state
	}
	bs.mu.RUnlock()
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if state, ok := bs.ConversationStates[converationID]; ok { //now do the same thing under write Lock
		return state
	}
	bs.ConversationStates[converationID] = &ConversationState{
		FirstUpdate: nil,
		Step:        0,
		ChatID:      0,
		Data:        nil,
	}
	state := bs.ConversationStates[converationID]
	return state
}

func (bs *botState) GetConversatonFirstUpdate(converationID int64) *tgbotapi.Update {
	return bs.getConversatonState(converationID).FirstUpdate
}

func (bs *botState) GetConversationStepAndData(converationID int64) (int, interface{}) {
	state := bs.getConversatonState(converationID)
	return state.Step, state.Data
}

func (bs *botState) StartConversationWithUpdate(conversationID, chatID int64, firstUpdate *tgbotapi.Update) error {
	bs.getConversatonState(conversationID).FirstUpdate = firstUpdate
	bs.getConversatonState(conversationID).ChatID = chatID
	return bs.saveState()
}

func (bs *botState) SaveConversationStepAndData(conversationID int64, step int, data interface{}) error {
	state := bs.getConversatonState(conversationID)
	bs.muIO.Lock() //forbid saving while updating
	state.Data = data
	state.Step = step
	bs.muIO.Unlock()
	return bs.saveState()
}

func (bs *botState) GetConversationChatID(conversationID int64) int64 {
	return bs.getConversatonState(conversationID).ChatID
}
