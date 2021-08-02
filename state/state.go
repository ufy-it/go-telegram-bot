package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
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
	muFile             sync.Mutex                   // mutex to synchronize saving to a file
}

// BotState is an interface for object that records current state of all ongoing conversations
type BotState interface {
	RemoveConverastionState(chatID int64) error // removes record about a current conversation. Should be called after a high-level handler is done
	LoadState(filename string) error            // Load state from a file
	Close() error                               // Forbid furter savings

	GetConversations() []int64                                  // get list of conversations
	GetConversatonFirstUpdate(chatID int64) *tgbotapi.Update    // get first update of the conversation
	GetConversationStepAndData(chatID int64) (int, interface{}) // get data and state of the conversation

	StartConversationWithUpdate(chatID int64, firstUpdate *tgbotapi.Update) error // create state for a conversation with first update
	SaveConversationStepAndData(chatID int64, step int, data interface{}) error   // save new conversation step and data, save all states to a file
}

// NewBotState method constructs a new BotState object
func NewBotState() BotState {
	return &botState{
		ConversationStates: make(map[int64]*ConversationState),
		closed:             false,
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

func (bs *botState) LoadState(filename string) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.filename = filename
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cannot read state file %s: %v", filename, err)
	}
	err = json.Unmarshal([]byte(file), bs)
	if err != nil {
		return fmt.Errorf("cannot load state from file %s: %v", filename, err)
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
	bs.muFile.Lock()
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	defer bs.muFile.Unlock()

	if bs.filename == "" {
		return errors.New("cannot save state: filename is not set")
	}
	content, err := json.MarshalIndent(bs, "", " ")
	if err != nil {
		return fmt.Errorf("cannot marshal state to json: %v", err)
	}
	tempFile, err := ioutil.TempFile(".", "persistant")
	if err != nil {
		return fmt.Errorf("cannot ctreate temp file: %v", err)
	}
	_, err = tempFile.Write(content)
	if err != nil {
		return fmt.Errorf("cannot write state to a temp file: %v", err)
	}
	err = tempFile.Close()
	if err != nil {
		return fmt.Errorf("cannot write state to a temp file: %v", err)
	}

	err = os.Rename(tempFile.Name(), bs.filename)
	if err != nil {
		return fmt.Errorf("cannot rename temp file to the target file %s: %v", bs.filename, err)
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
	state, _ := bs.ConversationStates[chatID]
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
	bs.muFile.Lock() //forbid saving while updating
	state.Data = data
	state.Step = step
	bs.muFile.Unlock()
	return bs.saveState()
}
