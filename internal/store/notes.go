package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadNotes(config model.Config) ([]model.Note, string, error) {
	noteJsonPath := filepath.Join(config.JsonDataDir, "notes.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// notes.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(noteJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(noteJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create notes.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check notes.json: %w", err)
	}

	// JSON をロード
	var notes []model.Note
	if err := LoadJson(noteJsonPath, &notes); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading notes from JSON: %w", err)
	}

	return notes, noteJsonPath, nil
}
