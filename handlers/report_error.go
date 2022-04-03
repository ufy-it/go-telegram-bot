package handlers

// NewReportErrorHandler is a simple handler that returns error to the conversation handler loop
func NewReportErrorHandler(conversationID int64, err error) Handler {
	return NewStatelessHandler(
		conversationID,
		[]ConversationStep{
			{
				Action: func() (StepResult, error) {
					return ActionResultError(err)
				},
			},
		})
}
