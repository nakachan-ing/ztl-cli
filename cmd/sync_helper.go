package cmd

import (
	"context"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/util"
)

// SyncWithS3 - S3 との同期処理
func SyncWithS3(ztlConfig *model.Config, direction string) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(ztlConfig.Sync.AWSProfile),
		config.WithRegion("ap-northeast-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	if direction == "pull" {
		// `metadata.json` を S3 から取得
		remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, *ztlConfig, "notes")
		if err != nil {
			return err
		}
		remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, *ztlConfig, "json")
		if err != nil {
			return err
		}

		// ローカルの `metadata.json` をロード
		localMetadataNotes, _ := util.LoadMetadata(filepath.Join(ztlConfig.ZettelDir, "metadata.json"))
		localMetadataJson, _ := util.LoadMetadata(filepath.Join(ztlConfig.JsonDataDir, "metadata.json"))

		// `notes/` ディレクトリの変更を取得
		notesDiff := util.DetectChanges(localMetadataNotes, remoteMetadataNotes, "s3")

		// `json/` ディレクトリの変更を取得
		jsonDiff := util.DetectChanges(localMetadataJson, remoteMetadataJson, "s3")

		// 変更があったファイルをダウンロード
		fileList := append(notesDiff, jsonDiff...)
		return util.SyncFilesToS3(*ztlConfig, "pull", fileList)
	}

	return nil

}

// ShowSyncStatus - S3 との同期状態を表示
func ShowSyncStatus(ztlConfig model.Config) error {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(ztlConfig.Sync.AWSProfile),
		config.WithRegion("ap-northeast-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	s3Client := s3.NewFromConfig(cfg)

	localMetadataNotes, _ := util.LoadMetadata(filepath.Join(ztlConfig.ZettelDir, "metadata.json"))
	localMetadataJson, _ := util.LoadMetadata(filepath.Join(ztlConfig.JsonDataDir, "metadata.json"))

	remoteMetadataNotes, err := util.DownloadMetadataFromS3(s3Client, ztlConfig, "notes")
	if err != nil {
		return err
	}
	remoteMetadataJson, err := util.DownloadMetadataFromS3(s3Client, ztlConfig, "json")
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
