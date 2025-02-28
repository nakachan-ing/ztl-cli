package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadProjects(config model.Config) ([]model.Project, error) {
	noteDir := config.ZettelDir
	var projects []model.Project
	err := LoadJson(noteDir, &projects)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return projects, nil
}
