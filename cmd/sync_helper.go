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
		log.Println("🔄 Downloading metadata from S3...")

		// **S3 から `metadata.json` を取得**
		remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata_notes.json from S3: %w", err)
		}
		remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata_json.json from S3: %w", err)
		}

		// **ローカルの `metadata.json` をロード**
		localMetadataNotes, _ := util.LoadMetadata(filepath.Join(config.ZettelDir, "metadata_notes.json"))
		localMetadataJson, _ := util.LoadMetadata(filepath.Join(config.JsonDataDir, "metadata_json.json"))

		// **差分を取得**
		notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "s3")
		jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "s3")

		// **変更があるファイルのみダウンロード**
		fileList := append(notesDiff, jsonDiff...)
		if len(fileList) == 0 {
			log.Println("✅ No changes detected. Everything is up-to-date.")
		} else {
			log.Println("🔄 Downloading changed files from S3...")
			err = util.SyncFilesToS3(config, "pull", fileList)
			if err != nil {
				return fmt.Errorf("❌ Sync failed: %w", err)
			}
		}

		// **ローカルの `metadata.json` を更新**
		log.Println("🔄 Saving updated metadata...")
		err = util.SaveMetadata(filepath.Join(config.ZettelDir, "metadata_notes.json"), remoteMetadataNotes)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata_notes.json: %w", err)
		}
		err = util.SaveMetadata(filepath.Join(config.JsonDataDir, "metadata_json.json"), remoteMetadataJson)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata_json.json: %w", err)
		}

		log.Println("✅ Sync completed successfully.")
		return nil

	} else if direction == "push" {
		log.Println("🔄 Generating metadata for push...")

		// **ローカルの `metadata.json` を生成**
		localMetadataNotes, err := util.GenerateMetadata(config.ZettelDir)
		if err != nil {
			return fmt.Errorf("❌ Failed to generate metadata_notes.json: %w", err)
		}
		localMetadataJson, err := util.GenerateMetadata(config.JsonDataDir)
		if err != nil {
			return fmt.Errorf("❌ Failed to generate metadata_json.json: %w", err)
		}

		// **`metadata.json` をローカルに保存**
		err = util.SaveMetadata(filepath.Join(config.ZettelDir, "metadata_notes.json"), localMetadataNotes)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata_notes.json: %w", err)
		}
		err = util.SaveMetadata(filepath.Join(config.JsonDataDir, "metadata_json.json"), localMetadataJson)
		if err != nil {
			return fmt.Errorf("❌ Failed to save metadata_json.json: %w", err)
		}

		// // **S3 から `metadata.json` を取得**
		remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata_notes.json from S3: %w", err)
		}
		remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to download metadata_json.json from S3: %w", err)
		}

		// // **差分を取得**
		notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "local")
		jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "local")

		// // **変更があるファイルのみアップロード**
		fileList := append(notesDiff, jsonDiff...)
		if len(fileList) == 0 {
			log.Println("✅ No changes detected. Everything is up-to-date.")
		} else {
			log.Println("🔄 Uploading changed files to S3...")
			err = util.SyncFilesToS3(config, "push", fileList)
			if err != nil {
				return fmt.Errorf("❌ Sync failed: %w", err)
			}
		}

		// // **`metadata.json` を S3 にアップロード**
		log.Println("🔄 Uploading metadata to S3...")
		err = util.UploadMetadataToS3(s3Client, config, "notes")
		if err != nil {
			return fmt.Errorf("❌ Failed to upload metadata_notes.json: %w", err)
		}
		err = util.UploadMetadataToS3(s3Client, config, "json")
		if err != nil {
			return fmt.Errorf("❌ Failed to upload metadata_json.json: %w", err)
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
