/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize config.yaml",
	Run: func(cmd *cobra.Command, args []string) {

		configPath, err := store.GetConfigPath()
		if err != nil {
			log.Printf("failed to get config path: %v", err)
		}

		configDir := filepath.Dir(configPath)

		configFile := filepath.Join(configDir, "config.yaml")

		// `~/.config/ztl/` を作成
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Fatalf("❌ Failed to create config directory: %v", err)
		}

		// デフォルトの設定を YAML に変換
		configData, err := yaml.Marshal(model.DefaultConfig())
		if err != nil {
			log.Fatalf("❌ Failed to generate config: %v", err)
		}

		// `config.yaml` を作成
		if err := os.WriteFile(configFile, configData, 0644); err != nil {
			log.Fatalf("❌ Failed to create config file: %v", err)
		}

		fmt.Println("✅ zk initialized successfully!")
		fmt.Println("📄 Config file created at:", configFile)

	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
