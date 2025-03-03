package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"gopkg.in/yaml.v3"
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

// func UpdateNoteToJson(noteID string, config model.Config) error {
// 	notes, _, err := LoadNotes(config)
// 	if err != nil {
// 		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
// 	}

// 	for i := range notes {
// 		if noteID == notes[i].SeqID {
// 			updatedContent, err := os.ReadFile(filepath.Join(config.ZettelDir, notes[i].ID+".md"))
// 			if err != nil {
// 				log.Printf("❌ Failed to read updated note file: %v", err)
// 				os.Exit(1)
// 			}

// 			// Parse front matter
// 			frontMatter, _, err := ParseFrontMatter(string(updatedContent))
// 			if err != nil {
// 				log.Printf("❌ Error parsing front matter: %v", err)
// 				os.Exit(1)
// 			}

// 			// Update note metadata
// 			notes[i].Title = frontMatter.Title
// 			notes[i].NoteType = frontMatter.NoteType
// 			// notes[i].Tags = frontMatter.Tags
// 			// notes[i].Links = frontMatter.Links
// 			// notes[i].TaskStatus = frontMatter.TaskStatus
// 			notes[i].UpdatedAt = frontMatter.UpdatedAt
// 		}
// 	}
// 	return nil
// }

func ParseFrontMatter(content string) (model.NoteFrontMatter, string, error) {
	if !strings.HasPrefix(content, "---") {
		return model.NoteFrontMatter{}, content, fmt.Errorf("❌ Front matter not found")
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return model.NoteFrontMatter{}, content, fmt.Errorf("❌ Invalid front matter format")
	}

	frontMatterStr := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	// Parse YAML
	var frontMatter model.NoteFrontMatter
	err := yaml.Unmarshal([]byte(frontMatterStr), &frontMatter)
	if err != nil {
		return model.NoteFrontMatter{}, content, fmt.Errorf("❌ Failed to parse front matter: %w", err)
	}

	return frontMatter, body, nil
}

func UpdateFrontMatter(frontMatter *model.NoteFrontMatter, body string) string {
	// Convert to YAML
	frontMatterBytes, err := yaml.Marshal(frontMatter)
	if err != nil {
		log.Printf("❌ Failed to convert front matter to YAML: %v", err)
		return body
	}

	// Preserve `---` and merge YAML with body
	return fmt.Sprintf("---\n%s---\n\n%s", string(frontMatterBytes), body)
}

func SaveUpdatedJson(notes []model.Note, jsonPath string) error {
	updatedJson, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(jsonPath, updatedJson, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Printf("✅ Successfully updated JSON file: %s", jsonPath)
	return nil
}
