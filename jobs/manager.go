package jobs

import (
	"errors"
	"sync"
	"time"

	"github.com/ufy-it/go-telegram-bot/logger"
)

// Messager is an interface for an object that can send a text message to a chat. In production it should be a Dispatcher object
type Messager interface {
	SendSingleMessage(chatID int64, text string) error // send single text message to chat with ID=chatID, might take time
}

// JobBody a type for a job-function
type JobBody func(messager Messager) error

// JobDescription is a struct that describes a job.
// The job will be started after OffsetSeconds, and will run every IntervalSeconds
type JobDescription struct {
	OffsetSeconds   int
	IntervalSeconds int
	Body            JobBody
}

// JobDescriptionsList is a type for array of JobDescriptions
type JobDescriptionsList []JobDescription

// JobManager runs each job in a separate thread
type JobManager struct {
	jobs     JobDescriptionsList
	messager Messager
	stopped  bool
	exit     []chan struct{}
	wg       sync.WaitGroup
}

// NewJobManager constructs new manager object and starts all jobs
func NewJobManager(jobs JobDescriptionsList, messager Messager) *JobManager {
	manager := JobManager{
		jobs:     jobs,
		messager: messager,
		exit:     make([]chan struct{}, len(jobs)),
		stopped:  false,
		wg:       sync.WaitGroup{},
	}
	manager.startAllJobs()
	return &manager
}

// Stop stops all jobs. If not all jobs were stopped within the timeout, the method will return error
func (jm *JobManager) Stop(timeoutSeconds int) error {
	if jm.stopped {
		return errors.New("job manager already stopped")
	}
	jm.stopped = true
	for _, ex := range jm.exit {
		close(ex) // notify all runners
	}
	c := make(chan struct{})
	go func() {
		defer close(c)
		jm.wg.Wait()
	}()
	select {
	case <-c:
		return nil
	case <-time.After(time.Duration(timeoutSeconds) * time.Second):
		return errors.New("some jobs were not finished in time")
	}
}

// run a job's Body continuously with offset and interval
func (jm *JobManager) runJob(job *JobDescription, exit *chan struct{}) {
	jm.wg.Add(1)
	defer jm.wg.Done()
	select {
	case <-*exit:
		return
	case <-time.After(time.Duration(job.OffsetSeconds) * time.Second):
	}
	for {
		err := job.Body(jm.messager)
		if err != nil {
			logger.Error("got error from a job: %v", err)
		}
		select {
		case <-*exit:
			return
		case <-time.After(time.Duration(job.IntervalSeconds) * time.Second):
		}
	}
}

func (jm *JobManager) startAllJobs() {
	for idx, j := range jm.jobs {
		jm.exit[idx] = make(chan struct{})
		job := j // local copy to deal with capture by refference
		go jm.runJob(&job, &jm.exit[idx])
	}
}
