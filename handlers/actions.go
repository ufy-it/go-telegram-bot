package handlers

// StepActionResultHelper is a type of helper function that returns next step transmision and error
type StepActionResultHelper func() (StepResult, error)

// ConditionalActionResult returns one of two ActionResultHelpers depending on condition
func ConditionalActionResult(condition bool, resultIfTrue, resultIfFalse StepActionResultHelper) StepActionResultHelper {
	if condition {
		return resultIfTrue
	}
	return resultIfFalse
}

// ActionResultWithError overrides error of result() with err
func ActionResultWithError(result StepActionResultHelper, err error) (StepResult, error) {
	r, _ := result()
	return r, err
}

// ActionResultError returns transfer to the conversation end and err as error
func ActionResultError(err error) (StepResult, error) {
	return ActionResultWithError(EndConversation, err)
}

// NextStep returns transfer to the next step and nil error
func NextStep() (StepResult, error) {
	return StepResult{
		Action: Next,
	}, nil
}

// PrevStep returns transfer to the previous step and nil error
func PrevStep() (StepResult, error) {
	return StepResult{
		Action: Prev,
	}, nil
}

// EndConversation returns transfer to the conversation end and nil error
func EndConversation() (StepResult, error) {
	return StepResult{
		Action: End,
	}, nil
}

// RepeatStep returns transfer to the current step and nil error
func RepeatStep() (StepResult, error) {
	return StepResult{
		Action: Repeat,
	}, nil
}

// RepeatCommand returns transfer to the first step of the conversation and nil error
func RepeatCommand() (StepResult, error) {
	return StepResult{
		Action: Begin,
	}, nil
}

// GoToCommand returns transfer to a custom step (by name) and nil error
func GoToCommand(command string) (StepResult, error) {
	return StepResult{
		Name:   command,
		Action: Custom,
	}, nil
}

func GoToCommandHelper(command string) StepActionResultHelper {
	return func() (StepResult, error) {
		return GoToCommand(command)
	}
}

// CloseCommand returns transfer to the close state an nil error
func CloseCommand() (StepResult, error) {
	return StepResult{
		Action: Close,
	}, nil
}
