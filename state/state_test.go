package state_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ufy-it/go-telegram-bot/state"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

func TestGetConversations(t *testing.T) {
	s := state.NewBotState(state.NewFileState(""))
	if s == nil {
		t.Error("cannot create state object")
	}
	if len(s.GetConversations()) > 0 {
		t.Errorf("Expected empty conversation list, got list with %d items", len(s.GetConversations()))
	}

	s.GetConversatonFirstUpdate(11)
	s.StartConversationWithUpdate(12, nil)
	s.SaveConversationStepAndData(13, 1, nil)

	cs := s.GetConversations()
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
	if !reflect.DeepEqual(cs, []int64{11, 12, 13}) {
		t.Errorf("Expected conversations [11, 12, 13], got %v", cs)
	}

	s.RemoveConverastionState(12)
	cs = s.GetConversations()
	sort.Slice(cs, func(i, j int) bool { return cs[i] < cs[j] })
	if !reflect.DeepEqual(cs, []int64{11, 13}) {
		t.Errorf("Expected conversations [11, 13], got %v", cs)
	}
}

func TestClose(t *testing.T) {
	s := state.NewBotState(state.NewFileState(""))
	if s == nil {
		t.Error("cannot create state object")
	}
	err := s.StartConversationWithUpdate(1, nil)
	if err != nil && strings.Contains(err.Error(), "the state is closed for saving") {
		t.Errorf("Unexpected error %v", err)
	}
	err = s.Close()
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	err = s.StartConversationWithUpdate(2, nil)
	if err == nil || !strings.Contains(err.Error(), "the state is closed for saving") {
		t.Errorf("Unexpected error %v", err)
	}
	err = s.Close()
	if err == nil || !strings.Contains(err.Error(), "state is already closed") {
		t.Errorf("Unexpected error %v", err)
	}
}

func TestGetConversatonFirstUpdate(t *testing.T) {
	s := state.NewBotState(state.NewFileState(""))
	if s == nil {
		t.Error("cannot create state object")
	}
	u := s.GetConversatonFirstUpdate(1)
	if u != nil {
		t.Errorf("Expected nil but got value %v", *u)
	}
	s.StartConversationWithUpdate(2, &tgbotapi.Update{UpdateID: 555})
	u = s.GetConversatonFirstUpdate(2)
	if u == nil || u.UpdateID != 555 {
		t.Errorf("Unexpected first update %v", u)
	}
}

func TestGetConversationStepAnddata(t *testing.T) {
	s := state.NewBotState(state.NewFileState(""))
	if s == nil {
		t.Error("cannot create state object")
	}
	step, data := s.GetConversationStepAndData(1)
	if step != 0 || data != nil {
		t.Errorf("expected (0, nil) got (%d, %v)", step, data)
	}
	s.SaveConversationStepAndData(1, 2, struct{ A int }{5})
	step, data = s.GetConversationStepAndData(1)
	if step != 2 || data == nil {
		t.Errorf("unexpected step and data (%d, %v)", step, data)
	}
}
