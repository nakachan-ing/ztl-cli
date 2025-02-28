package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadNoteTags(config model.Config) ([]model.NoteTag, error) {
	noteDir := config.ZettelDir
	var noteTags []model.NoteTag
	err := LoadJson(noteDir, &noteTags)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return noteTags, nil
}
