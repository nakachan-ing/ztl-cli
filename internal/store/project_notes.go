package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadProjectNotes(config model.Config) ([]model.ProjectNote, string, error) {
	projectNotesJsonPath := filepath.Join(config.JsonDataDir, "project_notes.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// project_notes.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(projectNotesJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(projectNotesJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create project_notes.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check project_notes.json: %w", err)
	}

	// JSON をロード
	var projectNotes []model.ProjectNote
	if err := LoadJson(projectNotesJsonPath, &projectNotes); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading projects & notes from JSON: %w", err)
	}

	return projectNotes, projectNotesJsonPath, nil
}

func InsertProjectNoteToJson(projectNote model.ProjectNote, config model.Config) error {
	projectNotesJsonPath := filepath.Join(config.JsonDataDir, "project_notes.json")

	// ファイルがなければ作成
	if _, err := os.Stat(projectNotesJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(projectNotesJsonPath, []byte("[]"), 0644); err != nil {
			return fmt.Errorf("❌ Failed to create project_notes.json: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("❌ Failed to check project_notes.json: %w", err)
	}

	// 既存のデータをロード
	projectNotes, projectNotesJsonPath, err := LoadProjectNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load project_notes.json: %w", err)
	}

	for _, pn := range projectNotes {
		if pn.NoteID == projectNote.NoteID && pn.ProjectID == projectNote.ProjectID {
			log.Printf("⚠️  Skip: Project-Note pair (%s, %s) already exists.", projectNote.ProjectID, projectNote.NoteID)
			return nil
		}
	}
	// 新しい NoteTag を追加
	projectNotes = append(projectNotes, model.ProjectNote{ProjectID: projectNote.ProjectID, NoteID: projectNote.NoteID})

	// JSON をシリアライズ
	jsonBytes, err := json.MarshalIndent(projectNotes, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert project_notes.json to JSON: %w", err)
	}

	// JSON ファイルに書き込み
	if err := os.WriteFile(projectNotesJsonPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("❌ Failed to write project_notes.json: %w", err)
	}

	log.Println("✅ Successfully updated project_notes.json!")
	return nil

}
