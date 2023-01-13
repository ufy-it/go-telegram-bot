package bot

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/ufy-it/go-telegram-bot/conversation"
	"github.com/ufy-it/go-telegram-bot/dispatcher"
	"github.com/ufy-it/go-telegram-bot/handlers"
	"github.com/ufy-it/go-telegram-bot/jobs"
	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Config describes configuration oprions for the bot
type botConfig struct {
	apiToken           string                   // Bot API token
	debug              bool                     // flag to indicate whether run the bot in debug
	webHook            bool                     // flag to indicate whether to run webhook or long pulling
	dispatcherConfig   dispatcher.Config        // configuration for the dispatcher
	botJobs            jobs.JobDescriptionsList // list of jobs to run
	updateTimeout      int
	stateIO            state.StateIO // interface for loading and saving the bot state
	allowBotUsers      bool          // flag that indicates whether conversation with bot users allowed
	webHookExternalURL string        // "https://www.google.com:8443/"+bot.Token
	webHookInternalURL string        // "0.0.0.0:8443"
	certFile           string        // "cert.pem"
	keyFile            string        // "key.pem"
}

// NewBot creates a new bot configuration with default values and no command handlers and jobs
func NewBot(apiToken string) *botConfig {
	return &botConfig{
		apiToken: apiToken,
		debug:    false,
		webHook:  false,
		dispatcherConfig: dispatcher.Config{
			MaxOpenConversations:         1000,
			SingleMessageTrySendInterval: 10,
			ConversationConfig: conversation.Config{
				MaxMessageQueue: 10,
				TimeoutMinutes:  10,
			},
			Handlers: &handlers.CommandHandlers{
				Default: handlers.EmptyHandlerCreator(),
				List:    []handlers.CommandHandler{},
			},
			GlobalHandlers:       []handlers.CommandHandler{},
			TechnicalMessageFunc: dispatcher.EmptyTechnicalMessageFunc,
			GloabalKeyboardFunc:  nil,
		},
		botJobs:            jobs.JobDescriptionsList{},
		updateTimeout:      0,
		stateIO:            nil,
		allowBotUsers:      false,
		webHookExternalURL: "",
		webHookInternalURL: "",
		certFile:           "",
		keyFile:            "",
	}
}

// SetDebug sets debug flag for the bot (by default false)
func (c *botConfig) SetDebug(debug bool) *botConfig {
	c.debug = debug
	return c
}

// SetWebHook sets webhook flag for the bot (by default false), and parameters for webhook
func (c *botConfig) SetWebHook(webHook bool, webHookExternalURL, webHookInternalUrl, certFile, keyFile string) *botConfig {
	c.webHook = webHook
	c.webHookExternalURL = webHookExternalURL
	c.webHookInternalURL = webHookInternalUrl
	c.certFile = certFile
	c.keyFile = keyFile
	return c
}

// SetUpdateTimeout sets timeout for bot updates (by default 0 - no timeout). Applies to long-pulling only.
func (c *botConfig) SetUpdateTimeout(timeout int) *botConfig {
	c.updateTimeout = timeout
	return c
}

// WithJobs sets list of jobs for the bot
func (c *botConfig) WithJobs(jobs jobs.JobDescriptionsList) *botConfig {
	c.botJobs = jobs
	return c
}

// SetAllowBotUsers sets flag that indicates whether conversation with bot users allowed (by default false)
func (c *botConfig) SetAllowBotUsers(allow bool) *botConfig {
	c.allowBotUsers = allow
	return c
}

// WithStateIO sets interface for loading and saving the bot state.
func (c *botConfig) WithStateIO(stateIO state.StateIO) *botConfig {
	c.stateIO = stateIO
	return c
}

// SetMaxOpenConversations sets maximum number of open conversations that the bot can handle at a time (by default 1000)
func (c *botConfig) SetMaxOpenConversations(max int) *botConfig {
	c.dispatcherConfig.MaxOpenConversations = max
	return c
}

// SetSingleMessageTrySendInterval sets interval in seconds between attempts to send a single message from bot to a chat (by default 10)
func (c *botConfig) SetSingleMessageTrySendInterval(interval int) *botConfig {
	c.dispatcherConfig.SingleMessageTrySendInterval = interval
	return c
}

// SetConversationTimeout sets timeout in minutes for the bot waiting an input from a user in a conversation (by default 10)
func (c *botConfig) SetConversationTimeout(timeout int) *botConfig {
	c.dispatcherConfig.ConversationConfig.TimeoutMinutes = timeout
	return c
}

// SetMaxMessageQueue sets maximum number of messages that the bot can queue for a single conversation (by default 10)
func (c *botConfig) SetMaxMessageQueue(max int) *botConfig {
	c.dispatcherConfig.ConversationConfig.MaxMessageQueue = max
	return c
}

// WithDefaultCommandHandler sets default command handler for the bot.
// The default handler is called when no other command handler is found for a command
func (c *botConfig) WithDefaultCommandHandler(handler handlers.HandlerCreatorType) *botConfig {
	c.dispatcherConfig.Handlers.Default = handler
	return c
}

// WithCommandHandlers sets list of command handlers for the bot
func (c *botConfig) WithCommandHandlers(handlers []handlers.CommandHandler) *botConfig {
	c.dispatcherConfig.Handlers.List = handlers
	return c
}

// WithGlobalHandlers sets list of global handlers for the bot
func (c *botConfig) WithGlobalHandlers(handlers []handlers.CommandHandler) *botConfig {
	c.dispatcherConfig.GlobalHandlers = handlers
	return c
}

// WithGlobalMessageFunc sets global message function for the bot
// Global message function is called to generate technical messages that bot sends to users
func (c *botConfig) WithTechnicalMessageFunc(globalMessageFunc dispatcher.TechnicalMessageFuncType) *botConfig {
	c.dispatcherConfig.TechnicalMessageFunc = globalMessageFunc
	return c
}

// WithGlobalKeyboardFunc sets global keyboard function for the bot
// Global keyboard function is called to generate keyboard that bot sends to users with technical messages
// If global keyboard function is not set, the bot will not send any keyboard with technical messages
// Command handlers can call this function to generate keyboard for a message if needed
func (c *botConfig) WithGlobalKeyboardFunc(globalKeyboardFunc dispatcher.GlobalKeyboardFuncType) *botConfig {
	c.dispatcherConfig.GloabalKeyboardFunc = globalKeyboardFunc
	return c
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// Run starts the bot and handlers conversations with uers and job runs in an infinite loop
// The function returns error if the bot cannot be started
// To stop the bot, cancel the context
func (config *botConfig) Run(ctx context.Context) error {
	bot, err := tgbotapi.NewBotAPI(config.apiToken)
	if err != nil {
		return fmt.Errorf("error accessing the bot: %v", err)
	}
	bot.Debug = config.debug
	logger.Note("Authorized on account %s", bot.Self.UserName)

	var upd tgbotapi.UpdatesChannel
	if config.webHook {
		salt := randStringBytes(10)
		var wh tgbotapi.WebhookConfig
		if config.certFile == "" {
			wh, err = tgbotapi.NewWebhook(config.webHookExternalURL + "/" + bot.Token + salt)
		} else {
			wh, err = tgbotapi.NewWebhookWithCert(config.webHookExternalURL+"/"+bot.Token+salt, tgbotapi.FilePath(config.certFile))
		}
		if err != nil {
			return fmt.Errorf("error creating webhook config: %v", err)
		}
		_, err = bot.Request(wh)
		if err != nil {
			return fmt.Errorf("error setting web-hook: %v", err)
		}
		info, err := bot.GetWebhookInfo()
		if err != nil {
			return fmt.Errorf("error getting web-hook info: %v", err)
		}
		if info.LastErrorDate != 0 {
			logger.Warning("[Telegram callback failed]%s", info.LastErrorMessage)
		}

		upd = bot.ListenForWebhook("/" + bot.Token + salt)
		if config.certFile == "" {
			go http.ListenAndServe(config.webHookInternalURL, nil)
		} else {
			go http.ListenAndServeTLS(config.webHookInternalURL, config.certFile, config.keyFile, nil)
		}
	} else {
		_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
		if err != nil {
			return fmt.Errorf("error removing web-hook: %v", err)
		}
		var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
		ucfg.Timeout = config.updateTimeout
		upd = bot.GetUpdatesChan(ucfg)
	}
	disp, err := dispatcher.NewDispatcher(ctx, config.dispatcherConfig, bot, config.stateIO)
	if err != nil {
		return err
	}
	jobs.RunJobs(ctx, config.botJobs, disp)
	for {
		select {
		case update := <-upd:
			if update.Message != nil && update.Message.From.IsBot && !config.allowBotUsers {
				continue // skip message from another bot
			}
			disp.DispatchUpdate(&update)
		case <-ctx.Done():
			logger.Note("context is closed, exiting")
			return nil
		}
	}
}
