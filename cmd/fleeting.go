/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/text"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/nakachan-ing/ztl-cli/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var fleetingTags []string
var fleetingFrom string
var fleetingTo string
var fleetingSearchQuery string
var fleetingPageSize int
var fleetingTrash bool
var fleetingArchive bool

func createNewFleetingNote(fleetingTitle string, config model.Config) (string, model.Note, error) {
	t := time.Now()
	noteId := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	createdAt := t.Format("2006-01-02 15:04:05")

	// Create front matter
	frontMatter := model.NoteFrontMatter{
		ID:        noteId,
		Title:     fleetingTitle,
		NoteType:  "fleeting",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	// Convert to YAML format
	frontMatterBytes, err := yaml.Marshal(frontMatter)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Create Markdown content
	content := fmt.Sprintf("---\n%s---\n\n## %s", string(frontMatterBytes), frontMatter.Title)

	// Write to file
	filePath := fmt.Sprintf("%s/%s.md", config.ZettelDir, noteId)
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to create note file (%s): %w", filePath, err)
	}

	// Write to JSON file
	note := model.Note{
		ID:        noteId,
		SeqID:     "",
		Title:     fleetingTitle,
		NoteType:  "fleeting",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	err = store.InsertNoteToJson(note, config, "fleeting")
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	fmt.Printf("✅ Fleeting Note %s has been created successfully.\n", filePath)
	return filePath, note, nil
}

// fleetingCmd represents the fleeting command
var fleetingCmd = &cobra.Command{
	Use:   "fleeting",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"f"},
}

var newFleetingCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "Add a new fleeting note",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {
		fleetingTitle := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		newFleetingStr, _, err := createNewFleetingNote(fleetingTitle, *config)
		if err != nil {
			log.Printf("❌ Failed to create task: %v\n", err)
			return
		}

		log.Printf("Opening %q (Title: %q)...", newFleetingStr, fleetingTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newFleetingStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}
	},
}

var fleetingListCmd = &cobra.Command{
	Use:     "list [title]",
	Short:   "List fleeting notes",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		// Load notes from JSON
		notes, _, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		filteredNotes := []model.Note{}

		for _, note := range notes {
			// Apply filters
			if fleetingTrash {
				if !note.Deleted {
					continue
				}
			} else if fleetingArchive {
				if !note.Archived {
					continue
				}
			} else {
				if note.Deleted {
					continue
				}
				if note.NoteType == "task" {
					continue
				}
				if note.NoteType != "fleeting" {
					continue
				}

			}

			filteredNotes = append(filteredNotes, note)

		}

		// Append filtered notes
		if fleetingSearchQuery != "" {
			filteredNotes = util.FullTextSearch(filteredNotes, fleetingSearchQuery)

		}
		filteredNotes = util.FilterNotes(filteredNotes, fleetingTags, fleetingFrom, fleetingTo)

		// Handle case where no notes match
		if len(filteredNotes) == 0 {
			fmt.Println("No matching notes found.")
			return
		}

		reader := bufio.NewReader(os.Stdin)
		page := 0

		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf("Zettelkasten: %v notes shown\n", len(filteredNotes))
		fmt.Println(strings.Repeat("=", 30))

		// `--limit` がない場合は全件表示
		if fleetingPageSize == -1 {
			fleetingPageSize = len(filteredNotes)
		}

		// ページネーションのループ
		for {
			start := page * fleetingPageSize
			end := start + fleetingPageSize

			// 範囲チェック
			if start >= len(filteredNotes) {
				fmt.Println("No more notes to display.")
				break
			}
			if end > len(filteredNotes) {
				end = len(filteredNotes)
			}

			// テーブル作成
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.SetStyle(table.StyleDouble)
			t.Style().Options.SeparateRows = false

			t.AppendHeader(table.Row{
				text.FgGreen.Sprintf("ID"), text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Title")),
				text.FgGreen.Sprintf("Type"),
				text.FgGreen.Sprintf("Created"), text.FgGreen.Sprintf("Updated"),
			})

			// フィルタされたノートをテーブルに追加
			for _, row := range filteredNotes[start:end] {
				noteType := row.NoteType
				typeColored := noteType

				switch noteType {
				case "permanent":
					typeColored = text.FgHiBlue.Sprintf("%s", noteType)
				case "literature":
					typeColored = text.FgHiYellow.Sprintf("%s", noteType)
				case "fleeting":
					typeColored = noteType
				case "index":
					typeColored = text.FgHiMagenta.Sprintf("%s", noteType)
				case "structure":
					typeColored = text.FgHiGreen.Sprintf("%s", noteType)
				}

				t.AppendRow(table.Row{
					row.SeqID,   // ノートのID
					row.Title,   // タイトル
					typeColored, // タイプ（色付き）
					// row.Tags,       // タグ
					row.CreatedAt, // 作成日時
					row.UpdatedAt, // 更新日時
					// len(row.Links), // リンクの数
				})
			}

			t.Render()

			if fleetingPageSize == len(filteredNotes) {
				break
			}

			if end >= len(filteredNotes) {
				break
			}

			fmt.Print("\nPress Enter for the next page (q to quit): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "q" {
				break
			}

			page++
		}
	},
}

func init() {
	fleetingCmd.AddCommand(newFleetingCmd)
	fleetingCmd.AddCommand(fleetingListCmd)
	rootCmd.AddCommand(fleetingCmd)
	fleetingListCmd.Flags().StringSliceVarP(&fleetingTags, "tag", "t", []string{}, "Filter by tags")
	fleetingListCmd.Flags().StringVar(&fleetingFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	fleetingListCmd.Flags().StringVar(&fleetingTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	fleetingListCmd.Flags().StringVarP(&fleetingSearchQuery, "search", "q", "", "Search by title or content")
	fleetingListCmd.Flags().IntVar(&fleetingPageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	fleetingListCmd.Flags().BoolVar(&fleetingTrash, "trash", false, "Show deleted notes")
	fleetingListCmd.Flags().BoolVar(&fleetingArchive, "archive", false, "Show archived notes")

}
