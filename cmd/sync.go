/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
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
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		return SyncWithS3(config, "push")
	},
}

var syncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download latest changes from S3",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		return SyncWithS3(config, "pull")
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
