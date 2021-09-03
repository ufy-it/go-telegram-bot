package bot

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ufy-it/go-telegram-bot/dispatcher"
	"github.com/ufy-it/go-telegram-bot/jobs"
	"github.com/ufy-it/go-telegram-bot/logger"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// Config describes configuration oprions for the bot
type Config struct {
	APIToken           string                   // Bot API token
	Debug              bool                     // flag to indicate whether run the bot in debug
	WebHook            bool                     // flag to indicate whether to run webhook or long pulling
	Dispatcher         dispatcher.Config        // configuration for the dispatcher
	Jobs               jobs.JobDescriptionsList // list of jobs to run
	UpdateTimeout      int
	StateFile          string // path to the state file
	AllowBotUsers      bool   // flag that indicates whether conversation with bot users allowed
	WebHookExternalURL string // "https://www.google.com:8443/"+bot.Token
	WebHookInternalURL string // "0.0.0.0:8443"
	CertFile           string // "cert.pem"
	KeyFile            string // "key.pem"
}

// RunBot handles conversation with bot users and runs jobs in an infinite loop
func RunBot(ctx context.Context, config Config) error {
	bot, err := tgbotapi.NewBotAPI(config.APIToken)
	if err != nil {
		return fmt.Errorf("Error accessing the bot: %v", err)
	}
	bot.Debug = config.Debug
	logger.Note("Authorized on account %s", bot.Self.UserName)

	var upd tgbotapi.UpdatesChannel
	if config.WebHook {
		_, err = bot.SetWebhook(tgbotapi.NewWebhookWithCert(config.WebHookExternalURL, config.CertFile))
		if err != nil {
			return fmt.Errorf("Error setting web-hook: %v", err)
		}
		info, err := bot.GetWebhookInfo()
		if err != nil {
			return fmt.Errorf("Error getting web-hook info: %v", err)
		}
		if info.LastErrorDate != 0 {
			logger.Warning("[Telegram callback failed]%s", info.LastErrorMessage)
		}
		upd = bot.ListenForWebhook("/" + bot.Token)
		go http.ListenAndServeTLS(config.WebHookInternalURL, config.CertFile, config.KeyFile, nil)
	} else {
		var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
		ucfg.Timeout = config.UpdateTimeout
		upd, _ = bot.GetUpdatesChan(ucfg)
	}
	disp, err := dispatcher.NewDispatcher(ctx, config.Dispatcher, bot, config.StateFile)
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
