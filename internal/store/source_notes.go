package store

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func LoadSourceNotes(config model.Config) ([]model.SourceNote, string, error) {
	sourceNotesJsonPath := filepath.Join(config.JsonDataDir, "note_tags.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// tags.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(sourceNotesJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(sourceNotesJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create tags.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check tags.json: %w", err)
	}

	// JSON をロード
	var sourceNotes []model.SourceNote
	if err := LoadJson(sourceNotesJsonPath, &sourceNotes); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading tags from JSON: %w", err)
	}

	return sourceNotes, sourceNotesJsonPath, nil
}
