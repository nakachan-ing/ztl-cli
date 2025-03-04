/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
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

var literatureTags []string
var literatureFrom string
var literatureTo string
var literatureSearchQuery string
var literaturePageSize int
var literatureTrash bool
var literatureArchive bool

func createNewLiteratureNote(literatureTitle string, config model.Config) (string, model.Note, error) {
	t := time.Now()
	noteId := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	createdAt := t.Format("2006-01-02 15:04:05")

	// Create front matter
	frontMatter := model.NoteFrontMatter{
		ID:        noteId,
		Title:     literatureTitle,
		NoteType:  "literature",
		Tags:      literatureTags,
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
		Title:     literatureTitle,
		NoteType:  "literature",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	err = store.InsertNoteToJson(note, config)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	fmt.Printf("✅ Literature Note %s has been created successfully.\n", filePath)
	return filePath, note, nil
}

// literatureCmd represents the literature command
var literatureCmd = &cobra.Command{
	Use:   "literature",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"lt"},
}

var newLiteratureCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "Add a new literature note",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {
		literatureTitle := args[0]

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

		if len(literatureTags) > 0 {
			if err := store.CreateNewTag(literatureTags, *config); err != nil {
				log.Printf("❌ Failed to create tag: %v\n", err)
				return
			}
		}

		newLiteratureStr, note, err := createNewLiteratureNote(literatureTitle, *config)
		if err != nil {
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}

		for _, tagID := range literatureTags {
			if err := store.InsertNoteTag(note.ID, tagID, *config); err != nil {
				log.Printf("❌ Failed to insert note-tag relation: %v\n", err)
			}
		}

		log.Printf("Opening %q (Title: %q)...", newLiteratureStr, literatureTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newLiteratureStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}

		noteID := strings.TrimSuffix(filepath.Base(newLiteratureStr), ".md")

		mdContent, err := os.ReadFile(newLiteratureStr)
		if err != nil {
			log.Printf("❌ Failed to read Markdown file: %v", err)
		}

		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(mdContent))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter for %s: %v", newLiteratureStr, err)
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

var literatureListCmd = &cobra.Command{
	Use:     "list [title]",
	Short:   "List literature notes",
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
			if literatureTrash {
				if !note.Deleted {
					continue
				}
			} else if literatureArchive {
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
				if note.NoteType != "literature" {
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

		if literatureSearchQuery != "" {
			searchResults := util.FullTextSearch(filteredNotes, literatureSearchQuery)
			if len(searchResults) > 0 {
				filteredNotes = searchResults
			}
		}

		// 検索結果がある場合のみフィルタリング
		if len(filteredNotes) > 0 {
			filteredNotes = util.FilterNotes(filteredNotes, literatureTags, literatureFrom, literatureTo, noteTagDisplay)
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
		if literaturePageSize == -1 {
			literaturePageSize = len(filteredNotes)
		}

		// ページネーションのループ
		for {
			start := page * literaturePageSize
			end := start + literaturePageSize

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

			if literaturePageSize == len(filteredNotes) {
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
	literatureCmd.AddCommand(newLiteratureCmd)
	literatureCmd.AddCommand(literatureListCmd)
	rootCmd.AddCommand(literatureCmd)
	newLiteratureCmd.Flags().StringSliceVarP(&literatureTags, "tag", "t", []string{}, "Specify tags")
	literatureListCmd.Flags().StringSliceVarP(&literatureTags, "tag", "t", []string{}, "Filter by tags")
	literatureListCmd.Flags().StringVar(&literatureFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	literatureListCmd.Flags().StringVar(&literatureTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	literatureListCmd.Flags().StringVarP(&literatureSearchQuery, "search", "q", "", "Search by title or content")
	literatureListCmd.Flags().IntVar(&literaturePageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	literatureListCmd.Flags().BoolVar(&literatureTrash, "trash", false, "Show deleted notes")
	literatureListCmd.Flags().BoolVar(&literatureArchive, "archive", false, "Show archived notes")
}
