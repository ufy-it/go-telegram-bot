package buttons

const (
	// NavigationBack is data for the back navigation button (that should be used for returnng on the previouse step)
	NavigationBack = "navigation_back"
	// NavigationAbort is data for the abort navigation button (that should be used for canceling conversation)
	NavigationAbort = "navigation_abort"
	// NavigationAccept is data for the accept navigation button (that should be used for confirming an action)
	NavigationAccept = "navigation_accept"
	// NavigationSkip is data for the skip navigation button (that should be used for skipping an action)
	NavigationSkip = "navigation_skip"
)

// NewBackButton creates a new Back navigation button
func NewBackButton(text string) Button {
	return NewButton(text, NavigationBack)
}

// NewAbortButton creates a new Abort navigation button
func NewAbortButton(text string) Button {
	return NewButton(text, NavigationAbort)
}

// NewAcceptButton creates a new Accept navigation button
func NewAcceptButton(text string) Button {
	return NewButton(text, NavigationAccept)
}

// NewSkipButton creates a new Skip navigation button
func NewSkipButton(text string) Button {
	return NewButton(text, NavigationSkip)
}
