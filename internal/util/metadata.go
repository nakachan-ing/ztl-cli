package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nakachan-ing/ztl-cli/internal/model"
)

// GenerateMetadata - 指定ディレクトリのファイル一覧と更新日時を取得
func GenerateMetadata(dir string) (map[string]string, error) {
	metadata := make(map[string]string)

	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to scan directory: %w", err)
	}

	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			log.Printf("⚠️ Failed to get file info: %s (%v)", filePath, err)
			continue
		}

		metadata[filepath.Base(filePath)] = info.ModTime().Format(time.RFC3339)
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

	// ディレクトリタイプに応じてパスを設定
	switch dirType {
	case "notes":
		metadataPath = filepath.Join(config.ZettelDir, "metadata.json")
		s3Key = "notes/metadata.json"
	case "json":
		metadataPath = filepath.Join(config.JsonDataDir, "metadata.json")
		s3Key = "json/metadata.json"
	default:
		return fmt.Errorf("❌ Invalid directory type: %s", dirType)
	}

	// ファイルを開く
	file, err := os.Open(metadataPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to open %s: %w", metadataPath, err)
	}
	defer file.Close()

	// S3 にアップロード
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
		metadataPath = filepath.Join(config.ZettelDir, "metadata.json")
		s3Key = "notes/metadata.json"
	case "json":
		metadataPath = filepath.Join(config.JsonDataDir, "metadata.json")
		s3Key = "json/metadata.json"
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

func SyncFilesToS3(ztlConfig model.Config, direction string, fileList []string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(ztlConfig.Sync.AWSProfile),
		config.WithRegion("ap-northeast-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	bucket := ztlConfig.Sync.Bucket

	for _, file := range fileList {
		var localPath string

		// `notes/` の場合は `ZettelDir` を基準にする
		cleanFile := filepath.Clean(file)

		if strings.HasPrefix(cleanFile, "notes"+string(os.PathSeparator)) {
			localPath = filepath.Join(ztlConfig.ZettelDir, filepath.Base(cleanFile))
		} else if strings.HasPrefix(cleanFile, "json"+string(os.PathSeparator)) {
			localPath = filepath.Join(ztlConfig.JsonDataDir, filepath.Base(cleanFile))
		} else {
			log.Printf("⚠️ Unknown file category: %s", file)
			continue
		}

		s3Key := file // S3 側のキーもそのまま

		if direction == "pull" {
			err := DownloadFromS3(s3Client, bucket, s3Key, localPath)
			if err != nil {
				log.Printf("⚠️ Failed to download %s: %v", file, err)
			}
		} else if direction == "push" {
			err := UploadToS3(s3Client, bucket, localPath, s3Key)
			if err != nil {
				log.Printf("⚠️ Failed to upload %s: %v", file, err)
			}
		}
	}

	return nil
}

func DetectChanges(localMeta, remoteMeta map[string]string, source string) []string {
	var filesToSync []string

	// nil チェックを追加（panic 回避）
	if localMeta == nil {
		localMeta = make(map[string]string)
	}
	if remoteMeta == nil {
		remoteMeta = make(map[string]string)
	}

	if source == "s3" {
		// S3 の方が新しい場合、ダウンロード対象
		for file, remoteTime := range remoteMeta {
			localTime, exists := localMeta[file]
			if !exists || localTime < remoteTime {
				filesToSync = append(filesToSync, file)
			}
		}
	} else if source == "local" {
		// ローカルの方が新しい場合、アップロード対象
		for file, localTime := range localMeta {
			remoteTime, exists := remoteMeta[file]
			if !exists || remoteTime < localTime {
				filesToSync = append(filesToSync, file)
			}
		}
	} else {
		log.Printf("⚠️ Unknown source type: %s", source)
	}

	return filesToSync
}
