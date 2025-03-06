package util

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/nakachan-ing/ztl-cli/internal/model"
)

// UploadToS3 - ローカルファイルを S3 にアップロード
func UploadToS3(s3Client *s3.Client, bucket, filePath string, s3Key string) error {
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
func DownloadFromS3(s3Client *s3.Client, bucket, s3Key string, localPath string) error {
	resp, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(s3Key),
	})
	if err != nil {
		return fmt.Errorf("❌ Failed to download %s from S3: %w", s3Key, err)
	}
	defer resp.Body.Close()

	// 保存先ディレクトリが存在しない場合は作成
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, os.ModePerm); err != nil {
		return fmt.Errorf("❌ Failed to create directory %s: %w", localDir, err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to create file %s: %w", localPath, err)
	}
	defer file.Close()

	_, err = file.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("❌ Failed to write file %s: %w", localPath, err)
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
