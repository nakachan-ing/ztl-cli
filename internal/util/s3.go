package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func SyncFilesToS3(config model.Config, direction string, fileList []string) error {
	s3Client, err := NewS3Client(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to initialize S3 client: %w", err)
	}

	bucket := config.Sync.Bucket

	for _, file := range fileList {
		var s3Key string
		var localPath string

		// ファイルが `notes/` にある場合
		if strings.HasSuffix(file, ".md") {
			localPath = filepath.Join(config.ZettelDir, file)
			s3Key = "notes/" + file
		} else if strings.HasSuffix(file, ".json") {
			localPath = filepath.Join(config.JsonDataDir, file)
			s3Key = "json/" + file
		} else {
			log.Printf("⚠️ Unknown file category: %s", file)
			continue
		}

		if direction == "push" {
			err = UploadToS3(s3Client, bucket, localPath, s3Key)
			if err != nil {
				log.Printf("❌ Failed to upload %s: %v", file, err)
			} else {
				log.Printf("✅ Uploaded: %s", file)
			}
		}

		if direction == "pull" {
			err = DownloadFromS3(s3Client, bucket, s3Key, localPath)
			if err != nil {
				log.Printf("❌ Failed to download %s: %v", file, err)
			} else {
				log.Printf("✅ Downloaded: %s", file)
			}
		}
	}

	return nil
}

// UploadToS3 - ローカルファイルを S3 にアップロード
func UploadToS3(s3Client *s3.Client, bucket, filePath, s3Key string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("❌ Failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	_, err = s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("❌ Failed to upload %s to S3: %w", s3Key, err)
	}

	log.Printf("✅ Uploaded %s to S3", s3Key)
	return nil
}

// DownloadFromS3 - S3 からローカルにファイルをダウンロード
func DownloadFromS3(s3Client *s3.Client, bucket, s3Key, filePath string) error {
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("❌ Failed to download %s from S3: %w", s3Key, err)
	}
	defer resp.Body.Close()

	// ローカルに保存
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("❌ Failed to create file %s: %w", filePath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("❌ Failed to write file %s: %w", filePath, err)
	}

	log.Printf("✅ Downloaded %s from S3", s3Key)
	return nil
}

func isNotFoundErr(err error) bool {
	var s3Err *types.NoSuchKey
	return errors.As(err, &s3Err)
}

func NewS3Client(ztlConfig model.Config) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(ztlConfig.Sync.AWSProfile),
		config.WithRegion(ztlConfig.Sync.AWSRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	return s3Client, nil
}
