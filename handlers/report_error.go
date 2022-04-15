package handlers

// NewReportErrorHandler is a simple handler that returns error to the conversation handler loop
func NewReportErrorHandler(err error) Handler {
	return NewStatelessHandler(
		[]ConversationStep{
			{
				Action: func() (StepResult, error) {
					return ActionResultError(err)
				},
			},
		})
}
