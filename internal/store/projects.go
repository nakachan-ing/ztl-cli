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

func LoadProjects(config model.Config) ([]model.Project, string, error) {
	projectsJsonPath := filepath.Join(config.JsonDataDir, "projects.json")

	// ディレクトリがない場合は作成
	if err := os.MkdirAll(config.JsonDataDir, 0755); err != nil {
		return nil, "", fmt.Errorf("❌ Failed to create json data directory: %w", err)
	}

	// projects.json が存在しない場合、空の JSON 配列 `[]` で初期化
	if _, err := os.Stat(projectsJsonPath); os.IsNotExist(err) {
		if err := os.WriteFile(projectsJsonPath, []byte("[]"), 0644); err != nil {
			return nil, "", fmt.Errorf("❌ Failed to create projects.json file: %w", err)
		}
	} else if err != nil {
		// ファイルの存在確認時の別のエラー（例: 権限エラー）
		return nil, "", fmt.Errorf("❌ Failed to check projects.json: %w", err)
	}

	// JSON をロード
	var projects []model.Project
	if err := LoadJson(projectsJsonPath, &projects); err != nil {
		return nil, "", fmt.Errorf("❌ Error loading projects from JSON: %w", err)
	}

	return projects, projectsJsonPath, nil
}

func InsertProjectToJson(project model.Project, config model.Config) error {
	projects, projectsJsonPath, err := LoadProjects(config)

	if err != nil {
		return fmt.Errorf("❌ Failed to load to JSON: %w", err)
	}

	for _, existingProject := range projects {
		if project.Name == existingProject.Name {
			log.Printf("⚠️  Skip: Tag '%s' already exists.", project.Name)
			return nil
		}
	}

	newProjectID := GetNextProjectID(projects)
	// project.SeqID = newID
	project.ProjectID = newProjectID

	projects = append(projects, project)

	// Serialize JSON
	jsonBytes, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to convert to JSON: %w", err)
	}

	err = os.WriteFile(projectsJsonPath, jsonBytes, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write JSON file: %w", err)
	}

	log.Println("✅ Successfully updated JSON file!")
	return nil

}

func GetNextProjectID(projects []model.Project) string {
	maxSeqID := 0
	re := regexp.MustCompile(`p(\d+)`) // "pXXX" の数字部分を抽出する正規表現

	// 最大IDを取得
	for _, project := range projects {
		match := re.FindStringSubmatch(project.ProjectID)
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
		return fmt.Sprintf("p%03d", newSeqID) // 3桁ゼロ埋め
	}
	return fmt.Sprintf("p%d", newSeqID) // 1000以上はゼロ埋めなし
}
