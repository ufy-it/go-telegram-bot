package dispatcher

// MessageIDType is an enum type for global message ID
type MessageIDType int

const (
	TooManyMessages          MessageIDType = iota // a message to send to user in case if there are to many unprocessed messages from them
	TooManyConversations                          // a message to send to user in case if there are to many open conversations
	UserError                                     // a message to send to user in case of an error in a handler
	ConversationClosedByBot                       // a message to send to user if a conversation was cancelled by the bot
	ConversationClosedByUser                      // a message to send to user if them switched to another global command
	ConversationEnded                             // a message to send to user after conversation was ended successfully
)

// GlobalMessageFuncType type of a function that should return global messages for a dedicated user
type GlobalMessageFuncType func(chatID int64, messageID MessageIDType) string

// EmptyGlobalMessageFunc returns empty string for all messages
func EmptyGlobalMessageFunc(chatID int64, messageID MessageIDType) string {
	return ""
}
