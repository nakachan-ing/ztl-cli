/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
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
	"github.com/spf13/cobra"
)

var tagSearchQuery string
var tagPageSize int

func AddTagToNote(noteID, tagName string, config model.Config) error {

	notes, _, err := store.LoadNotes(config)
	if err != nil {
		log.Printf("❌ Error loading notes from JSON: %v", err)
		os.Exit(1)
	}

	noteTags, noteTagsJsonPath, err := store.LoadNoteTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	tags, tagsJsonPath, err := store.LoadTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load tags.json: %w", err)
	}

	// 既にノートにタグが付いているか確認
	for _, noteTag := range noteTags {
		if noteTag.NoteID == noteID && noteTag.TagID == tagName {
			return fmt.Errorf("⚠️ Tag '%s' already exists on note %s", tagName, noteID)
		}
	}

	// `tags.json` にタグがなければ追加
	tagID := ""
	tagExists := false
	for _, tag := range tags {
		if tag.Name == tagName {
			tagID = tag.ID
			tagExists = true
			break
		}
	}
	if !tagExists {
		tagID = store.GetNextTagID(tags)
		newTag := model.Tag{ID: tagID, Name: tagName}
		tags = append(tags, newTag)
	}

	for i := range notes {
		if notes[i].SeqID == noteID {
			content, err := os.ReadFile(filepath.Join(config.ZettelDir, notes[i].ID+".md"))
			if err != nil {
				return fmt.Errorf("❌ Failed to read updated note file: %v", err)
			}
			frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(content))
			if err != nil {
				return fmt.Errorf("❌ Error parsing front matter: %v", err)
			}
			frontMatter.Tags = append(frontMatter.Tags, tagName)
			frontMatter.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

			notes[i].UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

			updatedContent := store.UpdateFrontMatter(&frontMatter, body)

			err = os.WriteFile(filepath.Join(config.ZettelDir, notes[i].ID+".md"), []byte(updatedContent), 0644)
			if err != nil {
				return fmt.Errorf("❌ Error writing updated note file: %v", err)
			}
			noteTags = append(noteTags, model.NoteTag{
				NoteID: notes[i].ID,
				TagID:  tagID,
			})
		}

	}
	// JSON ファイルを更新
	err = store.SaveUpdatedJson(noteTags, noteTagsJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update note_tags.json: %w", err)
	}

	err = store.SaveUpdatedJson(tags, tagsJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update tags.json: %w", err)
	}

	return nil
}

func RemoveTagFromNote(noteID, tagName string, config model.Config) error {
	notes, notesJsonPath, err := store.LoadNotes(config)
	if err != nil {
		log.Printf("❌ Error loading notes from JSON: %v", err)
		os.Exit(1)
	}

	noteTags, noteTagsJsonPath, err := store.LoadNoteTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	tags, tagsJsonPath, err := store.LoadTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load tags.json: %w", err)
	}

	// `tagName` に対応する `tagID` を取得
	var tagID string
	for _, tag := range tags {
		if tag.Name == tagName {
			tagID = tag.ID
			break
		}
	}

	if tagID == "" {
		return fmt.Errorf("❌ Tag '%s' not found in tags.json", tagName)
	}

	for i := range notes {
		if notes[i].SeqID == noteID {
			mdFilePath := filepath.Join(config.ZettelDir, notes[i].ID+".md")
			content, err := os.ReadFile(mdFilePath)
			if err != nil {
				return fmt.Errorf("❌ Failed to read updated note file: %v", err)
			}
			frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(content))
			if err != nil {
				return fmt.Errorf("❌ Error parsing front matter: %v", err)
			}
			frontMatter.Tags = removeTag(frontMatter.Tags, tagName)
			frontMatter.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

			notes[i].UpdatedAt = frontMatter.UpdatedAt

			updatedContent := store.UpdateFrontMatter(&frontMatter, body)

			err = os.WriteFile(mdFilePath, []byte(updatedContent), 0644)
			if err != nil {
				return fmt.Errorf("❌ Error writing updated note file: %v", err)
			}

			// `notes.json` を更新
			err = store.SaveUpdatedJson(notes, notesJsonPath)
			if err != nil {
				return fmt.Errorf("❌ Failed to update notes.json: %w", err)
			}

			// `note_tags.json` から `noteID` に紐づく `tagID` を削除
			updatedNoteTags := []model.NoteTag{}
			for _, noteTag := range noteTags {
				if !(noteTag.NoteID == notes[i].ID && noteTag.TagID == tagID) {
					updatedNoteTags = append(updatedNoteTags, noteTag)
				}
			}

			// `noteTags` を更新し、JSON に保存
			noteTags = updatedNoteTags
			err = store.SaveUpdatedJson(noteTags, noteTagsJsonPath)
			if err != nil {
				return fmt.Errorf("❌ Failed to update note_tags.json: %w", err)
			}

			// 修正: `updatedNoteTags` で `tagStillInUse` を判定
			tagStillInUse := false
			for _, noteTag := range updatedNoteTags { // 更新後の `updatedNoteTags` を使用
				if noteTag.TagID == tagID {
					tagStillInUse = true
					break
				}
			}

			// `tagID` がどのノートにも使われていなければ `tags.json` から削除
			if !tagStillInUse {
				updatedTags := []model.Tag{}
				for _, tag := range tags {
					if tag.ID != tagID {
						updatedTags = append(updatedTags, tag)
					}
				}

				// `tags.json` を更新し、データを保存
				tags = updatedTags
				err = store.SaveUpdatedJson(tags, tagsJsonPath)
				if err != nil {
					return fmt.Errorf("❌ Failed to update tags.json: %w", err)
				}
			}

			break // 対象ノートの更新が終わったらループを抜ける
		}
	}

	return nil
}

func removeTag(tags []string, tagToRemove string) []string {
	var updatedTags []string
	for _, tag := range tags {
		if tag != tagToRemove {
			updatedTags = append(updatedTags, tag)
		}
	}
	return updatedTags
}

func ListTags(config model.Config, searchQuery string, pageSize int) error {
	tags, _, err := store.LoadTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load tags.json: %w", err)
	}

	noteTags, _, err := store.LoadNoteTags(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	// タグの使用回数をカウント
	tagCount := make(map[string]int)
	for _, noteTag := range noteTags {
		tagCount[noteTag.TagID]++
	}

	// `--search` が指定された場合、タグ名でフィルタリング
	filteredTags := []model.Tag{}
	if searchQuery != "" {
		for _, tag := range tags {
			if strings.Contains(strings.ToLower(tag.Name), strings.ToLower(searchQuery)) {
				filteredTags = append(filteredTags, tag)
			}
		}
	} else {
		filteredTags = tags
	}

	if len(filteredTags) == 0 {
		fmt.Println("No matching tags found.")
		return nil
	}

	// `--limit` を適用（-1 の場合はすべて表示）
	if pageSize > 0 && len(filteredTags) > pageSize {
		filteredTags = filteredTags[:pageSize]
	}

	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("Zettelkasten: %v tags shown\n", len(filteredTags))
	fmt.Println(strings.Repeat("=", 30))

	// 表示用のテーブルを作成
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDouble)
	t.Style().Options.SeparateRows = false

	// ヘッダー
	t.AppendHeader(table.Row{
		text.FgGreen.Sprintf("Tag ID"), text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Tag Name")),
		text.FgGreen.Sprintf("Usage Count"),
	})

	// タグ一覧を表示
	for _, tag := range filteredTags {
		count := tagCount[tag.ID]
		t.AppendRow([]interface{}{tag.ID, tag.Name, count})
	}

	t.Render()
	return nil
}

// tagCmd represents the tag command
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

var addTagCmd = &cobra.Command{
	Use:     "add [noteID] [tag]",
	Short:   "Add a tag to a note",
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"a"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		tagName := args[1]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		err = AddTagToNote(noteID, tagName, *config)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

	},
}

var removeTagCmd = &cobra.Command{
	Use:     "remove [noteID] [tag]",
	Short:   "remove a tag from a note",
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"rm"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		tagName := args[1]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		err = RemoveTagFromNote(noteID, tagName, *config)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		fmt.Printf("✅ Tag '%s' removed from note %s\n", tagName, noteID)

	},
}

var listTagCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all tags",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		err = ListTags(*config, tagSearchQuery, tagPageSize)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

	},
}

func init() {
	tagCmd.AddCommand(addTagCmd)
	tagCmd.AddCommand(removeTagCmd)
	tagCmd.AddCommand(listTagCmd)
	rootCmd.AddCommand(tagCmd)
	listTagCmd.Flags().StringVarP(&tagSearchQuery, "search", "q", "", "Search by tag name")
	listTagCmd.Flags().IntVar(&tagPageSize, "limit", 20, "Set the number of tags to display per page (-1 for all)")
}
