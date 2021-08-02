# GO-TELEGRAM-BOT
This is a framework wrapper over "github.com/Syfaro/telegram-bot-api" 
## Configuration
The project requires GO 1.15.5 or higher. 

## Development
### Architecture 
The bot runs conversations with multiple users in parallel (in different threads).
The `Dispatcher` object receives a new updates from Telegram and forwards it to a particular conversation. If there is no ongoing conversation with the user, the `Dispatcher` will create a new conversation.
After the conversation is over, or after some time of user inactivity, the `Dispatcher` closes the conversation.
There is a limit on the number of conversations that can be run in parallel and a limit on the message queue for a conversation. All messages that exceeded the limit will be ignored by the bot. 
Each conversation starts with a top-level handler. A handler may contain a number of steps, and each step, as a result, might navigate to any other step (or to the conversation end). Each step can request an information from a user and wait for it.
### Adding a new handler
To add a new high-level handler you need two things:
#### 1. Write creator for a new handler in `custom_handlers` package 
*Example:*
```
var MyCustomCreator = func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) convrsationHandler {
    // ...
    // get needed data from firstMessage
    // define and set common variables for steps
    return &baseHandler{ 
		conversation: conversation,
		steps: []conversationStep{
            // ... multiple steps
			{
				action: func(conversation readers.BotConversation) (stepActionResult, error) {
					msg := tgbotapi.NewMessage(conversation.chatID(), "Message to user")
					err := conversation.SendBotMessage(msg) // send message to user
					return goToCommand("final"), err // go to step "final" 
				},
			},
            // ... 
			{
                name: "final",
				action: func(conversation readers.BotConversation) (stepActionResult, error) {
					update, exit := conversation.ReadUpdate() // read input message, this should be handled by helpers that reads needed type or process a button pressed
					if exit {
						return closeCommand(), nil // process situation if user did not respond during timeout
					}
                    replyText := "my text"
                    if update.Message != nil {
                        replyText = update.Message
                    }
					msg := tgbotapi.NewMessage(conversation.ChatID(), replyText)
					err := conversation.SendBotMessage(msg)
					return endConversation(), err // end the conversation
				},
			},
		},
	}
}
```
#### 2. Add command to `allHandlerCreators` list that should be passed to Dispatcher 
```
var AllHandlerCreators = h.CommandHandlers{
	Default: defaultHandlerCreator,
	List: []h.CommandHandler{
		{regexp.MustCompile("/me"), meHandlerCreator},
		{regexp.MustCompile("/reg"), regHandlerCreator},
	},
	UserErrorMessage: "Возникла ошибка, пожалуйста попробуйте еще раз или обратитесь к разработчикам.",
}
```

### TO DO
* Connect to a real database
* Separate bot to a package so that it can be used in many different bots