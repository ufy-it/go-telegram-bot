package conversation_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ufy-it/go-telegram-bot/conversation"
	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ------------------------------------------------
// Tests for methods of InputConversation interface (how conversation is visible to the Dispatcher)
// ------------------------------------------------
func TestGetUpdateChatID(t *testing.T) {
	id, err := conversation.GetUpdateChatID(nil)

	checkID := func(expected int64, actual int64) {
		if expected != actual {
			t.Errorf("Expected id %d, got %d", expected, actual)
		}
	}
	checkErr := func(expected error, actual error) {
		if (expected == nil) != (actual == nil) || expected != nil && expected.Error() != actual.Error() {
			t.Errorf("Expected error \"%v\", actual \"%v\"", expected, actual)
		}
	}

	checkID(0, id)
	checkErr(errors.New("update is nil"), err)

	id, err = conversation.GetUpdateChatID(
		&tgbotapi.Update{
			Message: &tgbotapi.Message{
				Chat: &tgbotapi.Chat{
					ID: 12,
				},
			},
		})

	checkID(12, id)
	checkErr(nil, err)

	id, err = conversation.GetUpdateChatID(
		&tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{
						ID: 13,
					},
				},
			},
		})

	checkID(13, id)
	checkErr(nil, err)

	id, err = conversation.GetUpdateChatID(&tgbotapi.Update{})

	checkID(0, id)
	checkErr(errors.New("usupported query type"), err)

	id, err = conversation.GetUpdateChatID(
		&tgbotapi.Update{
			Message: &tgbotapi.Message{},
		})
	checkID(0, id)
	checkErr(errors.New("expected chat field in update, got nil"), err)

	id, err = conversation.GetUpdateChatID(
		&tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				Message: &tgbotapi.Message{},
			},
		})
	checkID(0, id)
	checkErr(errors.New("expected chat field in update, got nil"), err)

	id, err = conversation.GetUpdateChatID(
		&tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{},
		})
	checkID(0, id)
	checkErr(errors.New("expected message field in update, got nil"), err)
}

type dummyBot struct {
	ReplyError   bool
	SentMessages []tgbotapi.Chattable
	Callbacks    []tgbotapi.CallbackConfig
}

func newDummyBot() *dummyBot {
	return &dummyBot{
		ReplyError:   false,
		SentMessages: make([]tgbotapi.Chattable, 0),
		Callbacks:    make([]tgbotapi.CallbackConfig, 0),
	}
}

func (d *dummyBot) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	if d.ReplyError {
		return tgbotapi.Message{}, errors.New("dummy error")
	}
	d.SentMessages = append(d.SentMessages, msg)
	return tgbotapi.Message{MessageID: 333}, nil
}

func (d *dummyBot) GetFile(tgbotapi.FileConfig) (tgbotapi.File, error) {
	return tgbotapi.File{}, nil
}

func (d *dummyBot) AnswerCallbackQuery(config tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error) {
	if d.ReplyError {
		return tgbotapi.APIResponse{}, errors.New("dummy error")
	}
	d.Callbacks = append(d.Callbacks, config)
	return tgbotapi.APIResponse{}, nil
}

func (d *dummyBot) GetFileDirectURL(fileID string) (string, error) {
	return "", nil
}

func getSendMessageFunc(bot *dummyBot, id int64, text string) conversation.SpecialMessageFuncType {
	return func() error {
		_, err := bot.Send(tgbotapi.NewMessage(id, text))
		return err
	}
}

func TestNewConversation(t *testing.T) {
	checkErr := func(expected error, actual error) {
		if (expected == nil) != (actual == nil) || expected != nil && expected.Error() != actual.Error() {
			t.Errorf("Expected error \"%v\", actual \"%v\"", expected, actual)
		}
	}
	config := conversation.Config{
		MaxMessageQueue: 1,
		TimeoutMinutes:  1,
	}
	conv, err := conversation.NewConversation(1, nil, nil, nil, nil, nil, nil, config)
	checkErr(errors.New("cannot create conversation, bot object should not be nil"), err)
	if conv != nil {
		t.Error("conversation should not been created")
	}

	conv, err = conversation.NewConversation(3, &dummyBot{}, state.NewBotState(state.NewFileState("")), nil, nil, nil, nil, config)
	checkErr(nil, err)
	if conv == nil {
		t.Error("conversation should be created succesfully")
	}
}

func TestCancelByUser(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 1,
		TimeoutMinutes:  1,
	}
	bot := newDummyBot()
	cancelByUserFunc := getSendMessageFunc(bot, 5, "cancel by user")
	conv, err := conversation.NewConversation(5, bot, state.NewBotState(state.NewFileState("")),
		nil, nil, cancelByUserFunc, nil, config) //non-active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.CancelByUser() != nil {
		t.Error("conversation should be succesfully cancelled")
	}
	if len(bot.SentMessages) != 1 {
		t.Errorf("expected 1 message sent throug bot, got %d", len(bot.SentMessages))
	}

	conv, err = conversation.NewConversation(5, bot, state.NewBotState(state.NewFileState("")),
		nil, nil, cancelByUserFunc, nil, config) //active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 5}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("expected nil, got error %v", err)
	}
	if conv.CancelByUser() != nil {
		t.Error("conversation should be succesfully cancelled")
	}
	if len(bot.SentMessages) != 2 {
		t.Errorf("expected 2 message sent throug bot, got %d", len(bot.SentMessages))
	}

	bot.ReplyError = true
	cancelByUserFunc = getSendMessageFunc(bot, 7, "cancel by user")
	conv, err = conversation.NewConversation(7, bot, state.NewBotState(state.NewFileState("")),
		nil, nil, cancelByUserFunc, nil, config) //active conversation with broken bot
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	update = tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 7}, Text: "Some update"}}
	conv.PushUpdate(&update) // to activate the conversation
	err = conv.CancelByUser()
	if err == nil || err.Error() != fmt.Sprintf("error while sending cancel message to chat %d: %s", 7, "dummy error") {
		t.Errorf("unexpected error %v", err)
	}
}

func TestPushUpdate(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 1,
		TimeoutMinutes:  1,
	}

	bot := newDummyBot()
	tooManyMessagesFunc := getSendMessageFunc(bot, 11, "too many messages")
	conv, err := conversation.NewConversation(11, bot, state.NewBotState(state.NewFileState("")),
		tooManyMessagesFunc, nil, nil, nil, config) // active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{} // unsupported update type
	err = conv.PushUpdate(&update)
	if err == nil || err.Error() != "usupported query type" {
		t.Errorf("unexpected error: %v", err)
	}

	update = tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 10}, Text: "Some update"}} //wrong chat id
	err = conv.PushUpdate(&update)
	if err == nil || err.Error() != fmt.Sprintf("tried to process update from UserID %d in the conversation with UserID %d", 10, 11) {
		t.Errorf("unexpected error: %v", err)
	}

	update = tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 11}, Text: "Some update"}} //succesful update
	err = conv.PushUpdate(&update)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = conv.PushUpdate(&update) // second try should not fit the queue
	if err == nil || err.Error() != fmt.Sprintf("to many open unprocessed messages in the conversation (%v)", nil) {
		t.Errorf("unexpected error: %v", err)
	}

	if len(bot.SentMessages) != 1 {
		t.Errorf("expected 1 message sent to bot, got %d", len(bot.SentMessages))
	}
	expectedMessage, err := json.Marshal(tgbotapi.NewMessage(11, "too many messages"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	sentMessage, err := json.Marshal(bot.SentMessages[0])
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(sentMessage) != string(expectedMessage) {
		t.Errorf("expected message '%s', got '%s'", string(expectedMessage), string(sentMessage))
	}

	// check that the hander recieves exectly the message we pushed

	config.MaxMessageQueue = 2

	conv, err = conversation.NewConversation(12, bot, state.NewBotState(state.NewFileState("")),
		tooManyMessagesFunc, nil, nil, nil, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	firstUpdate := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 12}, Text: "first Text"}}
	secondUpdate := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 12}, Text: "second Text"}}

	err = conv.PushUpdate(&firstUpdate)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = conv.PushUpdate(&secondUpdate)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	firstSentMessage, exit := conv.GetFirstUpdateFromUser(ctx)
	if exit != false || firstUpdate != *firstSentMessage {
		t.Error("unexpected conversation first message")
	}
	secondSendMessage, exit := conv.GetUpdateFromUser(ctx)
	if exit != false && *secondSendMessage != secondUpdate {
		t.Error("unexpected second message")
	}
}

func TestMaxMessageQueue(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 5,
		TimeoutMinutes:  1,
	}

	bot := newDummyBot()
	tooManyMessagesFunc := getSendMessageFunc(bot, 13, "too many messages")
	conv, err := conversation.NewConversation(13, bot, state.NewBotState(state.NewFileState("")),
		tooManyMessagesFunc, nil, nil, nil, config) //closed conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 13}, Text: "Some update"}}
	for i := 0; i < 5; i++ {
		if err = conv.PushUpdate(&update); err != nil {
			t.Errorf("update should be sent successfuly, but got error %v", err)
		}
	}
	if err = conv.PushUpdate(&update); err == nil || err.Error() != fmt.Sprintf("to many open unprocessed messages in the conversation (%v)", nil) {
		t.Errorf("unexpected error: %v", err)
	}
	expectedMessage, err := json.Marshal(tgbotapi.NewMessage(13, "too many messages"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	sentMessage, err := json.Marshal(bot.SentMessages[0])
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(sentMessage) != string(expectedMessage) {
		t.Errorf("expected message '%s', got '%s'", string(expectedMessage), string(sentMessage))
	}
}

//
// Test readers.BotConversation interface, the way how the conversation is visible to a handler
//

func TestMessageCreation(t *testing.T) {
	var photoConfig tgbotapi.PhotoConfig
	var textMessage tgbotapi.MessageConfig
	config := conversation.Config{
		MaxMessageQueue: 1,
	}
	bot := newDummyBot()

	conv, _ := conversation.NewConversation(17, bot, state.NewBotState(state.NewFileState("")),
		nil, nil, nil, nil, config)
	photoConfig = conv.NewPhotoShare("photoID", "Caption")
	textMessage = conv.NewMessage("some text")

	if photoConfig.Caption != "Caption" || photoConfig.File != tgbotapi.FileID("photoID") || photoConfig.ChatID != 17 || photoConfig.ParseMode != "HTML" {
		t.Errorf("unexpected photo config %v", photoConfig)
	}
	if textMessage.ChatID != 17 || textMessage.Text != "some text" || textMessage.ParseMode != "HTML" {
		t.Errorf("unexpected message config %v", textMessage)
	}
}
