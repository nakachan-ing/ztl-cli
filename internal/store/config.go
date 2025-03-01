package store

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"gopkg.in/yaml.v3"
)

func GetConfigPath() (string, error) {
	// Check if the environment variable `ZK_CONFIG` is set
	if customConfig := os.Getenv("ZTL_CONFIG"); customConfig != "" {
		return customConfig, nil
	}

	var configPath string

	switch runtime.GOOS {
	case "windows":
		// Use `APPDATA\ztl-cli\config.yaml` if available
		appData := os.Getenv("APPDATA")
		if appData != "" {
			configPath = filepath.Join(appData, "ztl", "config.yaml")
		} else {
			// Fallback to `USERPROFILE` if `APPDATA` is unavailable
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to determine home directory: %w", err)
			}
			configPath = filepath.Join(homeDir, "AppData", "Roaming", "ztl", "config.yaml")
		}

	default: // macOS / Linux
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("failed to determine home directory: %w", homeErr)
		}
		configPath = filepath.Join(homeDir, ".config", "ztl", "config.yaml")
		log.Printf("⚠️ Failed to get user config directory, using fallback: %s", configPath)
	}

	return configPath, nil
}

// Expand `~` to the home directory (Windows included)
func expandHomeDir(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("⚠️ Failed to get home directory: %v", err)
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func LoadConfig() (*model.Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file (%s): %w", configPath, err)
	}

	var config model.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Expand `~` in paths
	config.ZettelDir = expandHomeDir(config.ZettelDir)
	config.Backup.BackupDir = expandHomeDir(config.Backup.BackupDir)
	config.JsonDataDir = expandHomeDir(config.JsonDataDir)
	config.ArchiveDir = expandHomeDir(config.ArchiveDir)
	config.Trash.TrashDir = expandHomeDir(config.Trash.TrashDir)

	return &config, nil
}

func SaveConfig(config model.Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("❌ failed to get config path: %w", err)
	}
	configDir := filepath.Dir(configPath)

	// 設定ディレクトリを作成
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("❌ Failed to create config directory: %v", err)
	}

	// 設定をYAMLに変換
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("❌ Error marshaling config: %v", err)
	}

	// `config.yaml` に保存
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("❌ Failed to write config file: %v", err)
	}

	return nil
}
