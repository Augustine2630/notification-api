package job

import (
	"log"

	"notification-api/internal/notification/service"

	"github.com/robfig/cron/v3"
)

// Scheduler runs a fixed set of announcement Tasks on their cron schedules.
// Tasks are loaded once from a JSON file at startup (see LoadTasks) — adding
// or changing a task requires restarting the service.
type Scheduler struct {
	notifier *service.NotificationService
	cron     *cron.Cron
}

// NewScheduler builds a Scheduler and registers every task from the given
// jobs file. It returns an error if the file can't be read/parsed, or if any
// task's cron expression is invalid.
func NewScheduler(notifier *service.NotificationService, jobsFilePath string) (*Scheduler, error) {
	tasks, err := LoadTasks(jobsFilePath)
	if err != nil {
		return nil, err
	}

	s := &Scheduler{
		notifier: notifier,
		cron:     cron.New(),
	}

	for _, t := range tasks {
		if err := s.schedule(t); err != nil {
			return nil, err
		}
	}

	log.Printf("Scheduler: loaded %d job(s) from %s", len(tasks), jobsFilePath)
	return s, nil
}

func (s *Scheduler) schedule(t Task) error {
	task := t // capture for closure
	_, err := s.cron.AddFunc(task.Cron, func() {
		log.Printf("Scheduler: running job (cron=%q, recipients=%d)", task.Cron, len(task.UserIDs))
		result := s.notifier.SendAnnounce(task.UserIDs, task.Text, task.ImageLink)
		log.Printf("Scheduler: job done (cron=%q) sent=%d failed=%d", task.Cron, result.Sent, result.Failed)
		if result.Failed > 0 {
			log.Printf("Scheduler: job failures (cron=%q): %v", task.Cron, result.Errors)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

// Start begins running scheduled jobs in the background.
func (s *Scheduler) Start() {
	s.cron.Start()
}

// Stop halts the scheduler; running jobs are allowed to finish.
func (s *Scheduler) Stop() {
	<-s.cron.Stop().Done()
}
