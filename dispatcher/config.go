package dispatcher

import (
	"github.com/ufy-it/go-telegram-bot/conversation"
	"github.com/ufy-it/go-telegram-bot/handlers"
)

// Config contains configuration parameters for a new dispatcher
type Config struct {
	MaxOpenConversations         int                       // the maximum number of open conversations
	SingleMessageTrySendInterval int                       // interval between several tries of send single message to a user
	ConversationConfig           conversation.Config       // configuration for a conversation
	Handlers                     *handlers.CommandHandlers // list of handlers for command handling
	GlobalHandlers               []handlers.CommandHandler // list of handlers that can be started at any point of conversation
	GlobalMessageFunc            GlobalMessageFuncType     // Function that provides global messages that should be send to a user in special cases
	GloabalKeyboardFunc          GlobalKeyboardFuncType    // Function that provides global keyboard that would be attached to each global message
}
