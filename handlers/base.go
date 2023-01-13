package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ufy-it/go-telegram-bot/handlers/readers"
	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"
)

type HandlerContextVariables string

const (
	FirstUpdateVariable HandlerContextVariables = "first_update"
)

// GetFirstUpdate returns the first update for the conversation from the context
func GetFirstUpdate(ctx context.Context) (*tgbotapi.Update, error) {
	if ctx.Value(FirstUpdateVariable) == nil {
		return nil, errors.New("no first update")
	}
	return ctx.Value(FirstUpdateVariable).(*tgbotapi.Update), nil
}

// Handler is an interface for a conversation handler
type Handler interface {
	Execute(conversationID int64, bState state.BotState) error
}

// HandlerCreatorType is a type of a functor that can create a handler for a conversation
type HandlerCreatorType func(ctx context.Context, conversation readers.BotConversation) Handler

// CommandSelectorType is a type of functor that should tell whether the handler matches the command
type CommandSelectorType func(ctx context.Context, firstUpdate *tgbotapi.Update) bool

// CommandHandler is a struct that contains regexp to determine start command for the handler and function-creator to build the handler
type CommandHandler struct {
	CommandSelector CommandSelectorType // selector for the command
	HandlerCreator  HandlerCreatorType  // function to create a handler for the command
}

// CommandHandlers is a structure that contains list of command handlers
// and default handler that processes commands that has-not been mutched by any of command-handlers
type CommandHandlers struct {
	Default HandlerCreatorType // Default handler will handle any command that does
	List    []CommandHandler   // List of command handlers
}

// StepResultAction is a type for a handler's step result
type StepResultAction int

// possible step's reults tell a handler what step to process next
const (
	Next StepResultAction = iota
	Repeat
	Prev
	End
	Begin
	Custom
	Close
)

// StepResult result of a step action
type StepResult struct {
	Name   string           // name of the next step to run. would be used if action is Custom
	Action StepResultAction // action to perform as a result of a step
}

// ConversationStepPerformer is a function that performs a step in a conversaton
type ConversationStepPerformer func() (StepResult, error)

// ConversationStep represents a step of a conversation
type ConversationStep struct {
	Name   string
	Action ConversationStepPerformer
}

// userDataReader is a function that provides uer-data for serialization
type userDataReader func() interface{}

// userDataWriter is a function that fills user-data from []byte
type userDataWriter func(data interface{}) error

// standardHandler is a struct that represents a conversation handler
type standardHandler struct {
	Steps       []ConversationStep // conversation steps
	GetUserData userDataReader     // function to get user data for serialization
	SetUserData userDataWriter     // function to set user-data in case of resumed conversation
}

// NewStatefulHandler generates a handler for chatID with non-nil userData and steps
// userData should be a pointer to a json-serializible struct
func NewStatefulHandler(userData interface{}, steps []ConversationStep) Handler {
	return &standardHandler{
		Steps:       steps,
		GetUserData: func() interface{} { return userData },
		SetUserData: func(data interface{}) error {
			bytes, err := json.Marshal(data)
			if err != nil {
				return err
			}
			return json.Unmarshal(bytes, userData)
		},
	}
}

// NewStatelessHandler generates a handler for chatID with no state data (other than the index of current step)
func NewStatelessHandler(steps []ConversationStep) Handler {
	return &standardHandler{
		Steps:       steps,
		GetUserData: func() interface{} { return nil },
		SetUserData: func(data interface{}) error { return nil },
	}
}

// Execute processes the conversation between a user and a handler
func (h *standardHandler) Execute(conversationID int64, bState state.BotState) error {
	if len(h.Steps) == 0 {
		return nil
	}
	if bState == nil {
		return errors.New("handler started with nil bot state")
	}
	step, data := bState.GetConversationStepAndData(conversationID)
	if step < 0 || step >= len(h.Steps) {
		return fmt.Errorf("step index from state (%d) is out of range", step)
	}
	if h.GetUserData == nil {
		return errors.New("handler is incomplete, GetUserData is nil")
	}
	if h.SetUserData == nil {
		return errors.New("handler is incomplete, SetUserData is nil")
	}
	resumed := false
	if data != nil {
		err := h.SetUserData(data)
		if err != nil {
			return fmt.Errorf("failed to resume user-data from the state: %v", err)
		}
		resumed = true
	}
	for {
		if !resumed {
			err := bState.SaveConversationStepAndData(conversationID, step, h.GetUserData())
			if err != nil {
				logger.Error("error saving conversation state: %v", err)
			}
		}
		resumed = false
		if h.Steps[step].Action == nil {
			return fmt.Errorf("action for %d'th step is not defined", step)
		}
		result, err := h.Steps[step].Action() // run a conversation step
		if err != nil {
			return err
		}
		switch result.Action {
		case End:
			return nil
		case Close:
			return errors.New("conversation closed")
		case Next:
			step++
			if step >= len(h.Steps) {
				return errors.New("cannot execute step with index out of range")
			}
		case Repeat:
			continue
		case Prev:
			step--
			if step < 0 {
				return errors.New("cannot execute step with index < 0")
			}
		case Begin:
			step = 0
		case Custom:
			nextStep := -1
			for idx, step := range h.Steps {
				if step.Name == result.Name {
					nextStep = idx
					break
				}
			}
			if nextStep < 0 {
				return fmt.Errorf("cannot find step with name '%s'", result.Name)
			}
			step = nextStep
		}
	}
}
