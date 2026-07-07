package job

import (
	"encoding/json"
	"fmt"
	"os"
)

// Task is a single scheduled announcement, as read from the jobs JSON file:
//
//	[
//	  {
//	    "text": "text",
//	    "user_ids": [id, id],
//	    "cron": "*/10 * * * *"
//	  }
//	]
type Task struct {
	Text      string  `json:"text"`
	ImageLink string  `json:"image_link,omitempty"`
	UserIDs   []int64 `json:"user_ids"`
	Cron      string  `json:"cron"`
}

// LoadTasks reads and validates the jobs file. It fails fast on malformed
// JSON or an obviously broken task (missing text/cron/user_ids) rather than
// scheduling a job that can never fire correctly.
func LoadTasks(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read jobs file %s: %w", path, err)
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse jobs file %s: %w", path, err)
	}

	for i, t := range tasks {
		if t.Text == "" {
			return nil, fmt.Errorf("task %d: text is required", i)
		}
		if t.Cron == "" {
			return nil, fmt.Errorf("task %d: cron is required", i)
		}
		if len(t.UserIDs) == 0 {
			return nil, fmt.Errorf("task %d: user_ids cannot be empty", i)
		}
	}

	return tasks, nil
}
