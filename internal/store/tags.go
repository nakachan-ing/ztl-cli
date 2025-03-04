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

func LoadTags(config model.Config) ([]model.Tag, string, error) {
	tagJsonPath := filepath.Join(config.JsonDataDir, "tags.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// tags.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(tagJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(tagJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create tags.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check tags.json: %w", err)
	}

	// JSON をロード
	var tags []model.Tag
	if err := LoadJson(tagJsonPath, &tags); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading tags from JSON: %w", err)
	}

	return tags, tagJsonPath, nil
}

func CreateNewTag(tags []string, config model.Config) error {
	for i := range tags {
		tag := model.Tag{
			ID:   "",
			Name: tags[i],
		}
		err := InsertTagToJson(tag, config)
		if err != nil {
			return fmt.Errorf("failed to write to JSON file: %w", err)
		}
		fmt.Printf("✅ Tag [%v] has been created successfully.\n", tag)
	}
	return nil
}

func InsertTagToJson(tag model.Tag, config model.Config) error {
	tags, tagJsonPath, err := LoadTags(config)

	if err != nil {
		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
	}

	for _, existingTag := range tags {
		if tag.Name == existingTag.Name {
			log.Printf("⚠️  Skip: Tag '%s' already exists.", tag.Name)
			return nil
		}
	}

	newTagID := GetNextTagID(tags)
	tag.ID = newTagID

	tags = append(tags, tag)

	// Serialize JSON
	jsonBytes, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(tagJsonPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Println("✅ Successfully updated JSON file!")
	return nil

}

func GetNextTagID(tags []model.Tag) string {
	maxSeqID := 0
	re := regexp.MustCompile(`t(\d+)`) // "tXXX" の数字部分を抽出する正規表現

	// SeqID の最大値を探す
	for _, tag := range tags {
		match := re.FindStringSubmatch(tag.ID)
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
		return fmt.Sprintf("t%03d", newSeqID) // 3桁ゼロ埋め
	}
	return fmt.Sprintf("t%d", newSeqID) // 1000以上はゼロ埋めなし
}
