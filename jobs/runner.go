package jobs

import (
	"context"
	"time"

	"github.com/ufy-it/go-telegram-bot/handlers/buttons"
	"github.com/ufy-it/go-telegram-bot/logger"
)

// Messager is an interface for an object that can send a text message to a chat. In production it should be a Dispatcher object
type Messager interface {
	SendSingleMessage(ctx context.Context, chatID int64, text string, keyboard *buttons.ButtonSet) error                // send single text message to chat with ID=chatID, might take time
	SendSinglePhoto(ctx context.Context, chatID int64, photo []byte, caption string, keyboard *buttons.ButtonSet) error // send single photo to chat with ID=chatID, might take time
}

// JobBody a type for a job-function
type JobBody func(ctx context.Context, messager Messager) error

// JobDescription is a struct that describes a job.
// The job will be started after OffsetSeconds, and will run every IntervalSeconds
type JobDescription struct {
	OffsetSeconds   int
	IntervalSeconds int
	Body            JobBody
}

// JobDescriptionsList is a type for array of JobDescriptions
type JobDescriptionsList []JobDescription

// RunJob runs a job's Body continuously with offset and interval
func RunJob(ctx context.Context, job JobDescription, messager Messager) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(time.Duration(job.OffsetSeconds) * time.Second):
	}
	for {
		err := job.Body(ctx, messager)
		if err != nil {
			logger.Error("got error from a job: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(job.IntervalSeconds) * time.Second):
		}
	}
}

// RunJobs starts all jobs from the list in a separate threadsdefer cancel()
func RunJobs(ctx context.Context, jobs JobDescriptionsList, messager Messager) {
	for _, job := range jobs {
		go RunJob(ctx, job, messager)
	}
}
