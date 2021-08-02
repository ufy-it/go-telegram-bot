package handlers

import (
	"errors"
	"fmt"
	"regexp"

	"ufygobot/handlers/readers"
	"ufygobot/logger"
	"ufygobot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// HandleConversation handles user messages until the conversation is closed
func HandleConversation(conversation readers.BotConversation, bState state.BotState, config *CommandHandlers) {
	var exit bool = false
	if config == nil {
		logger.Error("config object is nil")
		return
	}
	if config.Default == nil {
		logger.Error("default handler is nil")
		return
	}
	if config.List == nil {
		logger.Error("list of handlers is nil")
		return
	}
	if conversation == nil {
		logger.Error("conversation object is nil")
		return
	}
	if bState == nil {
		logger.Error("state object is nil")
		return
	}
	for {
		if conversation.IsClosed() {
			return
		}
		update := bState.GetConversatonFirstUpdate(conversation.ChatID())
		if update == nil { // regular handling of incomming messages
			update, exit = conversation.GetUpdateFromUser()
			if !exit {
				err := bState.StartConversationWithUpdate(conversation.ChatID(), update)
				if err != nil {
					logger.Error("cannot add conversation to state: %v", err)
				}
			}
		}
		if exit {
			err := bState.RemoveConverastionState(conversation.ChatID())
			if err != nil {
				logger.Error("cannot remove conversation state: %v", err)
			}
			return //finish the conversation
		}
		if update.Message == nil {
			logger.Warning("cannot process non-message high-level command in conversation with %d", conversation.ChatID())
			bState.RemoveConverastionState(conversation.ChatID())
			continue
		}
		message := update.Message
		var handler Handler = nil
		if update.Message.Photo != nil && len(*update.Message.Photo) > 0 && config.Image != nil {
			handler = config.Image(conversation, message) // use image handler
		} else {
			for _, creator := range config.List { // find corresponding handler for the first message
				if creator.CommandRe.MatchString(message.Text) || creator.CommandRe.MatchString(message.Command()) {
					handler = creator.HandlerCreator(conversation, message)
					break
				}
			}
		}
		if handler == nil {
			handler = config.Default(conversation, message) // use default handler if there is no suitable
		}
		err := handler.Execute(bState) // execute handler
		if err != nil {
			logger.Error("in conversation with %d got error: %v", conversation.ChatID(), err)
			if !conversation.IsClosed() {
				_, err = conversation.SendText(config.UserErrorMessage)
				if err != nil {
					logger.Warning("cannot send error notification to %d", conversation.ChatID())
				}
			}
		}
		err = bState.RemoveConverastionState(conversation.ChatID()) // clear state for the conversation
		if err != nil {
			logger.Error("cannot remove conversation state: %v", err)
		}
		conversation.FinishConversation() // suggest to finish the conversation
	}
}

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
