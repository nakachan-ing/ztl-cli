package store

import (
	"encoding/json"
	"fmt"
	"os"
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
