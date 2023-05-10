package jobs_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
	"github.com/ufy-it/go-telegram-bot/jobs"
)

type mockMessage struct {
	ChatID int64
	Text   string
}

type mockMessager struct {
	messages []mockMessage
}

func newMockMessager() *mockMessager {
	return &mockMessager{
		messages: make([]mockMessage, 0),
	}
}

func (m *mockMessager) SendSingleMessage(ctx context.Context, chatID int64, text string, keyboard *buttons.ButtonSet) error {
	m.messages = append(m.messages, mockMessage{chatID, text})
	return nil
}

func (m *mockMessager) SendSinglePhoto(ctx context.Context, chatID int64, photo []byte, caption string, keyboard *buttons.ButtonSet) error {
	return nil
}

// Test that a job starts after an offset and runs each second
func TestJobIntervals(t *testing.T) {
	c := make(chan struct{})
	ind := 0
	job := func(ctx context.Context, messager jobs.Messager) error {
		if ind < 3 {
			c <- struct{}{}
			messager.SendSingleMessage(ctx, int64(ind), strconv.Itoa(ind), nil)
			ind++
		}
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startPoint := time.Now()
	messager := newMockMessager()
	go jobs.RunJob(ctx,
		jobs.JobDescription{
			OffsetSeconds:   2,
			IntervalSeconds: 1,
			Body:            job,
		}, messager)
	var timePoint time.Time
	checkStep := func(targetIntervalSeconds int) error {
		select {
		case <-c:
			timePoint = time.Now()
			if startPoint.Add(time.Duration(targetIntervalSeconds*1000-500)*time.Millisecond).After(timePoint) ||
				startPoint.Add(time.Duration(targetIntervalSeconds*1000+500)*time.Millisecond).Before(timePoint) {
				return fmt.Errorf("Interval duration is not in expected interval. Expected %d - milliseconds, actual - %d milliseconds",
					targetIntervalSeconds*1000, timePoint.Sub(startPoint).Milliseconds())
			}
			return nil
		case <-time.After(time.Duration(5) * time.Second):
			return errors.New("Job did not produce output in time")
		}
	}
	if err := checkStep(2); err != nil {
		t.Error(err.Error())
	}
	startPoint = timePoint
	if err := checkStep(1); err != nil {
		t.Error(err.Error())
	}
	startPoint = timePoint
	if err := checkStep(1); err != nil {
		t.Error(err.Error())
	}

	if len(messager.messages) != 3 {
		t.Errorf("expected %d messages sent, got %d", 3, len(messager.messages))
	}

	for i, message := range messager.messages {
		if int64(i) != message.ChatID || strconv.Itoa(i) != message.Text {
			t.Errorf("unexpected message (%d, %s)", message.ChatID, message.Text)
		}
	}
}

func TestThreeJobsParallelRun(t *testing.T) {
	c := make(chan int, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	genJob := func(i int) func(ctx context.Context, messager jobs.Messager) error {
		return func(ctx context.Context, messager jobs.Messager) error {
			c <- i
			return nil
		}
	}
	jobs.RunJobs(ctx, jobs.JobDescriptionsList{
		{0, 3, genJob(1)},
		{1, 3, genJob(2)},
		{2, 3, genJob(3)},
	}, newMockMessager())
	checkStep := func(i int) {
		select {
		case x := <-c:
			if x != i {
				t.Errorf("unexpected job result. Expected %d, got %d", i, x)
			}
		case <-time.After(time.Duration(2) * time.Second):
			t.Error("no job result in 2 seconds")
		}
	}
	checkStep(1)
	checkStep(2)
	checkStep(3)
	checkStep(1)
	checkStep(2)
	checkStep(3)
}

func TestCloseBeforeOffset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan struct{})
	go jobs.RunJob(ctx,
		jobs.JobDescription{1, 1, func(ctx context.Context, messager jobs.Messager) error {
			c <- struct{}{}
			return nil
		}}, newMockMessager())
	cancel()
	select {
	case <-c:
		t.Error("job did not stop")
	case <-time.After(time.Duration(2) * time.Second):
	}
}

func TestCloseInInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan struct{})
	go jobs.RunJob(ctx,
		jobs.JobDescription{1, 1, func(ctx context.Context, messager jobs.Messager) error {
			c <- struct{}{}
			return nil
		}}, newMockMessager())
	select {
	case <-c:
	case <-time.After(time.Duration(2) * time.Second):
		t.Errorf("no job run during 2 seconds")
	}
	cancel()
	select {
	case <-c:
		t.Error("job did not stop")
	case <-time.After(time.Duration(2) * time.Second):
	}
}
