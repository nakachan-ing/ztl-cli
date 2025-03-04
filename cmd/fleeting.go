/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

	err = store.InsertNoteToJson(note, config)
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
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}

		log.Printf("Opening %q (Title: %q)...", newFleetingStr, fleetingTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newFleetingStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}

		noteID := strings.TrimSuffix(filepath.Base(newFleetingStr), ".md")

		mdContent, err := os.ReadFile(newFleetingStr)
		if err != nil {
			log.Printf("❌ Failed to read Markdown file: %v", err)
		}

		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter for %s: %v", newFleetingStr, err)
			body = string(mdContent) // フロントマターの解析に失敗した場合、全文をセット
		}

		notes, noteJsonPath, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		found := false
		for i, note := range notes {
			if note.ID == noteID {
				notes[i].Title = frontMatter.Title
				notes[i].NoteType = frontMatter.NoteType
				notes[i].Content = body
				notes[i].UpdatedAt = time.Now().Format("2006-01-02 15:04:05") // 更新日時も更新
				found = true
				break
			}
		}

		if !found {
			log.Printf("❌ Note with ID %s not found", noteID)
		}

		err = store.SaveUpdatedJson(notes, noteJsonPath)
		if err != nil {
			log.Printf("❌ Failed to update notes.json: %v\n", err)
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

		noteTags, _, err := store.LoadNoteTags(*config)
		if err != nil {
			log.Printf("❌ Error loading note-tag relationships: %v", err)
			os.Exit(1)
		}

		tags, _, err := store.LoadTags(*config)
		if err != nil {
			log.Printf("❌ Error loading tags from JSON: %v", err)
			os.Exit(1)
		}

		// ノートIDごとにタグ名をマッピングするためのマップを作成
		noteTagMap := make(map[string][]string)

		// ノートIDとタグIDをマッピング
		for _, noteTag := range noteTags {
			noteTagMap[noteTag.NoteID] = append(noteTagMap[noteTag.NoteID], noteTag.TagID)
		}

		// タグIDからタグ名へのマッピングを作成
		tagMap := make(map[string]string)
		for _, tag := range tags {
			tagMap[tag.ID] = tag.Name
		}

		filteredNotes := []model.Note{}
		noteTagDisplay := make(map[string][]string)

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

			for _, note := range notes {
				// ループの中で新しいスライスを作成（前回の値をクリア）
				var tagNames []string

				// ノートに紐づくタグIDを取得
				for _, tagID := range noteTagMap[note.ID] {
					if tagName, exists := tagMap[tagID]; exists {
						tagNames = append(tagNames, tagName)
					}
				}

				// ノートIDとタグのマッピングを更新
				noteTagDisplay[note.ID] = tagNames
			}

			filteredNotes = append(filteredNotes, note)
		}

		if fleetingSearchQuery != "" {
			searchResults := util.FullTextSearch(filteredNotes, fleetingSearchQuery)
			if len(searchResults) > 0 {
				filteredNotes = searchResults
			}
		}

		// 検索結果がある場合のみフィルタリング
		if len(filteredNotes) > 0 {
			filteredNotes = util.FilterNotes(filteredNotes, fleetingTags, fleetingFrom, fleetingTo, noteTagDisplay)
		}

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
				text.FgGreen.Sprintf("Tags"),
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

				tagNames := noteTagDisplay[row.ID]
				tagStr := strings.Join(tagNames, ", ")

				t.AppendRow(table.Row{
					row.SeqID,     // ノートのID
					row.Title,     // タイトル
					typeColored,   // タイプ（色付き）
					tagStr,        // タグ
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

var editFleetingCmd = &cobra.Command{
	Use:     "edit [noteID]",
	Short:   "Edit a fleeting note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"e"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
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
		notes, notesJsonPath, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		found := false
		for i := range notes {
			if noteID == notes[i].SeqID {

				lockFile := filepath.Join(config.ZettelDir, notes[i].ID+".lock")
				if err := util.CreateLockFile(lockFile); err != nil {
					log.Printf("❌ Failed to create lock file: %v", err)
					os.Exit(1)
				}

				// if err := backupNote(zettels[i].NotePath, config.Backup.BackupDir); err != nil {
				// 	log.Printf("⚠️ Backup failed: %v", err)
				// }

				fmt.Printf("Found %v, opening...\n", filepath.Join(config.ZettelDir, notes[i].ID+".md"))
				time.Sleep(2 * time.Second)

				c := exec.Command(config.Editor, filepath.Join(config.ZettelDir, notes[i].ID+".md"))
				defer os.Remove(lockFile) // Ensure lock file is deleted after editing
				c.Stdin = os.Stdin
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					log.Printf("❌ Failed to open editor: %v", err)
					os.Exit(1)
				}

				mdContent, err := os.ReadFile(filepath.Join(config.ZettelDir, notes[i].ID+".md"))
				if err != nil {
					log.Printf("❌ Failed to read updated note file: %v", err)
					os.Exit(1)
				}

				// Parse front matter
				frontMatter, body, err := store.ParseFrontMatter[model.TaskFrontMatter](string(mdContent))
				if err != nil {
					log.Printf("❌ Error parsing front matter: %v", err)
					os.Exit(1)
				}

				frontMatter.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

				notes[i].UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

				updatedContent := store.UpdateFrontMatter(&frontMatter, body)
				err = os.WriteFile(filepath.Join(config.ZettelDir, notes[i].ID+".md"), []byte(updatedContent), 0644)
				if err != nil {
					log.Printf("❌ Error writing updated note file: %v", err)
				}

				notes[i].Title = frontMatter.Title
				// notes[i].Links = frontMatter.Links
				notes[i].UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
				notes[i].Content = body

				updatedJson, err := json.MarshalIndent(notes, "", "  ")
				if err != nil {
					log.Printf("❌ Failed to convert updated notes to JSON: %v", err)
					os.Exit(1)
				}

				// Write back to `zettel.json`
				if err := os.WriteFile(notesJsonPath, updatedJson, 0644); err != nil {
					log.Printf("❌ Failed to write updated notes to JSON file: %v", err)
					os.Exit(1)
				}

				fmt.Println("✅ Note metadata updated successfully:", notesJsonPath)

				found = true
				break
			}
		}
		if !found {
			log.Printf("❌ Note with ID %s not found", noteID)
		}
	},
}

func init() {
	fleetingCmd.AddCommand(newFleetingCmd)
	fleetingCmd.AddCommand(fleetingListCmd)
	fleetingCmd.AddCommand(editFleetingCmd)
	rootCmd.AddCommand(fleetingCmd)
	fleetingListCmd.Flags().StringSliceVarP(&fleetingTags, "tag", "t", []string{}, "Filter by tags")
	fleetingListCmd.Flags().StringVar(&fleetingFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	fleetingListCmd.Flags().StringVar(&fleetingTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	fleetingListCmd.Flags().StringVarP(&fleetingSearchQuery, "search", "q", "", "Search by title or content")
	fleetingListCmd.Flags().IntVar(&fleetingPageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	fleetingListCmd.Flags().BoolVar(&fleetingTrash, "trash", false, "Show deleted notes")
	fleetingListCmd.Flags().BoolVar(&fleetingArchive, "archive", false, "Show archived notes")

}
