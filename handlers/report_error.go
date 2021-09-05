package handlers

// NewReportErrorHandler is a simple handler that returns error to the conversation handler loop
func NewReportErrorHandler(chatID int64, err error) Handler {
	return NewStatelessHandler(
		chatID, // not a real chatID
		[]ConversationStep{
			{
				Action: func() (StepResult, error) {
					return ActionResultError(err)
				},
			},
		})
}
