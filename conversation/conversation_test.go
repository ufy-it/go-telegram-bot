package conversation_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
	"ufygobot/conversation"

	"ufygobot/handlers"
	"ufygobot/handlers/readers"
	"ufygobot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
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

func (d *dummyBot) AnswerCallbackQuery(config tgbotapi.CallbackConfig) (tgbotapi.APIResponse, error) {
	if d.ReplyError {
		return tgbotapi.APIResponse{}, errors.New("dummy error")
	}
	d.Callbacks = append(d.Callbacks, config)
	return tgbotapi.APIResponse{}, nil
}

func newDummyHandlers() *handlers.CommandHandlers {
	return &handlers.CommandHandlers{
		Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
			<-time.After(1 * time.Millisecond)
			return nil
		}),
		List: make([]handlers.CommandHandler, 0),
	}
}

func wgWaitTimeout(timeout time.Duration, wg *sync.WaitGroup) bool {
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()
	select {
	case <-time.After(timeout):
		return false
	case <-c:
		return true
	}
}

func wait(timeout time.Duration) {
	<-time.After(timeout)
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
		FinishSeconds:   1,
		Handlers:        newDummyHandlers(),
	}
	conv, err := conversation.NewConversation(1, nil, nil, nil, config)
	checkErr(errors.New("cannot create conversation, wait group should not be nil"), err)
	if conv != nil {
		t.Error("conversation should not been created")
	}

	wg := sync.WaitGroup{}
	conv, err = conversation.NewConversation(2, nil, nil, &wg, config)
	checkErr(errors.New("cannot create conversation, bot object should not be nil"), err)
	if conv != nil {
		t.Error("conversation should not been created")
	}
	if !wgWaitTimeout(1*time.Millisecond, &wg) {
		t.Error("Waight group should be closed due to invalid conversation configuration")
	}

	wg2 := sync.WaitGroup{}
	conv, err = conversation.NewConversation(3, &dummyBot{}, state.NewBotState(), &wg2, config)
	checkErr(nil, err)
	if conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.IsClosed() {
		t.Error("conversation should not be closed")
	}
	wait(1 * time.Millisecond) // let's wait to ensure that handler's thread started
	if wgWaitTimeout(10*time.Millisecond, &wg2) {
		t.Error("waiting group should not be done")
	}
	conv.Kill()
	if !wgWaitTimeout(10*time.Millisecond, &wg2) {
		t.Error("handlers should be killed in 10 ms")
	}
	if !conv.IsClosed() {
		t.Error("conversation should be closed after calling Kill() method")
	}
}

func TestIsClosed(t *testing.T) {
	config := conversation.Config{}
	wg := sync.WaitGroup{}
	conv, err := conversation.NewConversation(4, newDummyBot(), state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.IsClosed() {
		t.Error("conversation should not been closed")
	}
	conv.Kill()
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
	conv.Kill()
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
}

func TestTimeout(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 1,
		TimeoutMinutes:  1,
		Handlers:        newDummyHandlers(),
	}
	wg := sync.WaitGroup{}
	conv, err := conversation.NewConversation(4, &dummyBot{}, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	timeout, err := conv.Timeout()
	if timeout || err != nil {
		t.Error("the conversation should not timeout")
	}
	conv.Kill()
	timeout, err = conv.Timeout()
	if !timeout {
		t.Error("killed conversation should timeout")
	}
	if err == nil || err.Error() != "the conversaton is already closed" {
		t.Error("unexpected error")
	}
	config.TimeoutMinutes = 0 // timeout immidiatelly
	conv, err = conversation.NewConversation(4, &dummyBot{}, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	wait(1 * time.Millisecond) // wait for a timeout
	timeout, err = conv.Timeout()
	if !timeout {
		t.Error("killed conversation should timeout")
	}
	if err != nil {
		t.Error("unexpected error")
	}
}

func TestCancelByUser(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue:    1,
		Handlers:           newDummyHandlers(),
		CloseByUserMessage: "user canceled conversation",
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(5, bot, state.NewBotState(), &wg, config) //non-active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.CancelByUser() != nil {
		t.Error("conversation should be succesfully cancelled")
	}
	if len(bot.SentMessages) != 0 {
		t.Errorf("expected 0 message sent throug bot, got %d", len(bot.SentMessages))
	}

	conv, err = conversation.NewConversation(5, bot, state.NewBotState(), &wg, config) //active conversation
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
	if len(bot.SentMessages) != 1 {
		t.Errorf("expected 0 message sent throug bot, got %d", len(bot.SentMessages))
	}
	sentMessage, err := json.Marshal(bot.SentMessages[0])
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	msg := tgbotapi.NewMessage(5, config.CloseByUserMessage)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	expectedMessage, err := json.Marshal(msg)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if string(sentMessage) != string(expectedMessage) {
		t.Errorf("message about user cancel '%s' differs from expected '%s'", string(sentMessage), string(expectedMessage))
	}

	bot = newDummyBot()
	conv, err = conversation.NewConversation(6, bot, state.NewBotState(), &wg, config) //closed conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	conv.Kill()
	if conv.CancelByUser() != nil {
		t.Error("conversation should be succesfully canceled")
	}
	if len(bot.SentMessages) > 0 {
		t.Errorf("expected 0 messages through the bot, got %d", len(bot.SentMessages))
	}

	bot.ReplyError = true
	conv, err = conversation.NewConversation(7, bot, state.NewBotState(), &wg, config) //active conversation with broken bot
	update = tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 7}, Text: "Some update"}}
	conv.PushUpdate(&update) // to activate the conversation
	err = conv.CancelByUser()
	if err == nil || err.Error() != fmt.Sprintf("Error while sending cancel-by-user message to chat %d: %s", 7, "dummy error") {
		t.Errorf("unexpected error %v", err)
	}
}

func TestKill(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue:   1,
		Handlers:          newDummyHandlers(),
		CloseByBotMessage: "bot closed conversation",
	}

	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(8, bot, state.NewBotState(), &wg, config) //non-active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.IsClosed() {
		t.Error("conversation should not be closed")
	}
	conv.Kill()
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
	if len(bot.SentMessages) > 0 {
		t.Error("bot should not send any messages")
	}
	conv.Kill() //try it twice
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
	if len(bot.SentMessages) > 0 {
		t.Error("bot should not send any messages")
	}

	conv, err = conversation.NewConversation(9, bot, state.NewBotState(), &wg, config) //active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	if conv.IsClosed() {
		t.Error("conversation should not be closed")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 9}, Text: "Some update"}}
	err = conv.PushUpdate(&update) // to activate the conversation
	if err != nil {
		t.Error("conversation should allow receiving messages")
	}
	conv.Kill() //try it twice
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
	if len(bot.SentMessages) != 1 {
		t.Errorf("expected 1 message, got %d", len(bot.SentMessages))
	}
	sentMessage, err := json.Marshal(bot.SentMessages[0])
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	msg := tgbotapi.NewMessage(9, config.CloseByBotMessage)
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	expectedMessage, err := json.Marshal(msg)
	if string(sentMessage) != string(expectedMessage) {
		t.Errorf("expected message '%s', got '%s'", string(expectedMessage), string(sentMessage))
	}
	conv.Kill() //try it twice
	if !conv.IsClosed() {
		t.Error("conversation should be closed")
	}
	if len(bot.SentMessages) != 1 {
		t.Errorf("expected 1 message, got %d", len(bot.SentMessages))
	}
}

func TestPushUpdate(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue:       1,
		Handlers:              newDummyHandlers(),
		ToManyMessagesMessage: "too many messages",
	}

	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(10, bot, state.NewBotState(), &wg, config) //closed conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	conv.Kill() // to close the conversation
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 10}, Text: "Some update"}}
	err = conv.PushUpdate(&update)
	if err == nil || err.Error() != "the conversation is closed" {
		t.Errorf("unexpected error: %v", err)
	}

	conv, err = conversation.NewConversation(11, bot, state.NewBotState(), &wg, config) // active conversation
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update = tgbotapi.Update{} // unsupported update type
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
	expectedMessage, err := json.Marshal(tgbotapi.NewMessage(11, config.ToManyMessagesMessage))
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
	var recievedMessage *tgbotapi.Update
	var firstSentMessage *tgbotapi.Message
	var recievedExit bool
	config.MaxMessageQueue = 2
	wg2 := sync.WaitGroup{}
	testHandler := handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
		firstSentMessage = firstMessage
		update, exit := conversation.GetUpdateFromUser()
		recievedMessage = update
		recievedExit = exit
		return nil
	})
	config.Handlers = &handlers.CommandHandlers{
		Default: testHandler,
		List:    []handlers.CommandHandler{},
	}

	conv, err = conversation.NewConversation(12, bot, state.NewBotState(), &wg2, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	firstUpdate := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 12}, Text: "first Text"}}
	secondUpdate := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 12}, Text: "second Text"}}
	wait(1 * time.Millisecond) //wait for handler to start
	err = conv.PushUpdate(&firstUpdate)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = conv.PushUpdate(&secondUpdate)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond) // wait for handler to proceed
	conv.Kill()                // close conversation
	if !wgWaitTimeout(2*time.Millisecond, &wg2) {
		t.Error("handler did not finish in 2 ms")
	}
	if recievedExit {
		t.Error("unexpected exit signal")
	}
	if firstUpdate.Message != firstSentMessage {
		t.Error("unexpected conversation first message")
	}
	if recievedMessage != &secondUpdate {
		t.Error("unexpected second message")
	}
}

func TestMaxMessageQueue(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 5,
		Handlers:        newDummyHandlers(),
	}

	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(13, bot, state.NewBotState(), &wg, config) //closed conversation
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
	expectedMessage, err := json.Marshal(tgbotapi.NewMessage(13, config.ToManyMessagesMessage))
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

func TestFinishConversation(t *testing.T) {
	config := conversation.Config{
		MaxMessageQueue: 1,
		FinishSeconds:   1,
		Handlers:        newDummyHandlers(),
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(14, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 14}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	wait(500 * time.Millisecond) // after this time the conversation still should be alive
	timeout, err := conv.Timeout()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if timeout {
		t.Error("conversation should not timeout after 500ms")
	}
	wait(510 * time.Millisecond)
	timeout, err = conv.Timeout()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !timeout {
		t.Error("conversation should timeout after 1 s")
	}
}

func TestChatID(t *testing.T) {
	var chatID int64
	config := conversation.Config{
		MaxMessageQueue: 1,
		Handlers: &handlers.CommandHandlers{
			Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
				chatID = conversation.ChatID()
				return nil
			}),
			List: []handlers.CommandHandler{},
		},
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(15, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 15}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	wait(1 * time.Millisecond) // wait for handler to process
	if wgWaitTimeout(2*time.Millisecond, &wg) {
		t.Error("handler should finninsh in 2 ms")
	}
	if chatID != 15 {
		t.Errorf("expected chat id 15, got %d", chatID)
	}
}

func TestGetUpdateFromUser(t *testing.T) {
	var firstUpdate *tgbotapi.Update
	var secondUpdate *tgbotapi.Update
	var firstExit bool
	var secondExit bool
	config := conversation.Config{
		MaxMessageQueue: 5,
		Handlers: &handlers.CommandHandlers{
			Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
				firstUpdate, firstExit = conversation.GetUpdateFromUser()
				secondUpdate, secondExit = conversation.GetUpdateFromUser()
				return nil
			}),
			List: []handlers.CommandHandler{},
		},
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(16, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 16}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond) //wait for handler
	conv.Kill()                // cancel conversation
	if !wgWaitTimeout(1*time.Millisecond, &wg) {
		t.Error("handler should be finished in 1 ms")
	}
	if firstExit || firstUpdate != &update {
		t.Error("unexpected first update")
	}
	if secondUpdate != nil || !secondExit {
		t.Error("second update should be about exit")
	}
}

func TestMessageCreation(t *testing.T) {
	var photoConfig tgbotapi.PhotoConfig
	var textMessage tgbotapi.MessageConfig
	config := conversation.Config{
		MaxMessageQueue: 1,
		Handlers: &handlers.CommandHandlers{
			Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
				photoConfig = conversation.NewPhotoShare("photoID", "Caption")
				textMessage = conversation.NewMessage("some text")
				return nil
			}),
			List: []handlers.CommandHandler{},
		},
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(17, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 17}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond)
	conv.Kill()
	if !wgWaitTimeout(1*time.Millisecond, &wg) {
		t.Error("handler should finish in 1 ms")
	}
	if photoConfig.Caption != "Caption" || photoConfig.FileID != "photoID" || photoConfig.ChatID != 17 || photoConfig.ParseMode != "HTML" {
		t.Errorf("unexpected photo config %v", photoConfig)
	}
	if textMessage.ChatID != 17 || textMessage.Text != "some text" || textMessage.ParseMode != "HTML" {
		t.Errorf("unexpected message config %v", textMessage)
	}
}

func helperPrepareConversationForSend(t *testing.T, send func(conversation readers.BotConversation) (int, error)) *dummyBot {
	var sendErr error
	var sendID int
	config := conversation.Config{
		MaxMessageQueue: 1,
		Handlers: &handlers.CommandHandlers{
			Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
				sendID, sendErr = send(conversation)
				return nil
			}),
			List: []handlers.CommandHandler{},
		},
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(312, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 312}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond)
	if sendID != 333 || sendErr != nil {
		t.Errorf("unexpected results id %d, error %v", sendID, sendErr)
	}
	sentMessages := len(bot.SentMessages)
	sentCallbacks := len(bot.Callbacks)
	if sentMessages < 0 || sentCallbacks < 0 || sentMessages+sentCallbacks != 1 {
		t.Errorf("expected 1 sent message, got %d", sentMessages+sentCallbacks)
	}
	bot.ReplyError = true
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond)
	conv.Kill()
	if !wgWaitTimeout(1*time.Millisecond, &wg) {
		t.Error("handler should finish in 1 ms")
	}
	if sendID != 0 || sendErr == nil || sendErr.Error() != "dummy error" {
		t.Errorf("unexpected results id %d, error %v", sendID, sendErr)
	}
	if len(bot.SentMessages) != sentMessages {
		t.Errorf("expected 1 sent message, got %d", len(bot.SentMessages))
	}
	if len(bot.Callbacks) != sentCallbacks {
		t.Errorf("expected 1 sent callback, got %d", len(bot.Callbacks))
	}
	return bot
}

func helperTestSendOneMessage(t *testing.T, send func(conversation readers.BotConversation) (int, error), expected tgbotapi.Chattable) {
	bot := helperPrepareConversationForSend(t, send)
	sentMsg, err := json.Marshal(bot.SentMessages[0])
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	expectedMsg, err := json.Marshal(expected)
	if string(sentMsg) != string(expectedMsg) {
		t.Errorf("expected message '%s', got '%s'", string(expectedMsg), string(sentMsg))
	}

}

func TestSendGeneralMessage(t *testing.T) {
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			return conversation.SendGeneralMessage(tgbotapi.NewMessage(22, "some text"))
		},
		tgbotapi.NewMessage(22, "some text"),
	)
}

func TestSendGeneralMessageConversationClose(t *testing.T) {
	var sendErr error
	var sendID int
	config := conversation.Config{
		MaxMessageQueue: 1,
		Handlers: &handlers.CommandHandlers{
			Default: handlers.OneStepHandlerCreator(func(conversation readers.BotConversation, firstMessage *tgbotapi.Message) error {
				wait(2 * time.Millisecond)
				sendID, sendErr = conversation.SendGeneralMessage(tgbotapi.NewMessage(22, "some text"))
				return nil
			}),
			List: []handlers.CommandHandler{},
		},
	}
	wg := sync.WaitGroup{}
	bot := newDummyBot()
	conv, err := conversation.NewConversation(19, bot, state.NewBotState(), &wg, config)
	if err != nil || conv == nil {
		t.Error("conversation should be created succesfully")
	}
	update := tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 19}, Text: "Some update"}}
	if err = conv.PushUpdate(&update); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	wait(1 * time.Millisecond)
	conv.Kill()
	if !wgWaitTimeout(2*time.Millisecond, &wg) {
		t.Error("handler should finish in 2 ms")
	}
	if sendID != 0 || sendErr == nil || sendErr.Error() != "the conversation is closed" {
		t.Errorf("unexpected results id %d, error %v", sendID, sendErr)
	}
	if len(bot.SentMessages) != 1 { // 1 message because we close in the middle of a conversaton. So bot will send a message about it
		t.Errorf("expected 1 sent message, got %d", len(bot.SentMessages))
	}
}

func TestSendText(t *testing.T) {
	message := tgbotapi.NewMessage(312, "Some text")
	message.ParseMode = "HTML"
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			return conversation.SendText("Some text")
		},
		message,
	)
}

func TestReplyWithText(t *testing.T) {
	message := tgbotapi.NewMessage(312, "Some reply")
	message.ParseMode = "HTML"
	message.ReplyToMessageID = 777
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			return conversation.ReplyWithText("Some reply", 777)
		},
		message,
	)
}

func TestAnswerButton(t *testing.T) {
	bot := helperPrepareConversationForSend(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.AnswerButton("buttonID")
			if err == nil {
				return 333, err
			}
			return 0, err
		})
	if len(bot.Callbacks) == 0 {
		t.Errorf("expected callback sent")
	}
	if bot.Callbacks[0] != tgbotapi.NewCallback("buttonID", "") {
		t.Errorf("Expected callback %v, got %v", tgbotapi.NewCallback("buttonID", ""), bot.Callbacks[0])
	}
}

func TestDeleteMessage(t *testing.T) {
	message := tgbotapi.NewDeleteMessage(312, 888)
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.DeleteMessage(888)
			if err == nil {
				return 333, err
			}
			return 0, err
		},
		message,
	)
}

func TestRemoveReplyMarkup(t *testing.T) {
	message := tgbotapi.NewEditMessageReplyMarkup(312, 999, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: make([][]tgbotapi.InlineKeyboardButton, 0)})
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.RemoveReplyMarkup(999)
			if err == nil {
				return 333, err
			}
			return 0, err
		},
		message,
	)
}

func TestEditReplyMarkup(t *testing.T) {
	keyboard := make([][]tgbotapi.InlineKeyboardButton, 1)
	keyboard[0] = make([]tgbotapi.InlineKeyboardButton, 1)
	keyboard[0][0] = tgbotapi.NewInlineKeyboardButtonData("Text", "Data")
	message := tgbotapi.NewEditMessageReplyMarkup(312, 9999, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard})
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.EditReplyMarkup(9999, tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard})
			if err == nil {
				return 333, err
			}
			return 0, err
		},
		message,
	)
}

func TestEditMessageText(t *testing.T) {
	message := tgbotapi.NewEditMessageText(312, 8888, "New text")
	message.ParseMode = "HTML"
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.EditMessageText(8888, "New text")
			if err == nil {
				return 333, err
			}
			return 0, err
		},
		message,
	)
}

func TestEditMessageTextAndInlineMarkup(t *testing.T) {
	message := tgbotapi.NewEditMessageText(312, 8888, "New text")
	message.ParseMode = "HTML"
	keyboard := make([][]tgbotapi.InlineKeyboardButton, 1)
	keyboard[0] = make([]tgbotapi.InlineKeyboardButton, 1)
	keyboard[0][0] = tgbotapi.NewInlineKeyboardButtonData("Text", "Data")
	message.ReplyMarkup = &tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
	helperTestSendOneMessage(t,
		func(conversation readers.BotConversation) (int, error) {
			err := conversation.EditMessageTextAndInlineMarkup(8888, "New text", tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard})
			if err == nil {
				return 333, err
			}
			return 0, err
		},
		message,
	)
}
