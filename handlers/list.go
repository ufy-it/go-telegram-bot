package handlers

// allHandlerCreators is a list of pairs ("/command", handle's creator) that contains all available high-level commands
var allHandlerCreators = []struct {
	StartString    string
	HandlerCreator conversationHandlerCreatorType
}{
	{"/me", meHandlerCreator},
}
