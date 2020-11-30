package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"ufygobot/dispatcher"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

func main() {
	env, err := readEnvVars()
	if err != nil {
		log.Panicf("Error reading settings from environment: %v", err)
	}
	// connect to the bot using the token
	bot, err := tgbotapi.NewBotAPI(env.BotAPIToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	var ucfg tgbotapi.UpdateConfig = tgbotapi.NewUpdate(0)
	ucfg.Timeout = env.UpdateConfigTimeout
	upd, _ := bot.GetUpdatesChan(ucfg)
	disp := dispatcher.NewDispatcher(env.MaxOpenConversations,
		env.MaxConversationQueue,
		env.ClearJobIntervalSeconds,
		env.ConversationTimeoutMinutes,
		env.ConversationFinishSeconds,
		bot)

	c := make(chan os.Signal) // Gracefully terminate the program
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case update := <-upd:
			if update.Message.From.IsBot && !env.AllowBotUsers {
				continue // skip message from another bot
			}
			err := disp.DispatchMessage(update.Message)
			if err != nil {
				log.Printf("Error dispatching message from %s: %v", update.Message.From.UserName, err)
			}
		case sig := <-c:
			log.Printf("Recieved interrupt signal %v", sig)
			err := disp.Close()
			if err != nil {
				log.Printf("Error: %v", err)
			}
		}

	}
}
