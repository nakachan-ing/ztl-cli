package store

import (
	"fmt"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadLinks(config model.Config) ([]model.Link, error) {
	noteDir := config.ZettelDir
	var links []model.Link
	err := LoadJson(noteDir, &links)
	if err != nil {
		return nil, fmt.Errorf("❌ Error loading notes from JSON: %v", err)
	}
	return links, nil
}
