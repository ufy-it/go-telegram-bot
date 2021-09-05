# GO-TELEGRAM-BOT
This is a framework wrapper over "github.com/Syfaro/telegram-bot-api" that allows writing logic for Telegram Bots in Go easily. 

## Features
The framework handles conversations with multiple users at the same time, each message will be handled by a goroutine dedicated to a particular user.

All you need is to write handlers for initial commands. A handler can manage multistep conversations. 
The framework continuously saves states of each conversation into a file, so the conversation could be recovered in case of a reboot.

The code for handling a conversation could be written in a "sync" way. On each call of `GetUpdateFromUser()` the execution will wait for a user to enter new data: write a message, press a button, send an attachment.

Also, bot can run chrone jobs and send messages to users when they are not in an active conversation. 

## Usage

To run a bot you need to call `bot.RunBot( config )` with a proper configuration.

### Configuration

Below is the configuration for the bot: 

```
type Config struct {
	APIToken               string                   // Bot API token
	Debug                  bool                     // flag to indicate whether run the bot in debug
	WebHook                bool                     // flag to indicate whether to run webhook or polling
	Dispatcher             dispatcher.Config        // configuration for the dispatcher
	Jobs                   jobs.JobDescriptionsList // list of jobs to run
	UpdateTimeout          int
	StateFile              string // path to the state file
	AllowBotUsers          bool   // flag that indicates whether conversation with bot users allowed
	WebHookExternalURL     string // "https://www.google.com:8443/"+bot.Token
	WebHookInternalURL     string // "0.0.0.0:8443"
	CertFile               string // "cert.pem"
	KeyFile                string // "key.pem"
}
```

* `APIToken` - telegrambot IP token string, needed both for WebHook and polling modes
* `Debug` - boolean flag to indicate whether the bot should be run in debug mode
* `WebHook` - boolean flag to indicate whether the bot should be run as WebHook, or pulling. Read more about modes here: https://go-telegram-bot-api.dev/
* `UpdateTimeout` - timeout for reading updates in polling mode
* `StateFile` - path to the state file, where bot saves state (in JSON) for all ongoing conversations
* `AllowBotUsers` - boolean flag that allows bot to answer other bots, otherwise it will ignore
* `WebHookExternalURL`, `WebHookInternalURL`, `CertFile`, `KeyFile` - configuration parameters for the web hook mode
* `Jobs` - list of jobs to run
* `Dispatcher` - configuration for the message dispacher object. Is described in details below

Conversation dispatcher should be configured with the following parameters:

```
// Config contains configuration parameters for a new dispatcher
type Config struct {
	MaxOpenConversations           int                 // the maximum number of open conversations
	SingleMessageTrySendInterval   int                 // interval between several tries of send single message to a user
	ConversationConfig             conversation.Config // configuration for a conversation
	CancelCommand                  string              // the command that a user can use to cancel any conversation
	ToManyOpenConversationsMessage string              // message to send to a user if a conversation cannot be started
	Handlers        			   *handlers.CommandHandlers // list of handlers for command handling
}
```

* `MaxOpenConversations` - the maximum number of open conversations, the dispatcher will refuse messages from new users if it is already running `MaxOpenConversations` conversations
* `SingleMessageTrySendInterval` - interval between tries of sending single message to a user with `SendSingleMessage` method
* `CancelCommand` - command that allows user to cancel any ongoing conversation (i.e. `/cancel`)
* `ToManyOpenConversationsMessage` - text that will be send to a user that tries to start a new conversation when the limit is already reached. 
* `ConversationConfig` - config for a conversation object. Is described in detail below
* `Handlers` - list of command handlers, the only code that should be written by the framework user

Conversation object should be configured with the following parameters:

```
// Config is struct with configuration parameters for a conversation
type Config struct {
	MaxMessageQueue int                       // the maximum size of message queue for a conversation
	TimeoutMinutes  int                       // timeout for a user's input

	CloseByBotMessage     string // message to send to a user if the conversation is closed from the bot side
	CloseByUserMessage    string // message to send to a user if they decided to close the conversation
	ToManyMessagesMessage string // message to send to a user in case to too many unprocessed messages
}
```

* `MaxMessageQueue` - the maximum number of unprocessed messages from a user that conversation queue can keep. All new messages will be descarded if the queue is already full
* `TimeoutMinutes` - for how long the bot will wait for a user's input. After the timeout a conversation will e closed by the bot.
* `CloseByBotMessage` - text message that will be send to a user if the conversation is closed by the bot
* `CloseByUserMessage` - a confirmation text message to send to a user if they closed the conversation
* `ToManyMessagesMessage` - text message that will be send to a user as a reply on it's action that is descard due to the full message queue


## Development
### Architecture 
The bot runs conversations with multiple users in parallel (in different threads).
The `Dispatcher` object receives a new updates from Telegram and forwards it to a particular conversation. If there is no ongoing conversation with the user, the `Dispatcher` will create a new conversation.
After the conversation is over, or after some time of user inactivity, the `Dispatcher` closes the conversation.
There is a limit on the number of conversations that can be run in parallel and a limit on the message queue for a conversation. All messages that exceeded the limit will be ignored by the bot. 
Each conversation starts with a top-level handler. A handler may contain a number of steps, and each step, as a result, might navigate to any other step (or to the conversation end). Each step can request an information from a user and wait for it.
### Adding a new handler
To add a new high-level handler you need two things:
#### 1. Write creator for a new handler 
*Example:*
```
var MyCustomCreator1 = func(ctx context.Context, conversation readers.BotConversation) convrsationHandler {
    // ...
    // get needed data from firstMessage
    // define and set common variables that reflects the convrsation state that should be preserved
	var userData = struct {
		// ...
	}{
		// ...
	}
    return NewStatefulHandler(
		conversation.ChatID(),
		&userData,
		[]ConversationStep{
            // ... multiple steps
			{
				Action: func() (handlers.StepResult, error) {
					err := conversation.SendText("Welcome to the sample bot!") // send message to user
					return handlers.ActionResultWithError(handlers.NextStep, err) // go to the next step 
				},
			},
			// ...
			{
				Name: "final",
				Action: func() (handlers.StepResult, error) {
					// there already some usefull readers in `readers` package, you can create your own
					reply, err := readers.AskReplyEmail(ctx, conversation, "Enter your email", buttons.NewAbortButton("Abort"), "Please enter a valid email")
					// bot received new update from the user and needs to handle it properly 
					if err != nil { // if something went wrong
						return handlers.ActionResultError(err)	
					}
					if reply.Exit { //user or bot ended the conversation
						return handlers.EndConversation()
					}
					if reply.Data == buttons.NavigationAbort { // Abort button pressed
						_, err = conversation.SendText("See you next time")
						return handlers.ActionResultWithError(handlers.EndConversation, err)
					}
					_, err = conversation.SendText(fmt.Sprintf("Your email is %s", reply.Text))
					return handlers.ActionResultWithError(handlers.EndConversation, err) // end the conversation
				},
			},
		})
}

```

If you do not want to manage multy-step conversations, you can simplify your code, using helpers:

```
var MyCustomCreator2 = handlers.OneStepHandlerCreator(
	func(ctx context.Context, conversation readers.BotConversation) error {
	_, err = conversation.SendText("Wellcome to the bot!")
	return err
})

var DefaultHandlerCreator = handlers.OneStepHandlerCreator(
	func(ctx context.Context, conversation readers.BotConversation) error {
	_, err = conversation.SendText("The command is not implemented yet")
	return err
})
```
#### 2. Add command to `allHandlerCreators` list that should be passed to Dispatcher 
```
var AllHandlerCreators = h.CommandHandlers{
	Default: DefaultHandlerCreator, // this handler is a fallback if the command entered by a user does not match any from the List
	List: []h.CommandHandler{
		{regexp.MustCompile("/command1"), MyCustomCreator1},
		{regexp.MustCompile("/command2"), MyCustomCreator2},
	},
	UserErrorMessage: "Error occured during the execution",
}
```

#### 3. Run the bot from your code
```
err = bot.RunBot(
	context.Background(),
	bot.Config{
		APIToken: BotAPIToken, // some API token
		Debug:    false,
		WebHook:  false,
		Dispatcher: dispatcher.Config{
			MaxOpenConversations:           1000,
			SingleMessageTrySendInterval:   10,
			CancelCommand:                  "/cancel",
			ToManyOpenConversationsMessage: "Bot is ovverloaded. Try to send your request later",
			ConversationConfig: conversation.Config{
				MaxMessageQueue:       100,
				TimeoutMinutes:        15,
				CloseByBotMessage:     "The bot had to abort the conversation",
				CloseByUserMessage:    "You canceled the conversation with the bot",
				ToManyMessagesMessage: "You sent too many messages to the bot, wait for proccessing. All new messages will be discarded for now",
			},
			Handlers:            &AllHandlerCreators,
		},
		Jobs:          make([]jobs.JobDescription, 0), // empty list of jobs
		UpdateTimeout: 60, // 
		StateFile:     "botstate.json",
		AllowBotUsers: false,
	})
```

### TO DO
* Add multy-language support in technical messages
