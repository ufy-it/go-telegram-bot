#GO-TELEGRAM-BOT
This is a framework wrapper over "github.com/Syfaro/telegram-bot-api" 
## Configuration
The project requires GO 1.15.5 or higher and some environment variables to be set
### Installing GO
1. Install Go in your environment (for Linux: `sudo apt install golang-go`)
1. Install GVM (to maintain different GO versions)
    1. `bash < <(curl -s -S -L https://raw.githubusercontent.com/moovweb/gvm/master/binscripts/gvm-installer)`
    1. `source ~/.gvm/scripts/gvm`
    1. `gvm install go1.15.5`
    1. `gvm use go1.15.5`
    1. `echo $GOPATH` (use this value in your IDE setup)
### Configure VS Code IDE
1. Install VS Code & open the project
1. Add settings (into settings.json) to point to correct GOPATH and auto-install dependancies:
    ```
    "go.gopath": "your $GOPATH",
    "go.installDependenciesWhenBuilding": true,
    "program": "${workspaceRoot}",
    ```
1. Add `BOT_API_TOKEN` to environment variables (in `.vscode/launch.json`) 
    ```
    "env": {
                "BOT_API_TOKEN": <your BOT API token>,
            },
    ```
1. (Optional) Specify other environment variables (in `.vscode/launch.json`)
    * `TELEGRAM_UPDATE_CONFIG_TIMEOUT` - timeout for getting messages from the telegram to the bot. Default 60.
    * `MAX_OPEN_CONVERSATIONS` - maximum number of active conversations at the moment of time. Default 1000.
    * `MAX_CONVERSATION_QUEUE` - the maximum number of unprocessed messages in a conversation's queue. Default 100.
    * `CLEAR_JOB_INTERVAL_SECONDS` - interval in seconds between continuous runs of the job that clears outdated conversations. Default 10.
    * `CONVERSATION_TIMEOUT_MINUTES` - the number of minutes from the latest user activity after which an active conversation will be closed. Default 10.
    * `CONVERSATION_FINISH_SECONDS` - the number of seconds after which an inactive conversation object will be cleared. Default 10.
    * `ALLOW_BOT_USERS` - the flag indicates whether the bot will communicate with other bots. Default `false`.
    * `SHUTDOWN_TIMEOUT_SECONDS` - the number of seconds the bot has to finish all threads after receiving an interrupt signal. Default 10.
    * `DEBUG_BOT_API` - the flag indicates whether BOT API should log their messages. Default `false`. 

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