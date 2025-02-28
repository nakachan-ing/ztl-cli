package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadNotes(config model.Config) ([]model.Note, error) {
	noteDir := config.ZettelDir
	var notes []model.Note
	err := LoadJson(noteDir, &notes)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return notes, nil
}
