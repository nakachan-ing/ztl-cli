package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadTasks(config model.Config) ([]model.Task, error) {
	noteDir := config.ZettelDir
	var tasks []model.Task
	err := LoadJson(noteDir, &tasks)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return tasks, nil
}
