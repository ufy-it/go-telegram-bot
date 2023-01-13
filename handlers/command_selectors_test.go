package handlers_test

import (
	"context"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ufy-it/go-telegram-bot/handlers"
)

func TestRegExpCommandSelector(t *testing.T) {
	selector := handlers.RegExpCommandSelector("/test\\d*_")
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_"}}) {
		t.Error("/test_ command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest"}}) {
		t.Error("/fest command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test1e"}}) {
		t.Error("/test1e command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test33_"}}) {
		t.Error("/test33_ command selector failed")
	}
}

func TestCallbackCommandSelector(t *testing.T) {
	selector := handlers.RegExCallbackSelector("/test\\d*_")
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/test_"}}) {
		t.Error("/test_ callback selector failed")
	}
	if selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/fest"}}) {
		t.Error("/fest callback selector failed")
	}
	if selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/test1e"}}) {
		t.Error("/test1e callback selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/test33_"}}) {
		t.Error("/test33_ callback selector failed")
	}
}

func TestCommandOrCallbackRegExpSelector(t *testing.T) {
	selector := handlers.CommandOrCallbackRegExpSelector("/test\\d*_", "/callback\\d*_")
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_"}}) {
		t.Error("/test_ command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/callback_"}}) {
		t.Error("/callback_ callback selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest"}}) {
		t.Error("/fest command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/callback1e"}}) {
		t.Error("/callback1e callback selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test33_"}}) {
		t.Error("/test33_ command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{Data: "/callback33_"}}) {
		t.Error("/callback33_ callback selector failed")
	}
}

func TestImageSelector(t *testing.T) {
	selector := handlers.ImageSelector()
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Photo: []tgbotapi.PhotoSize{}}}) {
		t.Error("image selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_"}}) {
		t.Error("text selector failed")
	}
}

func TestAndSelector(t *testing.T) {
	selector := handlers.AndSelector(handlers.RegExpCommandSelector("/test\\d*_"), handlers.ImageSelector())
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_", Photo: []tgbotapi.PhotoSize{}}}) {
		t.Error("/test_ command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest", Photo: []tgbotapi.PhotoSize{}}}) {
		t.Error("/fest command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test1_"}}) {
		t.Error("text1) selector failed")
	}
}

func TestOrSelector(t *testing.T) {
	selector := handlers.OrSelector(handlers.RegExpCommandSelector("/test\\d*_"), handlers.ImageSelector())
	ctx := context.Background()
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_", Photo: []tgbotapi.PhotoSize{}}}) {
		t.Error("/test_ command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest", Photo: []tgbotapi.PhotoSize{}}}) {
		t.Error("/fest command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test1_"}}) {
		t.Error("text1) selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest"}}) {
		t.Error("/fest command selector failed")
	}
}

func TestNotSelector(t *testing.T) {
	selector := handlers.NotSelector(handlers.RegExpCommandSelector("/test\\d*_"))
	ctx := context.Background()
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test_"}}) {
		t.Error("/test_ command selector failed")
	}
	if !selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/fest"}}) {
		t.Error("/fest command selector failed")
	}
	if selector(ctx, &tgbotapi.Update{Message: &tgbotapi.Message{Text: "/test1_"}}) {
		t.Error("text1_ selector failed")
	}
}
