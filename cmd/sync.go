/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
)

// syncCmd は `ztl sync` コマンドを定義
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Zettelkasten notes and metadata with S3",
}

// pushCmd は `ztl sync push` の定義
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local notes and JSON metadata to S3",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		// AWSプロファイルを指定
		awsProfile := config.Sync.AWSProfile

		// S3 バケット名を取得
		s3Bucket := config.Sync.Bucket
		if s3Bucket == "" {
			log.Fatalf("❌ S3_BUCKET is not configured in config.yaml")
		}

		// ノートと JSON の同期を実行
		err = syncWithS3(config.ZettelDir, config.JsonDataDir, awsProfile, s3Bucket, "push")
		if err != nil {
			log.Fatalf("❌ Push failed: %v", err)
		}

		// メタデータを更新
		err = updateMetadata(*config)
		if err != nil {
			log.Fatalf("❌ Failed to update metadata.json: %v", err)
		}

		fmt.Println("✅ Push completed successfully!")
	},
}

// pullCmd は `ztl sync pull` の定義
var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download notes and JSON metadata from S3 to local",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		// AWSプロファイルを指定
		awsProfile := config.Sync.AWSProfile

		// S3 バケット名を取得
		s3Bucket := config.Sync.Bucket
		if s3Bucket == "" {
			log.Fatalf("❌ S3_BUCKET is not configured in config.yaml")
		}

		// ノートと JSON の同期を実行
		err = syncWithS3(config.ZettelDir, config.JsonDataDir, awsProfile, s3Bucket, "pull")
		if err != nil {
			log.Fatalf("❌ Pull failed: %v", err)
		}

		// メタデータを更新
		err = updateMetadata(*config)
		if err != nil {
			log.Fatalf("❌ Failed to update metadata.json: %v", err)
		}

		fmt.Println("✅ Pull completed successfully!")
	},
}

// syncWithS3 は AWS S3 との同期処理を実行（ノートと JSON）
func syncWithS3(localDir, jsonDir, awsProfile, s3Bucket, direction string) error {
	var cmdNotes, cmdJson *exec.Cmd

	if direction == "push" {
		fmt.Printf("🚀 Pushing %s and %s → s3://%s\n", localDir, jsonDir, s3Bucket)
		cmdNotes = exec.Command("aws", "--profile", awsProfile, "s3", "sync", localDir, "s3://"+s3Bucket+"/notes", "--delete")
		cmdJson = exec.Command("aws", "--profile", awsProfile, "s3", "sync", jsonDir, "s3://"+s3Bucket+"/json", "--delete")
	} else if direction == "pull" {
		fmt.Printf("⬇️ Pulling s3://%s → %s and %s\n", s3Bucket, localDir, jsonDir)
		cmdNotes = exec.Command("aws", "--profile", awsProfile, "s3", "sync", "s3://"+s3Bucket+"/notes", localDir, "--delete")
		cmdJson = exec.Command("aws", "--profile", awsProfile, "s3", "sync", "s3://"+s3Bucket+"/json", jsonDir, "--delete")
	} else {
		return fmt.Errorf("invalid sync direction: %s", direction)
	}

	// ノートの同期
	cmdNotes.Stdout = os.Stdout
	cmdNotes.Stderr = os.Stderr
	if err := cmdNotes.Run(); err != nil {
		return fmt.Errorf("AWS CLI sync for notes failed: %w", err)
	}

	// JSON の同期
	cmdJson.Stdout = os.Stdout
	cmdJson.Stderr = os.Stderr
	if err := cmdJson.Run(); err != nil {
		return fmt.Errorf("AWS CLI sync for JSON metadata failed: %w", err)
	}

	return nil
}

// `metadata_notes.json` と `metadata_json.json` を更新する処理
func updateMetadata(config model.Config) error {
	notesMetadataPath := filepath.Join(config.ZettelDir, "metadata_notes.json")
	jsonMetadataPath := filepath.Join(config.JsonDataDir, "metadata_json.json")

	// ノート用のメタデータ
	notesMetadata := make(map[string]string)
	err := updateFileMetadata(config.ZettelDir, notesMetadata)
	if err != nil {
		return fmt.Errorf("failed to read note directory: %w", err)
	}

	// JSON 用のメタデータ
	jsonMetadata := make(map[string]string)
	err = updateFileMetadata(config.JsonDataDir, jsonMetadata)
	if err != nil {
		return fmt.Errorf("failed to read JSON directory: %w", err)
	}

	// `metadata_notes.json` に保存
	err = saveMetadata(notesMetadataPath, notesMetadata)
	if err != nil {
		return fmt.Errorf("failed to save metadata_notes.json: %w", err)
	}

	// `metadata_json.json` に保存
	err = saveMetadata(jsonMetadataPath, jsonMetadata)
	if err != nil {
		return fmt.Errorf("failed to save metadata_json.json: %w", err)
	}

	return nil
}

// 指定ディレクトリ内のファイルの最終更新時刻を取得し、メタデータを更新
func updateFileMetadata(dirPath string, metadata map[string]string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue // ディレクトリはスキップ
		}

		filePath := filepath.Join(dirPath, file.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		metadata[file.Name()] = info.ModTime().Format(time.RFC3339)
	}

	return nil
}

// メタデータを JSON ファイルに保存
func saveMetadata(filePath string, metadata map[string]string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func init() {
	syncCmd.AddCommand(pushCmd, pullCmd)
	rootCmd.AddCommand(syncCmd)
}
