package jobs_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"ufygobot/jobs"
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

func (m *mockMessager) SendSingleMessage(chatID int64, text string) error {
	m.messages = append(m.messages, mockMessage{chatID, text})
	return nil
}

// Test that a job starts after an offset and runs each second
func TestJobManagerJobIntervals(t *testing.T) {
	c := make(chan struct{})
	ind := 0
	job := func(messager jobs.Messager) error {
		if ind < 3 {
			c <- struct{}{}
			messager.SendSingleMessage(int64(ind), strconv.Itoa(ind))
			ind++
		}
		return nil
	}
	startPoint := time.Now()
	messager := newMockMessager()
	manager := jobs.NewJobManager(jobs.JobDescriptionsList{
		jobs.JobDescription{
			OffsetSeconds:   2,
			IntervalSeconds: 1,
			Body:            job,
		},
	}, messager)
	defer func() {
		if err := manager.Stop(1); err != nil {
			t.Error(err.Error())
		}
	}()
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
	genJob := func(i int) func(messager jobs.Messager) error {
		return func(messager jobs.Messager) error {
			c <- i
			return nil
		}
	}
	manager := jobs.NewJobManager(jobs.JobDescriptionsList{
		{0, 3, genJob(1)},
		{1, 3, genJob(2)},
		{2, 3, genJob(3)},
	}, newMockMessager())
	defer func() {
		if err := manager.Stop(1); err != nil {
			t.Error(err.Error())
		}
	}()
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
	c := make(chan struct{})
	manager := jobs.NewJobManager(jobs.JobDescriptionsList{
		{1, 1, func(messager jobs.Messager) error {
			c <- struct{}{}
			return nil
		}},
	}, newMockMessager())
	if err := manager.Stop(1); err != nil {
		t.Errorf("JobManager did not stop: %v", err)
	}
	select {
	case <-c:
		t.Error("job did not stop")
	case <-time.After(time.Duration(2) * time.Second):
	}
}

func TestCloseInInterval(t *testing.T) {
	c := make(chan struct{})
	manager := jobs.NewJobManager(jobs.JobDescriptionsList{
		{1, 1, func(messager jobs.Messager) error {
			c <- struct{}{}
			return nil
		}},
	}, newMockMessager())
	select {
	case <-c:
	case <-time.After(time.Duration(2) * time.Second):
		t.Errorf("no job run during 2 seconds")
	}
	if err := manager.Stop(1); err != nil {
		t.Errorf("JobManager did not stop: %v", err)
	}
	select {
	case <-c:
		t.Error("job did not stop")
	case <-time.After(time.Duration(2) * time.Second):
	}
}
