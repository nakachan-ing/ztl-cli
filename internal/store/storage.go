package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadJson[T any](filePath string, v *[]T) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// ファイルが存在しない場合は空のスライスを返す
		*v = []T{}
		return nil
	} else if err != nil {
		return fmt.Errorf("❌ Failed to check JSON file: %w", err)
	}

	jsonBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("❌ Failed to read JSON file: %w", err)
	}

	if len(jsonBytes) > 0 {
		err = json.Unmarshal(jsonBytes, v)
		if err != nil {
			return fmt.Errorf("❌ Failed to parse JSON: %w", err)
		}
	}

	return nil
}

// Insert a new note into the JSON file
func InsertNoteToJson(note model.Note, config model.Config) error {

	notes, noteJsonPath, err := LoadNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
	}
	// Assign a new ID (incremental)
	newID := 1
	if len(notes) > 0 {
		newID = len(notes) + 1
	}
	note.SeqID = strconv.Itoa(newID)

	notes = append(notes, note)

	// Serialize JSON
	jsonBytes, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(noteJsonPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Println("✅ Successfully updated JSON file!")
	return nil
}
