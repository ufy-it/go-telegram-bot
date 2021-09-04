package handlers

// ReportErrorHandler is a simple handler that returns error to the conversation handler loop
func ReportErrorHandler(err error) Handler {
	return &StandardHandler{
		ChatID:      0, // do not need it
		GetUserData: func() interface{} { return nil },
		SetUserData: func(data interface{}) error { return nil },
		Steps: []ConversationStep{
			{
				Action: func() (StepResult, error) {
					return ActionResultError(err)
				},
			},
		},
	}
}
