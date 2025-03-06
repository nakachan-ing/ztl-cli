package cmd

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/util"
)

// SyncWithS3 - S3 との同期処理
func SyncWithS3(config model.Config, direction string) error {
	s3Client, err := util.NewS3Client(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize S3 client: %w", err)
	}

	if direction == "pull" {
		// S3 から `metadata.json` を取得
		remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata.json for notes: %w", err)
		}
		remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata.json for json: %w", err)
		}

		// ローカルの `metadata.json` をロード
		localMetadataNotes, _ := util.LoadMetadata(filepath.Join(config.ZettelDir, "metadata.json"))
		localMetadataJson, _ := util.LoadMetadata(filepath.Join(config.JsonDataDir, "metadata.json"))

		// `notes/` の変更を取得
		notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "s3")

		// `json/` の変更を取得
		jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "s3")

		// 変更があるファイルのみ取得
		fileList := append(notesDiff, jsonDiff...)
		if len(fileList) == 0 {
			log.Println("✅ No changes detected. Everything is up-to-date.")
			return nil
		}

		log.Println("🔄 Syncing files from S3...")
		err = util.SyncFilesToS3(config, "pull", fileList)
		if err != nil {
			return fmt.Errorf("❌ Sync failed: %w", err)
		}

		log.Println("✅ Sync completed successfully.")
		return nil

	} else if direction == "push" {
		// ローカルの `metadata.json` を生成
		localMetadataNotes, err := util.GenerateMetadata(config.ZettelDir)
		if err != nil {
			return fmt.Errorf("❌ Failed to generate metadata.json for notes: %w", err)
		}
		localMetadataJson, err := util.GenerateMetadata(config.JsonDataDir)
		if err != nil {
			return fmt.Errorf("❌ Failed to generate metadata.json for json: %w", err)
		}

		// `metadata.json` をローカルに保存
		err = util.SaveMetadata(filepath.Join(config.ZettelDir, "metadata.json"), localMetadataNotes)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata.json for notes: %w", err)
		}
		err = util.SaveMetadata(filepath.Join(config.JsonDataDir, "metadata.json"), localMetadataJson)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata.json for json: %w", err)
		}

		// S3 から `metadata.json` を取得
		remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata.json for notes: %w", err)
		}
		remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata.json for json: %w", err)
		}

		// `notes/` の変更を取得
		notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "local")

		// `json/` の変更を取得
		jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "local")

		// 変更があるファイルのみアップロード
		fileList := append(notesDiff, jsonDiff...)
		if len(fileList) == 0 {
			log.Println("✅ No changes detected. Everything is up-to-date.")
			return nil
		}

		log.Println("🔄 Uploading changed files to S3...")
		err = util.SyncFilesToS3(config, "push", fileList)
		if err != nil {
			return fmt.Errorf("❌ Sync failed: %w", err)
		}

		// `metadata.json` を S3 にアップロード
		err = util.UploadMetadataToS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to upload metadata.json for notes: %w", err)
		}
		err = util.UploadMetadataToS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to upload metadata.json for json: %w", err)
		}

		log.Println("✅ Sync completed successfully.")
		return nil
	}

	return fmt.Errorf("❌ Unknown sync direction: %s", direction)
}

// ShowSyncStatus - S3 との同期状態を表示
func ShowSyncStatus(config model.Config) error {
	s3Client, err := util.NewS3Client(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize S3 client: %w", err)
	}

	localMetadataNotes, _ := util.LoadMetadata(filepath.Join(config.ZettelDir, "metadata.json"))
	localMetadataJson, _ := util.LoadMetadata(filepath.Join(config.JsonDataDir, "metadata.json"))

	remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, config, "notes")
	if err != nil {
		return err
	}
	remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, config, "json")
	if err != nil {
		return err
	}

	notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "s3")
	jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "s3")

	log.Println("📌 Files to be updated from S3:")
	for _, file := range append(notesDiff, jsonDiff...) {
		log.Println("   -", file)
	}

	return nil
}
