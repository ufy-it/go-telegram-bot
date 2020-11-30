package handlers

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"ufygobot/handlers/readers"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// HandleConversation handles user messages until the conversation is closed
func HandleConversation(conversation readers.BotConversation) {
	for {
		if conversation.IsClosed() {
			return
		}
		message, exit := conversation.ReadMessage()
		if exit {
			return //finish the conversation
		}
		var handler convrsationHandler = nil
		for _, creator := range allHandlerCreators {
			if strings.HasPrefix(message.Text, creator.StartString) {
				handler = creator.HandlerCreator(conversation, message)
				break
			}
		}
		if handler == nil {
			handler = defaultHandlerCreator(conversation, message)
		}
		err := handler.Execute()
		if err != nil {
			log.Printf("Error: %v", err)
		}
		conversation.Finish()
	}
}

type convrsationHandler interface {
	Execute() error
}

type conversationHandlerCreatorType func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) convrsationHandler

type handlerAction int

const (
	Next handlerAction = iota
	Repeat
	Prev
	End
	Begin
	Custom
	Close
)

type stepActionResult struct {
	NextCommandName string
	NextAction      handlerAction
}

type conversationStepAction func(conversation readers.BotConversation) (stepActionResult, error)

type conversationStep struct {
	name   string
	action conversationStepAction
}

type baseHandler struct {
	conversation readers.BotConversation // pointer to the object that handles conversaation between bot and user
	firstMessage *tgbotapi.Message       // first message, that started conversation with a particular handler
	steps        []conversationStep      // conversation steps
}

// Execute() processes the conversation between a user and a handler
func (h *baseHandler) Execute() error {
	if len(h.steps) == 0 {
		return nil
	}
	index := 0
	for {
		result, err := h.steps[index].action(h.conversation)
		if err != nil {
			return err
		}
		switch result.NextAction {
		case End:
			return nil
		case Close:
			return errors.New("conversation closed")
		case Next:
			index++
			if index >= len(h.steps) {
				return errors.New("cannot execute step with index out of range")
			}
		case Repeat:
			continue
		case Prev:
			index--
			if index < 0 {
				return errors.New("cannot execute step with index -1")
			}
		case Begin:
			index = 0
		case Custom:
			nextIndex := -1
			for idx, step := range h.steps {
				if step.name == result.NextCommandName {
					nextIndex = idx
					break
				}
			}
			if nextIndex < 0 {
				return fmt.Errorf("Cannot find step with name '%s'", result.NextCommandName)
			}
			index = nextIndex
		}
	}
}
