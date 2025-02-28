package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadTags(config model.Config) ([]model.Tag, error) {
	noteDir := config.ZettelDir
	var tags []model.Tag
	err := LoadJson(noteDir, &tags)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return tags, nil
}
