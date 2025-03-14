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

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/text"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/nakachan-ing/ztl-cli/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var structureTags []string
var structureFrom string
var structureTo string
var structureSearchQuery string
var structurePageSize int
var structureTrash bool
var structureArchive bool
var structureForceDelete bool
var structureRestoreTrash bool
var structureRestoreArchive bool

func createNewStructureNote(structureTitle string, config model.Config) (string, model.Note, error) {
	t := time.Now()
	noteId := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	createdAt := t.Format("2006-01-02 15:04:05")

	// Create front matter
	frontMatter := model.NoteFrontMatter{
		ID:        noteId,
		Title:     structureTitle,
		NoteType:  "structure",
		Tags:      structureTags,
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
		Title:     structureTitle,
		NoteType:  "structure",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	err = store.InsertNoteToJson(note, config)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	fmt.Printf("✅ Structure Note %s has been created successfully.\n", filePath)
	return filePath, note, nil
}

// structureCmd represents the structure command
var structureCmd = &cobra.Command{
	Use:   "structure",
	Short: "Manage structure notes",
	Args:  cobra.MaximumNArgs(1),
}

var newStructureCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "Add a new structure note",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {
		structureTitle := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

		if len(structureTags) > 0 {
			if err := store.CreateNewTag(structureTags, *config); err != nil {
				log.Printf("❌ Failed to create tag: %v\n", err)
				return
			}
		}

		newStructureStr, note, err := createNewStructureNote(structureTitle, *config)
		if err != nil {
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}

		for _, tagID := range permanentTags {
			if err := store.InsertNoteTag(note.ID, tagID, *config); err != nil {
				log.Printf("❌ Failed to insert note-tag relation: %v\n", err)
			}
		}

		log.Printf("Opening %q (Title: %q)...", newStructureStr, structureTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newStructureStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}

		noteID := strings.TrimSuffix(filepath.Base(newStructureStr), ".md")

		mdContent, err := os.ReadFile(newStructureStr)
		if err != nil {
			log.Printf("❌ Failed to read Markdown file: %v", err)
		}

		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter for %s: %v", newStructureStr, err)
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

var structureListCmd = &cobra.Command{
	Use:     "list [title]",
	Short:   "List structure notes",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

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
			if structureTrash {
				if !note.Deleted {
					continue
				}
			} else if structureArchive {
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
				if note.NoteType != "structure" {
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

		if structureSearchQuery != "" {
			searchResults := util.FullTextSearch(filteredNotes, structureSearchQuery)
			if len(searchResults) > 0 {
				filteredNotes = searchResults
			}
		}

		// 検索結果がある場合のみフィルタリング
		if len(filteredNotes) > 0 {
			filteredNotes = util.FilterNotes(filteredNotes, structureTags, structureFrom, structureTo, noteTagDisplay)
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
		if structurePageSize == -1 {
			structurePageSize = len(filteredNotes)
		}

		// ページネーションのループ
		for {
			start := page * structurePageSize
			end := start + structurePageSize

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

			if structurePageSize == len(filteredNotes) {
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

var showStructureCmd = &cobra.Command{
	Use:     "show [Note ID]",
	Short:   "Show note detail",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

		// Load notes from JSON
		notes, _, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		var ID string
		found := false

		for _, note := range notes {
			if note.SeqID == noteID {
				ID = note.ID
				found = true
				break
			}
		}

		if !found {
			log.Printf("❌ Task with ID %s not found", noteID)
		}

		mdFilePath := filepath.Join(config.ZettelDir, ID+".md")

		if _, err := os.Stat(mdFilePath); os.IsNotExist(err) {
			log.Printf("❌ Markdown file not found: %s", mdFilePath)
		}

		// Markdown ファイルを読み込んで表示
		mdContent, err := os.ReadFile(mdFilePath)
		if err != nil {
			log.Printf("❌ Failed to read updated note file: %v", err)
		}

		titleStyle := color.New(color.FgCyan, color.Bold).SprintFunc()
		frontMatterStyle := color.New(color.FgHiGreen).SprintFunc()

		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("❌ Error parsing front matter: %v", err)
			os.Exit(1)
		}

		fmt.Printf("[%v] %v\n", titleStyle(frontMatter.ID), titleStyle(frontMatter.Title))
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Type: %v\n", frontMatterStyle(frontMatter.NoteType))
		fmt.Printf("Tags: %v\n", frontMatterStyle(frontMatter.Tags))
		fmt.Printf("Links: %v\n", frontMatterStyle(frontMatter.Links))
		fmt.Printf("Created at: %v\n", frontMatterStyle(frontMatter.CreatedAt))
		fmt.Printf("Updated at: %v\n", frontMatterStyle(frontMatter.UpdatedAt))

		// Render Markdown content unless --meta flag is used
		if !taskMeta {
			renderedContent, err := glamour.Render(body, "dark")
			if err != nil {
				log.Printf("⚠️ Failed to render markdown content: %v", err)
			} else {
				fmt.Println(renderedContent)
			}
		}

	},
}

var editStructureCmd = &cobra.Command{
	Use:     "edit [noteID]",
	Short:   "Edit a structure note",
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
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

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

				if err := store.BackupNote(filepath.Join(config.ZettelDir, notes[i].ID+".md"), config.Backup.BackupDir); err != nil {
					log.Printf("⚠️ Backup failed: %v", err)
				}

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

var deleteStructureCmd = &cobra.Command{
	Use:     "remove [noteID]",
	Short:   "Delete a structure note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"rm"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

		if literatureForceDelete {
			err = store.DeleteNotePermanently(noteID, *config)
		} else {
			err = store.MoveNoteToTrash(noteID, *config)
		}

		if err != nil {
			log.Fatalf("❌ %v", err)
		}
	},
}

var archiveStructureCmd = &cobra.Command{
	Use:     "archive [noteID]",
	Short:   "Archive a structure note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"mv"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		// Perform cleanup tasks
		if err := store.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Backup cleanup failed: %v", err)
		}
		if err := store.CleanupTrash(*config, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
			log.Printf("⚠️ Trash cleanup failed: %v", err)
		}

		// Load notes from JSON
		notes, notesJsonPath, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		found := false
		for i := range notes {
			if noteID == notes[i].SeqID {
				found = true

				originalPath := filepath.Join(config.ZettelDir, notes[i].ID+".md")
				archivedPath := filepath.Join(config.ArchiveDir, notes[i].ID+".md")

				note, err := os.ReadFile(originalPath)
				if err != nil {
					log.Printf("❌ Error reading note file: %v", err)
					return
				}

				// Parse front matter
				frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(note))
				if err != nil {
					log.Printf("❌ Error parsing front matter: %v", err)
					return
				}

				// Update `deleted:` field
				updatedFrontMatter := store.UpdateArchivedToFrontMatter(&frontMatter)
				updatedContent := store.UpdateFrontMatter(updatedFrontMatter, body)

				// Write back to file
				err = os.WriteFile(originalPath, []byte(updatedContent), 0644)
				if err != nil {
					log.Printf("❌ Error writing updated note file: %v", err)
					return
				}

				if _, err := os.Stat(config.ArchiveDir); os.IsNotExist(err) {
					err := os.MkdirAll(config.ArchiveDir, 0755)
					if err != nil {
						log.Printf("❌ Failed to create trash directory: %v", err)
						return
					}
				}

				err = os.Rename(originalPath, archivedPath)
				if err != nil {
					log.Printf("❌ Error moving note to trash: %v", err)
				}

				notes[i].Archived = true

				err = store.SaveUpdatedJson(notes, notesJsonPath)
				if err != nil {
					log.Printf("❌ Error updating JSON file: %v", err)
					return
				}

				log.Printf("✅ Note %s moved to trash: %s", notes[i].ID, archivedPath)
				break
			}
		}
		if !found {
			log.Printf("❌ Note with ID %s not found", noteID)
		}
	},
}

var restoreStructureCmd = &cobra.Command{
	Use:     "restore [noteID]",
	Short:   "Restore a structure note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"rs"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		if structureRestoreTrash && structureRestoreArchive {
			log.Fatalf("❌ You cannot specify both --deleted and --archived")
		}

		// デフォルトは `--deleted`
		if !structureRestoreTrash && !structureRestoreArchive {
			structureRestoreTrash = true
		}

		err = store.RestoreNote(noteID, *config, structureRestoreTrash, structureRestoreArchive)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

	},
}

func init() {
	structureCmd.AddCommand(newStructureCmd)
	structureCmd.AddCommand(structureListCmd)
	structureCmd.AddCommand(showStructureCmd)
	structureCmd.AddCommand(editStructureCmd)
	structureCmd.AddCommand(deleteStructureCmd)
	structureCmd.AddCommand(archiveStructureCmd)
	structureCmd.AddCommand(restoreStructureCmd)
	rootCmd.AddCommand(structureCmd)
	newStructureCmd.Flags().StringSliceVarP(&structureTags, "tag", "t", []string{}, "Specify tags")
	structureListCmd.Flags().StringSliceVarP(&structureTags, "tag", "t", []string{}, "Filter by tags")
	structureListCmd.Flags().StringVar(&structureFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	structureListCmd.Flags().StringVar(&structureTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	structureListCmd.Flags().StringVarP(&structureSearchQuery, "search", "q", "", "Search by title or content")
	structureListCmd.Flags().IntVar(&structurePageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	structureListCmd.Flags().BoolVar(&structureTrash, "trash", false, "Show deleted notes")
	structureListCmd.Flags().BoolVar(&structureArchive, "archive", false, "Show archived notes")
	deleteStructureCmd.Flags().BoolVarP(&structureForceDelete, "force", "f", false, "Permanently delete the note")
	restoreStructureCmd.Flags().BoolVar(&structureRestoreTrash, "trash", false, "Restore from trash")
	restoreStructureCmd.Flags().BoolVar(&structureRestoreArchive, "archive", false, "Restore from archive")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// structureCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// structureCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
