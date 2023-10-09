# GO-TELEGRAM-BOT
This is a framework wrapper over "github.com/go-telegram-bot-api/telegram-bot-api/v5" that allows writing logic for Telegram Bots in Go easily. 

## Features
The framework handles conversations with multiple users at the same time, each message will be handled by a goroutine dedicated to a particular chat.

All you need is to write handlers for initial commands. A handler can manage multistep conversations, and the framwork saves conversation state between steps, so that any conversation can be resumed in case of a failure or bot reboot.

The code for handling a conversation could be written in a "sync" way. On each call of `GetUpdateFromUser()` the execution will wait for a user to enter new data: write a message, press a button, send an attachment.

Also, the bot can run chrone jobs and send messages to users when they are not in an active conversation. 

## Usage

To run a bot you need:
1. Receive a bot API Token from telegram for your bot
2. Create new bot object with `bot.NewBot(apiToken)`
3. Apply configuration settings for the bot object (otherwise defaults will be used)
4. Add command handlers
5. Call `Run(ctx)`


### 3. Configuration

Below is the configuration methods for the bot: 

```go

// SetDebug sets debug flag for the bot (by default false)
SetDebug(debug bool)

// SetWebHook sets webhook flag for the bot (by default false), and parameters for webhook
// certFile and keyFile can be empty for http connection
SetWebHook(webHook bool, webHookExternalURL, webHookInternalUrl, certFile, keyFile string)

// SetUpdateTimeout sets timeout for bot updates (by default 0 - no timeout). Applies to long-pulling only.
SetUpdateTimeout(timeout int)

// WithJobs sets list of jobs for the bot
WithJobs(jobs jobs.JobDescriptionsList)

// SetAllowBotUsers sets flag that indicates whether conversation with bot users allowed (by default false)
SetAllowBotUsers(allow bool)

// WithStateIO sets interface for loading and saving the bot state.
WithStateIO(stateIO state.StateIO)

// SetMaxOpenConversations sets maximum number of open conversations that the bot can handle at a time (by default 1000)
SetMaxOpenConversations(max int)

// SetSingleMessageTrySendInterval sets interval in seconds between attempts to send a single message from bot to a chat (by default 10)
SetSingleMessageTrySendInterval(interval int)

// SetConversationTimeout sets timeout in minutes for the bot waiting an input from a user in a conversation (by default 10)
SetConversationTimeout(timeout int)

// SetMaxMessageQueue sets maximum number of messages that the bot can queue for a single conversation (by default 10)
SetMaxMessageQueue(max int)

// WithDefaultCommandHandler sets default command handler for the bot.
// The default handler is called when no other command handler is found for a command
WithDefaultCommandHandler(handler handlers.HandlerCreatorType)

// WithCommandHandlers sets list of command handlers for the bot
WithCommandHandlers(handlers []handlers.CommandHandler)

// WithGlobalHandlers sets list of global handlers for the bot
WithGlobalHandlers(handlers []handlers.CommandHandler)

// WithGlobalMessageFunc sets global message function for the bot
// Global message function is called to generate technical messages that bot sends to users
WithTechnicalMessageFunc(technicalMessageFunc dispatcher.TechnicalMessageFuncType)

// WithGlobalKeyboardFunc sets global keyboard function for the bot (by default nil)
// Global keyboard function is called to generate keyboard that bot sends to users with technical messages
// If global keyboard function is not set, the bot will not send any keyboard with technical messages
// Command handlers can call this function to generate keyboard for a message if needed
WithGlobalKeyboardFunc(globalKeyboardFunc dispatcher.GlobalKeyboardFuncType)

```

* `technicalMessageFunc` - function that generates common message for a user. Currently bot supports 6 common messages:
1. Message in case of error in a handler
2. Message in case of too many open conversation
3. Message in case of too many unprocessed updates from a user
4. Message when conversation was closed by the bot
5. Message when conversation was closed becouse user switched to another global handler
6. Message when conversation was finished normally
Each global message could be generated for a specific user, so you can add multy-language support. If the function returns an empty string, the correspondend message will not be shown.

## Development
### Architecture 
The bot runs conversations with multiple users in parallel (in different threads).
The `Dispatcher` object receives a new updates from Telegram and forwards it to a particular conversation. If there is no ongoing conversation for the chat or message triggers global handler, the `Dispatcher` will create a new conversation.
After the conversation is over, or a user switched to another conversation (triggered a global handler), or after some time of the user's inactivity, the `Dispatcher` closes the conversation.
There is a limit on the number of conversations that can be run in parallel and a limit on the message queue for a conversation. All messages that exceeded the limit will be ignored by the bot. 
Each conversation starts with a top-level handler. A handler may contain a number of steps, and each step, as a result, might navigate to any other step (or to the conversation end). Each step can request an information from a user and wait for it.
### Adding a new handler
To add a new high-level handler you need two things:
#### 1. Write creator for a new handler 
*Example:*
```go
var MyCustomHandlerCreator1 = func(ctx context.Context, conversation readers.BotConversation) convrsationHandler {
    // ...
    // get needed data from firstMessage
    // define and set common variables that reflects the convrsation state that should be preserved
	var userData = struct {
		// ...
	}{
		// ...
	}
    return handlers.NewStatefulHandler(
		&userData,
		[]handlers.ConversationStep{
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

```go
var MyCustomHandlerCreator2 = handlers.OneStepHandlerCreator(
	func(ctx context.Context, conversation readers.BotConversation) error {
	_, err = conversation.SendText("Wellcome to the bot!")
	return err
})

var DefaultHandlerCreator = handlers.ReplyMessageHandlerCreator("The command is not implemented yet")
```
#### 2. Create list of command handlers that should be passed to Dispatcher
```go
var AllHandlerCreators = []h.CommandHandler{
	{
		CommandSelector: handlers.RegExpCommandSelector("/command1"),
		HandlerCreator:  MyCustomCreator1,
	},
	{
		CommandSelector: handlers.RegExpCommandSelector("/command2"),
		HandlerCreator:  MyCustomCreator2,
	},
}
```

#### 3. Create list of `Global Handlers` that will terminate an ongoing conversation and start a new one
```go
var MyGlobalHandlers = []h.CommandHandler{
		{
			CommandSelector: handlers.RegExpCommandSelector("/cancel"),  // for example, /cancel command allows a user to terminate any convrsation
			HandlerCreator:  handlers.OneStepHandlerCreator(
				func(ctx context.Context, conversation readers.BotConversation) error {
				_, err = conversation.SendText("You terminated the conversation")
				return err
			}),
		},
	},
}
```

#### 4. Create function that returns technical messages
```go
var TechnicalMessagesFunction = func(chatID int64, messageID dispatcher.MessageIDType) string {
	// in the example we ignore chatID, but you can use it to show different messages to different users
	// for example, you can orginize multy-language support
		switch messageID {
		case dispatcher.TooManyMessages:
			return "Bot recieved too many messages from you, it will skip all new messages until process the existing"
		case dispatcher.ConversationClosedByBot:
			return "Bot closed the conversation with you"
		case dispatcher.ConversationEnded: // in case of normal ending, bot will not send any additional message
			return ""
		case dispatcher.UserError:
			return "There was an error during the conversation"
		case dispatcher.ConversationClosedByUser:
			return "You finished the conversation with the bot"
		}
		return ""
	}
```

#### 5. Create function that returns global keyboard that will be attached to each technical message
```go
var MyGloabalKeyboard = func(chatID int64) dispatcher.GlobalKeyboardType {
	return dispatcher.RemoveKeyboard() // remove keyboard if it was left by any conversation
}
```

#### 6. Run the bot from your code
```go
err := bot.NewBot(BotAPIToken).
	WithStateIO(state.NewFileState("botstate.json")).
	WithDefaultCommandHandler(DefaultHandlerCreator).
	WithCommandHandlers(AllHandlerCreators).
	WithGlobalKeyboard(MyGloabalKeyboard).
	WithGlobalHandlers(MyGlobalHandlers).
	WithTechnicalMessageFunc(TechnicalMessagesFunction).
	Run(context.Background())
if err != nil { // cannot start the bot
	panic(err)
}
```

### TO DO
* 