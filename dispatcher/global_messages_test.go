package dispatcher_test

import (
	"testing"

	. "github.com/ufy-it/go-telegram-bot/dispatcher"
)

func TestEmptyGlobalMessageFunc(t *testing.T) {
	var messageIDs []MessageIDType = []MessageIDType{
		TooManyMessages,
		TooManyConversations,
		UserError,
		ConversationClosedByBot,
		ConversationClosedByUser,
		ConversationEnded,
	}
	for _, id := range messageIDs {
		if EmptyTechnicalMessageFunc(777, id) != "" {
			t.Errorf("non-empty string generated for message ID %d", id)
		}
	}
}
