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

func LoadSources(config model.Config) ([]model.Source, string, error) {
	sourcesJsonPath := filepath.Join(config.JsonDataDir, "sources.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// sources.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(sourcesJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(sourcesJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create notes.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check notes.json: %w", err)
	}

	// JSON をロード
	var sources []model.Source
	if err := LoadJson(sourcesJsonPath, &sources); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading notes from JSON: %w", err)
	}

	return sources, sourcesJsonPath, nil
}

func InsertSourceToJson(source model.Source, config model.Config) error {
	sources, sourcesJsonPath, err := LoadSources(config)

	if err != nil {
		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
	}

	for _, existingSource := range sources {
		if source.Title == existingSource.Title {
			log.Printf("⚠️  Skip: Tag '%s' already exists.", source.Title)
			return nil
		}
	}

	newSourceID := GetNextSourceID(sources)
	source.SourceID = newSourceID

	sources = append(sources, source)

	// Serialize JSON
	jsonBytes, err := json.MarshalIndent(sources, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(sourcesJsonPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Println("✅ Successfully updated JSON file!")
	return nil

}

func GetNextSourceID(sources []model.Source) string {
	maxSeqID := 0
	re := regexp.MustCompile(`s(\d+)`) // "tXXX" の数字部分を抽出する正規表現

	// SeqID の最大値を探す
	for _, source := range sources {
		match := re.FindStringSubmatch(source.SourceID)
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
		return fmt.Sprintf("s%03d", newSeqID) // 3桁ゼロ埋め
	}
	return fmt.Sprintf("s%d", newSeqID) // 1000以上はゼロ埋めなし
}
