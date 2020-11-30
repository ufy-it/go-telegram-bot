package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

type envVars struct {
	BotAPIToken                string // The tocken to access BOT API, cannot be empty. BOT_API_TOKEN env variable
	UpdateConfigTimeout        int    // Timeout for getting messages from the telegram to the bot. Default 60. TELEGRAM_UPDATE_CONFIG_TIMEOUT env variable
	MaxOpenConversations       int    // Maximum number of active conversation at the moment of time. Default 1000. MAX_OPEN_CONVERSATIONS env variable
	MaxConversationQueue       int    // Maximum number of unprocessed messages in a conversation's queue. Default 100. MAX_CONVERSATION_QUEUE env variable
	ClearJobIntervalSeconds    int    // Interval in seconds between continuous runs of the job that clears outdated conversations. Default 10. CLEAR_JOB_INTERVAL_SECONDS env variable
	ConversationTimeoutMinutes int    // The number of minutes from latest user activity after which an active conversation will be closed. Default 10. CONVERSATION_TIMEOUT_MINUTES env variable
	ConversationFinishSeconds  int    // The number of seconds after which an inactive conversation object will be cleared. Default 10. CONVERSATION_FINISH_SECONDS env variable
	AllowBotUsers              bool   // Flag indicates whether the bot will communicate with another bots. Default false. ALLOW_BOT_USERS env variable
}

func parseIntWithDefault(varName string, def int, isPositive bool) (int, error) {
	tmpStr := os.Getenv(varName)
	if tmpStr == "" {
		return def, nil
	}
	tmpInt, err := strconv.Atoi(tmpStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %s environment variable: %v", varName, err)
	}
	if isPositive && tmpInt < 0 {
		return 0, fmt.Errorf("%s environment variable should not be negative", varName)
	}
	return tmpInt, nil
}

func parseBoolWithDefault(varName string, def bool) (bool, error) {
	tmpStr := os.Getenv(varName)
	if tmpStr == "" {
		return def, nil
	}
	tmpBool, err := strconv.ParseBool(tmpStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse %s environment variable: %v", varName, err)
	}
	return tmpBool, nil
}

func readEnvVars() (envVars, error) {
	result := envVars{}
	result.BotAPIToken = os.Getenv("BOT_API_TOKEN")
	if result.BotAPIToken == "" {
		return envVars{}, errors.New("BOT_API_TOKEN is not set")
	}

	var err error

	result.UpdateConfigTimeout, err = parseIntWithDefault("TELEGRAM_UPDATE_CONFIG_TIMEOUT", 60, true)
	if err != nil {
		return envVars{}, err
	}

	result.MaxOpenConversations, err = parseIntWithDefault("MAX_OPEN_CONVERSATIONS", 1000, true)
	if err != nil {
		return envVars{}, err
	}

	result.MaxConversationQueue, err = parseIntWithDefault("MAX_CONVERSATION_QUEUE", 100, true)
	if err != nil {
		return envVars{}, err
	}

	result.ClearJobIntervalSeconds, err = parseIntWithDefault("CLEAR_JOB_INTERVAL_SECONDS", 10, true)
	if err != nil {
		return envVars{}, err
	}

	result.ConversationTimeoutMinutes, err = parseIntWithDefault("CONVERSATION_TIMEOUT_MINUTES", 10, true)
	if err != nil {
		return envVars{}, err
	}

	result.ConversationTimeoutMinutes, err = parseIntWithDefault("CONVERSATION_FINISH_SECONDS", 10, true)
	if err != nil {
		return envVars{}, err
	}
	return result, nil
}
