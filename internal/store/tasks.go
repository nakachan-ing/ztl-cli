package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadTasks(config model.Config) ([]model.Task, string, error) {
	tasksJsonPath := filepath.Join(config.JsonDataDir, "tasks.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// tasks.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(tasksJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(tasksJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create tasks.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check tasks.json: %w", err)
	}

	// JSON をロード
	var tasks []model.Task
	if err := LoadJson(tasksJsonPath, &tasks); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading projects from JSON: %w", err)
	}

	return tasks, tasksJsonPath, nil
}

func InsertTaskToJson(task model.Task, config model.Config) error {
	tasks, tasksJsonPath, err := LoadTasks(config)

	if err != nil {
		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
	}

	for _, existingTask := range tasks {
		if task.ID == existingTask.ID {
			log.Printf("⚠️  Skip: Tag '%s' already exists.", task.ID)
			return nil
		}
	}

	newTaskID := GetNextTaskID(tasks)
	// Task.SeqID = newID
	task.ID = newTaskID

	tasks = append(tasks, task)

	// Serialize JSON
	jsonBytes, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(tasksJsonPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Println("✅ Successfully updated JSON file!")
	return nil

}

func GetNextTaskID(tasks []model.Task) string {
	maxSeqID := 0
	re := regexp.MustCompile(`task-(\d+)`) // "pXXX" の数字部分を抽出する正規表現

	// 最大IDを取得
	for _, task := range tasks {
		match := re.FindStringSubmatch(task.ID)
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
		return fmt.Sprintf("task-%03d", newSeqID) // 3桁ゼロ埋め
	}
	return fmt.Sprintf("task-%d", newSeqID) // 1000以上はゼロ埋めなし
}

func RestoreTask(taskID string, config model.Config, restoreDeleted bool, restoreArchived bool) error {
	// Load notes from JSON
	tasks, _, err := LoadTasks(config)
	if err != nil {
		log.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	var noteID string
	found := false

	for _, task := range tasks {
		if task.ID == taskID {
			noteID = task.NoteID
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Task with ID %s not found", taskID)
	}

	notes, notesJsonPath, err := LoadNotes(config)
	if err != nil {
		log.Printf("❌ Error loading notes from JSON: %v", err)
		os.Exit(1)
	}

	found = false
	for i := range notes {
		if noteID == notes[i].SeqID {
			found = true

			var sourceDir, action string

			if restoreDeleted {
				sourceDir = config.Trash.TrashDir
				notes[i].Deleted = false
				action = "trash"
			} else if restoreArchived {
				sourceDir = config.ArchiveDir
				notes[i].Archived = false
				action = "archive"
			} else {
				return fmt.Errorf("❌ No valid restore option specified")
			}

			deletedPath := filepath.Join(sourceDir, notes[i].ID+".md")
			restoredPath := filepath.Join(config.ZettelDir, notes[i].ID+".md")

			note, err := os.ReadFile(deletedPath)
			if err != nil {
				return fmt.Errorf("❌ Error reading note file: %v", err)
			}

			// Parse front matter
			frontMatter, body, err := ParseFrontMatter[model.TaskFrontMatter](string(note))
			if err != nil {
				return fmt.Errorf("❌ Error parsing front matter: %v", err)
			}

			// Update `deleted:` or `archived:` field
			updatedFrontMatter := UpdateTaskInFrontMatter(&frontMatter, restoreDeleted, restoreArchived)
			updatedContent := UpdateFrontMatter(updatedFrontMatter, body)

			// Write back to file
			err = os.WriteFile(deletedPath, []byte(updatedContent), 0644)
			if err != nil {
				return fmt.Errorf("❌ Error writing updated note file: %v", err)
			}

			err = os.Rename(deletedPath, restoredPath)
			if err != nil {
				return fmt.Errorf("❌ Error moving note to %s: %v", action, err)
			}

			err = SaveUpdatedJson(notes, notesJsonPath)
			if err != nil {
				return fmt.Errorf("❌ Error updating JSON file: %v", err)
			}

			log.Printf("✅ Note %s restored from %s to Zettelkasten: %s", notes[i].ID, action, restoredPath)
			break
		}
	}

	if !found {
		log.Printf("❌ Note with ID %s not found", noteID)
	}
	return nil
}

func UpdateTaskInFrontMatter(frontMatter *model.TaskFrontMatter, restoreDeleted bool, restoreArchived bool) *model.TaskFrontMatter {
	if restoreDeleted {
		frontMatter.Deleted = false
	}
	if restoreArchived {
		frontMatter.Archived = false
	}
	return frontMatter
}
