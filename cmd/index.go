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

var indexBookID string
var indexTags []string
var indexFrom string
var indexTo string
var indexSearchQuery string
var indexPageSize int
var indexTrash bool
var indexArchive bool

var indexForceDelete bool
var indexRestoreTrash bool
var indexRestoreArchive bool

func createNewIndexNote(indexTitle, sourceID string, config model.Config) (string, model.Note, error) {
	t := time.Now()
	noteId := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	createdAt := t.Format("2006-01-02 15:04:05")

	// Create front matter
	frontMatter := model.NoteFrontMatter{
		ID:        noteId,
		Title:     indexTitle,
		NoteType:  "index",
		Tags:      indexTags,
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
		Title:     indexTitle,
		NoteType:  "index",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	err = store.InsertNoteToJson(note, config)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	// `--book` を指定した場合、`source_notes.json` に索引ノートを紐づける
	if indexBookID != "" {
		err = linkIndexToSource(note.ID, sourceID, config)
		if err != nil {
			log.Fatalf("❌ Failed to link index to source: %v", err)
		}
	}

	log.Printf("✅ Index note '%s' created!", indexTitle)

	fmt.Printf("✅ Index Note %s has been created successfully.\n", filePath)
	return filePath, note, nil
}

func linkIndexToSource(noteID, sourceID string, config model.Config) error {
	// `source_notes.json` をロード
	sources, _, err := store.LoadSources(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load sources.json: %w", err)
	}

	notes, _, err := store.LoadNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load notes.json: %w", err)
	}

	sourceNotes, sourceNotesJsonPath, err := store.LoadSourceNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load source_notes.json: %w", err)
	}

	// `sourceID` の存在を確認
	foundSource := false
	for _, s := range sources {
		if s.SourceID == sourceID {
			foundSource = true
			break
		}
	}
	if !foundSource {
		return fmt.Errorf("❌ Source ID '%s' not found", sourceID)
	}

	// `noteID` のタイトルを取得
	var noteTitle string
	foundNote := false
	for _, n := range notes {
		if n.ID == noteID {
			noteTitle = n.Title
			foundNote = true
			break
		}
	}
	if !foundNote {
		return fmt.Errorf("❌ Note ID '%s' not found", noteID)
	}

	// 既に紐づいていないか確認
	for _, sn := range sourceNotes {
		if sn.SourceID == sourceID && sn.NoteID == noteID {
			log.Printf("⚠️ Note %s is already linked to source %s", noteID, sourceID)
			return nil
		}
	}

	// `source_notes.json` に追加
	sourceNotes = append(sourceNotes, model.SourceNote{SourceID: sourceID, NoteID: noteID})

	// `source_notes.json` を保存
	err = store.SaveUpdatedJson(sourceNotes, sourceNotesJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update source_notes.json: %w", err)
	}

	log.Printf("✅ Index note '%s' added to source '%s'!", noteTitle, sourceID)
	return nil
}

// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Manage index notes",
	Args:  cobra.MaximumNArgs(1),
}

var newIndexCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "Add a new index note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {

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

		var indexTitle string
		var sourceID string

		if indexBookID != "" {
			sources, _, err := store.LoadSources(*config)
			if err != nil {
				log.Fatalf("❌ Failed to load sources.json: %v", err)
			}

			found := false
			for i := range sources {
				if sources[i].SourceID == indexBookID {
					indexTitle = sources[i].Title
					sourceID = sources[i].SourceID
					found = true
					break
				}
			}

			if !found {
				log.Fatalf("❌ Source ID '%s' not found", indexBookID)
			}
		} else {
			// 手動でタイトルを指定
			if len(args) == 0 {
				log.Fatalf("❌ You must specify a title or use --book")
			}
			indexTitle = args[0]
		}

		if len(indexTags) > 0 {
			if err := store.CreateNewTag(indexTags, *config); err != nil {
				log.Printf("❌ Failed to create tag: %v\n", err)
				return
			}
		}

		newIndexStr, note, err := createNewIndexNote(indexTitle, sourceID, *config)
		if err != nil {
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}

		for _, tagID := range indexTags {
			if err := store.InsertNoteTag(note.ID, tagID, *config); err != nil {
				log.Printf("❌ Failed to insert note-tag relation: %v\n", err)
			}
		}

		log.Printf("Opening %q (Title: %q)...", newIndexStr, indexTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newIndexStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}

		noteID := strings.TrimSuffix(filepath.Base(newIndexStr), ".md")

		mdContent, err := os.ReadFile(newIndexStr)
		if err != nil {
			log.Printf("❌ Failed to read Markdown file: %v", err)
		}

		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter for %s: %v", newIndexStr, err)
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

var indexListCmd = &cobra.Command{
	Use:     "list [title]",
	Short:   "List index notes",
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
			if indexTrash {
				if !note.Deleted {
					continue
				}
			} else if indexArchive {
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
				if note.NoteType != "index" {
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

		if indexSearchQuery != "" {
			searchResults := util.FullTextSearch(filteredNotes, indexSearchQuery)
			if len(searchResults) > 0 {
				filteredNotes = searchResults
			}
		}

		// 検索結果がある場合のみフィルタリング
		if len(filteredNotes) > 0 {
			filteredNotes = util.FilterNotes(filteredNotes, indexTags, indexFrom, indexTo, noteTagDisplay)
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
		if indexPageSize == -1 {
			indexPageSize = len(filteredNotes)
		}

		// ページネーションのループ
		for {
			start := page * indexPageSize
			end := start + indexPageSize

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

			if indexPageSize == len(filteredNotes) {
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

var showIndexCmd = &cobra.Command{
	Use:     "show [Note ID]",
	Short:   "Show index note detail",
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

		frontMatter, body, err := store.ParseFrontMatter[model.TaskFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("❌ Error parsing front matter: %v", err)
			os.Exit(1)
		}

		fmt.Printf("[%v] %v\n", titleStyle(frontMatter.ID), titleStyle(frontMatter.Title))
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Type: %v\n", frontMatterStyle(frontMatter.NoteType))
		fmt.Printf("Tags: %v\n", frontMatterStyle(frontMatter.Tags))
		fmt.Printf("Links: %v\n", frontMatterStyle(frontMatter.Links))
		fmt.Printf("Task status: %v\n", frontMatterStyle(frontMatter.Status))
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

var editIndexCmd = &cobra.Command{
	Use:     "edit [noteID]",
	Short:   "Edit an index note",
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

var deleteIndexCmd = &cobra.Command{
	Use:     "remove [noteID]",
	Short:   "Delete an index note",
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

var archiveIndexCmd = &cobra.Command{
	Use:     "archive [noteID]",
	Short:   "Archive a fleeting note",
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

var restoreIndexCmd = &cobra.Command{
	Use:     "restore [noteID]",
	Short:   "Restore an index note",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"rs"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v", err)
			os.Exit(1)
		}

		if permanentRestoreTrash && permanentRestoreArchive {
			log.Fatalf("❌ You cannot specify both --deleted and --archived")
		}

		// デフォルトは `--deleted`
		if !permanentRestoreTrash && !permanentRestoreArchive {
			permanentRestoreTrash = true
		}

		err = store.RestoreNote(noteID, *config, permanentRestoreTrash, permanentRestoreArchive)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

	},
}

func init() {
	indexCmd.AddCommand(newIndexCmd)
	indexCmd.AddCommand(indexListCmd)
	indexCmd.AddCommand(showIndexCmd)
	indexCmd.AddCommand(editIndexCmd)
	indexCmd.AddCommand(deleteIndexCmd)
	indexCmd.AddCommand(archiveIndexCmd)
	indexCmd.AddCommand(restoreIndexCmd)
	rootCmd.AddCommand(indexCmd)
	newIndexCmd.Flags().StringVar(&indexBookID, "book", "", "Create an index for a specific book (source ID)")
	newIndexCmd.Flags().StringSliceVarP(&indexTags, "tag", "t", []string{}, "Specify tags")
	indexListCmd.Flags().StringSliceVarP(&indexTags, "tag", "t", []string{}, "Filter by tags")
	indexListCmd.Flags().StringVar(&indexFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	indexListCmd.Flags().StringVar(&indexTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	indexListCmd.Flags().StringVarP(&indexSearchQuery, "search", "q", "", "Search by title or content")
	indexListCmd.Flags().IntVar(&indexPageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	indexListCmd.Flags().BoolVar(&indexTrash, "trash", false, "Show deleted notes")
	indexListCmd.Flags().BoolVar(&indexArchive, "archive", false, "Show archived notes")
	deleteIndexCmd.Flags().BoolVarP(&indexForceDelete, "force", "f", false, "Permanently delete the note")
	restoreIndexCmd.Flags().BoolVar(&indexRestoreTrash, "trash", false, "Restore from trash")
	restoreIndexCmd.Flags().BoolVar(&indexRestoreArchive, "archive", false, "Restore from archive")
}
