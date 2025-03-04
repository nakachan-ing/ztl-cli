package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
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

	newID := GetNextNoteID(notes)

	note.SeqID = newID

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

func GetNextNoteID(notes []model.Note) string {
	maxSeqID := 0
	re := regexp.MustCompile(`n(\d+)`) // "pXXX" の数字部分を抽出する正規表現

	// 最大IDを取得
	for _, note := range notes {
		match := re.FindStringSubmatch(note.SeqID)
		if match != nil {
			seq, err := strconv.Atoi(match[1]) // "XXX" 部分を整数に変換
			if err == nil && seq > maxSeqID {
				maxSeqID = seq
			}
		}
	}

	// 新しいIDを生成
	newSeqID := maxSeqID + 1

	// 999 までは3桁ゼロ埋め、それ以上はそのまま
	if newSeqID < 1000 {
		return fmt.Sprintf("n%03d", newSeqID) // 3桁ゼロ埋め
	}
	return fmt.Sprintf("n%d", newSeqID) // 1000以上はゼロ埋めなし
}

func ParseFrontMatter[T any](content string) (T, string, error) {
	var frontMatter T

	if !strings.HasPrefix(content, "---") {
		return frontMatter, content, fmt.Errorf("❌ Front matter not found")
	}

	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return frontMatter, content, fmt.Errorf("❌ Invalid front matter format")
	}

	frontMatterStr := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	// Parse YAML
	err := yaml.Unmarshal([]byte(frontMatterStr), &frontMatter)
	if err != nil {
		return frontMatter, content, fmt.Errorf("❌ Failed to parse front matter: %w", err)
	}

	return frontMatter, body, nil
}

func UpdateFrontMatter[T any](frontMatter T, body string) string {
	// Convert to YAML
	frontMatterBytes, err := yaml.Marshal(frontMatter)
	if err != nil {
		log.Printf("❌ Failed to convert front matter to YAML: %v", err)
		return body
	}

	// Preserve `---` and merge YAML with body
	return fmt.Sprintf("---\n%s---\n\n%s", string(frontMatterBytes), body)
}

func SaveUpdatedJson[T any](v []T, jsonPath string) error {
	updatedJson, err := json.MarshalIndent(v, "", "  ")
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

func UpdateDeletedToFrontMatter[T model.Deletable](frontMatter T) T {
	frontMatter.SetDeleted()
	return frontMatter
}

func UpdateArchivedToFrontMatter[T model.Archivable](frontMatter T) T {
	frontMatter.SetArchived()
	return frontMatter
}

func MoveNoteToTrash(noteID string, config model.Config) error {
	// Load notes from JSON
	notes, notesJsonPath, err := LoadNotes(config)
	if err != nil {
		log.Printf("❌ Error loading notes from JSON: %v", err)
		os.Exit(1)
	}

	found := false
	for i := range notes {
		if noteID == notes[i].SeqID {
			found = true

			originalPath := filepath.Join(config.ZettelDir, notes[i].ID+".md")
			deletedPath := filepath.Join(config.Trash.TrashDir, notes[i].ID+".md")

			note, err := os.ReadFile(originalPath)
			if err != nil {
				return fmt.Errorf("❌ Error reading note file: %v", err)
			}

			// Parse front matter
			frontMatter, body, err := ParseFrontMatter[model.NoteFrontMatter](string(note))
			if err != nil {
				return fmt.Errorf("❌ Error parsing front matter: %v", err)
			}

			// Update `deleted:` field
			updatedFrontMatter := UpdateDeletedToFrontMatter(&frontMatter)
			updatedContent := UpdateFrontMatter(updatedFrontMatter, body)

			// Write back to file
			err = os.WriteFile(originalPath, []byte(updatedContent), 0644)
			if err != nil {
				return fmt.Errorf("❌ Error writing updated note file: %v", err)
			}

			if _, err := os.Stat(config.Trash.TrashDir); os.IsNotExist(err) {
				err := os.MkdirAll(config.Trash.TrashDir, 0755)
				if err != nil {
					return fmt.Errorf("❌ Failed to create trash directory: %v", err)
				}
			}

			err = os.Rename(originalPath, deletedPath)
			if err != nil {
				return fmt.Errorf("❌ Error moving note to trash: %v", err)
			}

			notes[i].Deleted = true

			err = SaveUpdatedJson(notes, notesJsonPath)
			if err != nil {
				return fmt.Errorf("❌ Error updating JSON file: %v", err)
			}

			log.Printf("✅ Note %s moved to trash: %s", notes[i].ID, deletedPath)
			break
		}
	}
	if !found {
		log.Printf("❌ Note with ID %s not found", noteID)
	}
	return nil
}

func DeleteNotePermanently(noteID string, config model.Config) error {
	notes, notesJsonPath, err := LoadNotes(config)
	if err != nil {
		log.Printf("❌ Error loading notes from JSON: %v", err)
		os.Exit(1)
	}

	updatedNotes := []model.Note{}
	for i := range notes {
		if noteID != notes[i].SeqID {
			updatedNotes = append(updatedNotes, notes[i])
		} else {
			originalPath := filepath.Join(config.ZettelDir, notes[i].ID+".md")
			err := os.Remove(originalPath)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("❌ Failed to delete note file: %w", err)
			}
		}
	}
	err = SaveUpdatedJson(updatedNotes, notesJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update notes.json: %w", err)
	}

	// `note_tags.json` から該当ノートのタグ情報を削除
	noteTags, noteTagsJsonPath, err := LoadNoteTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	updatedNoteTags := []model.NoteTag{}
	for _, noteTag := range noteTags {
		if noteTag.NoteID != noteID {
			updatedNoteTags = append(updatedNoteTags, noteTag)
		}
	}

	err = SaveUpdatedJson(updatedNoteTags, noteTagsJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update note_tags.json: %w", err)
	}

	fmt.Printf("✅ Note %s permanently deleted\n", noteID)

	return nil
}
