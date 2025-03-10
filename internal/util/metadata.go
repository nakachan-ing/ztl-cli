package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nakachan-ing/ztl-cli/internal/model"
)

// GenerateMetadata - 指定ディレクトリのファイル一覧と更新日時を取得
func GenerateMetadata(dir string) (map[string]string, error) {
	metadata := make(map[string]string)

	// **ディレクトリを再帰的に探索**
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("⚠️ Failed to access path: %s (%v)", path, err)
			return nil
		}

		// **ディレクトリはスキップ**
		if info.IsDir() {
			return nil
		}

		// **dir からの相対パスをキーにする**
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			log.Printf("⚠️ Failed to get relative path for: %s (%v)", path, err)
			return nil
		}

		// **ファイルの最終更新時刻を取得**
		metadata[relPath] = info.ModTime().Format(time.RFC3339)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("❌ Failed to scan directory: %w", err)
	}

	return metadata, nil
}

// SaveMetadata - metadata.json をローカルに保存
func SaveMetadata(metadataPath string, metadata map[string]string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("❌ Failed to marshal metadata.json: %w", err)
	}

	err = os.WriteFile(metadataPath, data, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to write metadata.json: %w", err)
	}

	log.Println("✅ metadata.json updated!")
	return nil
}

// LoadMetadata - metadata.json をロード
func LoadMetadata(metadataPath string) (map[string]string, error) {
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, fmt.Errorf("❌ Failed to read metadata.json: %w", err)
	}

	var metadata map[string]string
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to parse metadata.json: %w", err)
	}

	return metadata, nil
}

func UploadMetadataToS3(s3Client *s3.Client, config model.Config, dirType string) error {
	var metadataPath string
	var s3Key string

	// **ディレクトリタイプに応じてパスを設定**
	switch dirType {
	case "notes":
		metadataPath = filepath.Join(config.ZettelDir, "metadata_notes.json")
		s3Key = "notes/metadata_notes.json"
	case "json":
		metadataPath = filepath.Join(config.JsonDataDir, "metadata_json.json")
		s3Key = "json/metadata_json.json"
	default:
		return fmt.Errorf("❌ Invalid directory type: %s", dirType)
	}

	// **ファイルを開く**
	file, err := os.Open(metadataPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to open %s: %w", metadataPath, err)
	}
	defer file.Close()

	log.Printf("🔄 Uploading %s to S3...", s3Key)

	// **S3 にアップロード**
	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(config.Sync.Bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("❌ Failed to upload %s to S3: %w", s3Key, err)
	}

	log.Printf("✅ %s uploaded to S3!", s3Key)
	return nil
}

func DownloadMetadataFromS3(s3Client *s3.Client, config model.Config, dirType string) (map[string]string, error) {
	var metadataPath string
	var s3Key string

	// ディレクトリタイプに応じてパスを設定
	switch dirType {
	case "notes":
		metadataPath = filepath.Join(config.ZettelDir, "metadata_notes.json")
		s3Key = "notes/metadata_notes.json"
	case "json":
		metadataPath = filepath.Join(config.JsonDataDir, "metadata_json.json")
		s3Key = "json/metadata_json.json"
	default:
		return nil, fmt.Errorf("❌ Invalid directory type: %s", dirType)
	}

	// S3 から `metadata.json` を取得
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(config.Sync.Bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		if isNotFoundErr(err) {
			log.Printf("⚠️ No %s found on S3, returning empty metadata.", s3Key)
			return make(map[string]string), nil // S3 にない場合は空の `map` を返す
		}
		return nil, fmt.Errorf("❌ Failed to download %s from S3: %w", s3Key, err)
	}
	defer resp.Body.Close()

	// `metadata.json` をローカルに保存
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to read %s from S3: %w", s3Key, err)
	}

	err = os.WriteFile(metadataPath, data, 0644)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to save %s: %w", metadataPath, err)
	}

	log.Printf("✅ %s downloaded from S3!", s3Key)

	// `metadata.json` をロード
	metadata, err := LoadMetadata(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to load downloaded metadata: %w", err)
	}

	return metadata, nil
}

func DetectChanges(localMeta, remoteMeta map[string]string, source string) []string {
	var filesToSync []string

	// **ローカル vs S3 の比較**
	for file, remoteTimeStr := range remoteMeta {
		// **metadata.json は比較対象外**
		if file == "metadata.json" {
			continue
		}

		localTimeStr, exists := localMeta[file]

		// **ローカルに存在しないファイル (S3 にあるがローカルにない)**
		if !exists {
			log.Printf("📌 File missing locally, adding to sync (pull): %s", file)
			filesToSync = append(filesToSync, file)
			continue
		}

		// **タイムスタンプ比較**
		remoteTime, err := time.Parse(time.RFC3339, remoteTimeStr)
		if err != nil {
			log.Printf("⚠️ Failed to parse remote timestamp for %s: %v", file, err)
			continue
		}

		localTime, err := time.Parse(time.RFC3339, localTimeStr)
		if err != nil {
			log.Printf("⚠️ Failed to parse local timestamp for %s: %v", file, err)
			continue
		}

		// **S3 の方が新しければ pull**
		if source == "s3" && remoteTime.After(localTime.Add(1*time.Second)) {
			log.Printf("📌 Newer version on S3, adding to sync (pull): %s", file)
			filesToSync = append(filesToSync, file)
		}

		// **ローカルの方が新しければ push**
		if source == "local" && localTime.After(remoteTime.Add(1*time.Second)) {
			log.Printf("📌 Newer version locally, adding to sync (push): %s", file)
			filesToSync = append(filesToSync, file)
		}
	}

	// **ローカルにあるが S3 にないファイルを追加 (push の場合)**
	if source == "local" {
		for file := range localMeta {
			if _, exists := remoteMeta[file]; !exists {
				log.Printf("📌 File missing on S3, adding to sync (push): %s", file)
				filesToSync = append(filesToSync, file)
			}
		}
	}

	return filesToSync
}
