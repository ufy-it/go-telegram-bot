package bot

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ufy-it/go-telegram-bot/dispatcher"
	"github.com/ufy-it/go-telegram-bot/jobs"
	"github.com/ufy-it/go-telegram-bot/logger"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// Config describes configuration oprions for the bot
type Config struct {
	APIToken               string                   // Bot API token
	Debug                  bool                     // flag to indicate whether run the bot in debug
	WebHook                bool                     // flag to indicate whether to run webhook or long pooling
	Dispatcher             dispatcher.Config        // configuration for the dispatcher
	Jobs                   jobs.JobDescriptionsList // list of jobs to run
	UpdateTimeout          int
	StateFile              string // path to the state file
	AllowBotUsers          bool   // flag that indicates whether conversation with bot users allowed
	JobsStopTimeoutSeconds int    // timeout for stopping all jobs
	WebHookExternalURL     string // "https://www.google.com:8443/"+bot.Token
	WebHookInternalURL     string // "0.0.0.0:8443"
	CertFile               string // "cert.pem"
	KeyFile                string // "key.pem"
}

// RunBot handles conversation with bot users and runs jobs in an infinite loop
func RunBot(config Config) error {
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
	disp := dispatcher.NewDispatcher(config.Dispatcher, bot, config.StateFile)
	manager := jobs.NewJobManager(config.Jobs, disp)
	c := make(chan os.Signal) // Gracefully terminate the program
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case update := <-upd:
			if update.Message != nil && update.Message.From.IsBot && !config.AllowBotUsers {
				continue // skip message from another bot
			}
			err := disp.DispatchUpdate(&update)
			if err != nil {
				logger.Error("error dispatching message from %s: %v", update.Message.From.UserName, err)
			}
		case sig := <-c:
			logger.Note("Recieved interrupt signal %v", sig)
			wg := sync.WaitGroup{}
			var dispCloseErr error
			var managerStopErr error
			// close dispatcher and job manager in parallel
			go func() {
				wg.Add(1)
				dispCloseErr = disp.Close()
				wg.Done()
			}()
			go func() {
				wg.Add(1)
				managerStopErr = manager.Stop(config.JobsStopTimeoutSeconds)
				wg.Done()
			}()
			wg.Wait()
			if dispCloseErr != nil || managerStopErr != nil {
				return fmt.Errorf("error during closing dispather: %v, error during stopping jobs: %v", dispCloseErr, managerStopErr)
			}
			logger.Note("Dispatcher closed")
			logger.Note("Jobs stopped")
			return nil
		}
	}
}
