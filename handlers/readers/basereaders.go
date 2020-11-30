package readers

import tgbotapi "github.com/Syfaro/telegram-bot-api"

type BotConversation interface {
	IsClosed() bool
	Finish()
	ChatID() int64
	ReadMessage() (*tgbotapi.Message, bool)
	SendBotMessage(msg tgbotapi.Chattable) error
}
