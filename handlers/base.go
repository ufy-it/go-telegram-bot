package handlers

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/ufy-it/go-telegram-bot/handlers/readers"
	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// Handler is an interface for a conversation handler
type Handler interface {
	Execute(bState state.BotState) error
}

// HandlerCreatorType is a type of a functor that can create a handler for a conversation
type HandlerCreatorType func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) Handler

// CommandHandler is a struct that contains regexp to determine start command for the handler and function-creator to build the handler
type CommandHandler struct {
	CommandRe      *regexp.Regexp     // regular expression that specifies the command
	HandlerCreator HandlerCreatorType // function to create a handler for the command
}

// CommandHandlers is a structure that contains list of command handlers
// and default handler that processes commands that has-not been mutched by any of command-handlers
type CommandHandlers struct {
	Default          HandlerCreatorType // Default handler will handle any command that does
	Image            HandlerCreatorType // Handler for the image input
	List             []CommandHandler   // List of command handlers
	UserErrorMessage string             // Message to display to user in case of error in a handler
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
type ConversationStepPerformer func(conversation readers.BotConversation) (StepResult, error)

// ConversationStep represents a step of a conversation
type ConversationStep struct {
	Name   string
	Action ConversationStepPerformer
}

// UserDataReader is a function that provides uer-data for serialization
type UserDataReader func() interface{}

// UserDataWriter is a function that fills user-data from []byte
type UserDataWriter func(data interface{}) error

// StandardHandler is a struct that represents a conversation handler
type StandardHandler struct {
	Conversation readers.BotConversation // pointer to the object that handles conversation between bot and user
	FirstMessage *tgbotapi.Message       // first message, that started conversation with a particular handler
	Steps        []ConversationStep      // conversation steps
	GetUserData  UserDataReader          // function to get user data for serialization
	SetUserData  UserDataWriter          // function to set user-data in case of resumed conversation
}

// Execute processes the conversation between a user and a handler
func (h *StandardHandler) Execute(bState state.BotState) error {
	if len(h.Steps) == 0 {
		return nil
	}
	if bState == nil {
		return errors.New("Handler started with nil bot state")
	}
	step, data := bState.GetConversationStepAndData(h.Conversation.ChatID())
	if step < 0 || step >= len(h.Steps) {
		return fmt.Errorf("step index from state (%d) is out of range", step)
	}
	if h.Conversation == nil {
		return errors.New("Handler started with nil conversation")
	}
	if h.GetUserData == nil {
		return errors.New("Handler is incomplete, GetUserData is nil")
	}
	if h.SetUserData == nil {
		return errors.New("Handler is incomplete, SetUserData is nil")
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
			err := bState.SaveConversationStepAndData(h.Conversation.ChatID(), step, h.GetUserData())
			if err != nil {
				logger.Error("error saving conversation state: %v", err)
			}
		}
		resumed = false
		if h.Steps[step].Action == nil {
			return fmt.Errorf("action for %d'th step is not defined", step)
		}
		result, err := h.Steps[step].Action(h.Conversation) // run a conversation step
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
				return fmt.Errorf("Cannot find step with name '%s'", result.Name)
			}
			step = nextStep
		}
	}
}
