package bot

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/ufy-it/go-telegram-bot/dispatcher"
	"github.com/ufy-it/go-telegram-bot/jobs"
	"github.com/ufy-it/go-telegram-bot/logger"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Config describes configuration oprions for the bot
type Config struct {
	APIToken           string                   // Bot API token
	Debug              bool                     // flag to indicate whether run the bot in debug
	WebHook            bool                     // flag to indicate whether to run webhook or long pulling
	Dispatcher         dispatcher.Config        // configuration for the dispatcher
	Jobs               jobs.JobDescriptionsList // list of jobs to run
	UpdateTimeout      int
	StateIO            state.StateIO // interface for loading and saving the bot state
	AllowBotUsers      bool          // flag that indicates whether conversation with bot users allowed
	WebHookExternalURL string        // "https://www.google.com:8443/"+bot.Token
	WebHookInternalURL string        // "0.0.0.0:8443"
	CertFile           string        // "cert.pem"
	KeyFile            string        // "key.pem"
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

// RunBot handles conversation with bot users and runs jobs in an infinite loop
func RunBot(ctx context.Context, config Config) error {
	bot, err := tgbotapi.NewBotAPI(config.APIToken)
	if err != nil {
		return fmt.Errorf("error accessing the bot: %v", err)
	}
	bot.Debug = config.Debug
	logger.Note("Authorized on account %s", bot.Self.UserName)

	var upd tgbotapi.UpdatesChannel
	if config.WebHook {
		salt := randStringBytes(10)
		var wh tgbotapi.WebhookConfig
		if config.CertFile == "" {
			wh, err = tgbotapi.NewWebhook(config.WebHookExternalURL + "/" + bot.Token + salt)
		} else {
			wh, err = tgbotapi.NewWebhookWithCert(config.WebHookExternalURL+"/"+bot.Token+salt, tgbotapi.FilePath(config.CertFile))
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
		if config.CertFile == "" {
			go http.ListenAndServe(config.WebHookInternalURL, nil)
		} else {
			go http.ListenAndServeTLS(config.WebHookInternalURL, config.CertFile, config.KeyFile, nil)
		}
	} else {
		_, err = bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: true})
		if err != nil {
			return fmt.Errorf("error removing web-hook: %v", err)
		}
		var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
		ucfg.Timeout = config.UpdateTimeout
		upd = bot.GetUpdatesChan(ucfg)
	}
	disp, err := dispatcher.NewDispatcher(ctx, config.Dispatcher, bot, config.StateIO)
	if err != nil {
		return err
	}
	jobs.RunJobs(ctx, config.Jobs, disp)
	for {
		select {
		case update := <-upd:
			if update.Message != nil && update.Message.From.IsBot && !config.AllowBotUsers {
				continue // skip message from another bot
			}
			disp.DispatchUpdate(&update)
		case <-ctx.Done():
			logger.Note("context is closed, exiting")
			return nil
		}
	}
}
