package handlers

import "ufygobot/handlers/readers"

// ReportErrorHandler is a simple handler that returns error to the conversation handler loop
func ReportErrorHandler(conversation readers.BotConversation, err error) Handler {
	return &StandardHandler{
		Conversation: conversation,
		GetUserData:  func() interface{} { return nil },
		SetUserData:  func(data interface{}) error { return nil },
		Steps: []ConversationStep{
			{
				Action: func(conversation readers.BotConversation) (StepResult, error) {
					return ActionResultError(err)
				},
			},
		},
	}
}
