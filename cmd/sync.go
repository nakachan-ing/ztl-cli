/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize local files with S3",
}

var syncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Upload local changes to S3",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("🔄 Running `ztl sync push`...") // デバッグログ
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			return fmt.Errorf("❌ Error loading config: %w", err)
		}

		err = SyncWithS3(*config, "push")
		if err != nil {
			log.Printf("❌ Sync failed: %v", err)
			return fmt.Errorf("❌ Sync failed: %w", err)
		}

		log.Println("✅ `ztl sync push` completed successfully.")
		return nil
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download latest changes from S3",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Println("🔄 Running `ztl sync pull`...") // デバッグログ
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			return fmt.Errorf("❌ Error loading config: %w", err)
		}

		err = SyncWithS3(*config, "pull")
		if err != nil {
			log.Printf("❌ Sync failed: %v", err)
			return fmt.Errorf("❌ Sync failed: %w", err)
		}

		log.Println("✅ `ztl sync pull` completed successfully.")
		return nil
	},
}

var syncStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show differences between local and S3 files",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		return ShowSyncStatus(*config)
	},
}

func init() {
	syncCmd.AddCommand(syncPushCmd, syncPullCmd, syncStatusCmd)
	rootCmd.AddCommand(syncCmd)
}
