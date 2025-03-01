package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadNoteTags(config model.Config) ([]model.NoteTag, string, error) {
	noteTagsJsonPath := filepath.Join(config.JsonDataDir, "note_tags.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// tags.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(noteTagsJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(noteTagsJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create tags.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check tags.json: %w", err)
	}

	// JSON をロード
	var noteTags []model.NoteTag
	if err := LoadJson(noteTagsJsonPath, &noteTags); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading tags from JSON: %w", err)
	}

	return noteTags, noteTagsJsonPath, nil
}

// InsertNoteTag は note_tag.json に NoteTag を追加する
func InsertNoteTag(noteID string, tag string, config model.Config) error {
	noteTagsJsonPath := filepath.Join(config.JsonDataDir, "note_tags.json")

	// ファイルがなければ作成
	if _, err := os.Stat(noteTagsJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(noteTagsJsonPath, []byte("[]"), 0644); err != nil {
			return fmt.Errorf("❌ Failed to create note_tags.json: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("❌ Failed to check note_tags.json: %w", err)
	}

	tags, _, err := LoadTags(config)

	if err != nil {
		return fmt.Errorf("❌ Failed to load tags.json: %w", err)
	}

	// 既存のデータをロード
	noteTags, noteTagsJsonPath, err := LoadNoteTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	for _, t := range tags {
		if tag == t.Name {
			// 既に同じNoteIDとTagIDの組み合わせがあるかチェック
			for _, nt := range noteTags {
				if nt.NoteID == noteID && nt.TagID == t.ID {
					log.Printf("⚠️  Skip: Note-Tag pair (%s, %s) already exists.", noteID, t.ID)
					return nil
				}
			}
			// 新しい NoteTag を追加
			noteTags = append(noteTags, model.NoteTag{NoteID: noteID, TagID: t.ID})
		}
	}

	// JSON をシリアライズ
	jsonBytes, err := json.MarshalIndent(noteTags, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert note_tags.json to JSON: %w", err)
	}

	// JSON ファイルに書き込み
	if err := os.WriteFile(noteTagsJsonPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("❌ Failed to write note_tags.json: %w", err)
	}

	log.Println("✅ Successfully updated note_tags.json!")
	return nil
}
