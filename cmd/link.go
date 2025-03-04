/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
)

var filterTag string

func filterLinksByTag(links []model.Link, tag string, config model.Config) ([]model.Link, error) {
	// ノートとタグの対応関係を取得
	noteTags, _, err := store.LoadNoteTags(config)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to load note_tags.json: %w", err)
	}

	tags, _, err := store.LoadTags(config)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to load tags.json: %w", err)
	}

	// タグ名から `tagID` を取得
	var tagID string
	for _, t := range tags {
		if strings.EqualFold(t.Name, tag) {
			tagID = t.ID
			break
		}
	}

	if tagID == "" {
		return nil, fmt.Errorf("❌ Tag '%s' not found", tag)
	}

	// `tagID` を持つノートIDを取得
	noteMap := make(map[string]bool)
	for _, nt := range noteTags {
		if nt.TagID == tagID {
			noteMap[nt.NoteID] = true
		}
	}

	// 該当タグを持つノートのリンクのみフィルタリング
	var filteredLinks []model.Link
	for _, link := range links {
		if noteMap[link.SourceNoteID] || noteMap[link.TargetNoteID] {
			filteredLinks = append(filteredLinks, link)
		}
	}

	return filteredLinks, nil
}

func UpdateLinksJson(config model.Config) error {
	notes, _, err := store.LoadNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load notes.json: %w", err)
	}

	var links []model.Link

	// すべてのノートを走査
	for _, note := range notes {
		mdFilePath := filepath.Join(config.ZettelDir, note.ID+".md")

		content, err := os.ReadFile(mdFilePath)
		if err != nil {
			log.Printf("⚠️ Failed to read note file: %s (%v)", mdFilePath, err)
			continue
		}

		// フロントマターからリンクを取得
		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(content))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter: %s (%v)", mdFilePath, err)
			continue
		}

		// フロントマターの `links:` を取得
		for _, targetID := range frontMatter.Links {
			links = append(links, model.Link{
				SourceNoteID: note.ID,
				TargetNoteID: targetID,
			})
		}

		// 本文から `[タイトル](yyyymmddhhmmss.md)` 形式のリンクを解析
		markdownLinks := extractMarkdownLinks(string(body))
		for _, targetID := range markdownLinks {
			links = append(links, model.Link{
				SourceNoteID: note.ID,
				TargetNoteID: targetID,
			})
		}
	}

	// `links.json` に保存
	linksJsonPath := filepath.Join(config.JsonDataDir, "links.json")
	err = store.SaveUpdatedJson(links, linksJsonPath)
	if err != nil {
		return fmt.Errorf("❌ Failed to update links.json: %w", err)
	}

	log.Println("✅ Updated links.json")
	return nil
}

func extractMarkdownLinks(content string) []string {
	var links []string
	re := regexp.MustCompile(`\[(.*?)\]\((\d{14})\.md\)`) // `[タイトル](yyyymmddhhmmss.md)`
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 2 {
			links = append(links, match[2]) // `yyyymmddhhmmss`
		}
	}
	return links
}

func displayLinks(links []model.Link, config model.Config) error {
	// `notes.json` をロード
	notes, _, err := store.LoadNotes(config)
	if err != nil {
		return fmt.Errorf("❌ Failed to load notes.json: %w", err)
	}

	// ノートID → タイトルのマッピングを作成
	noteTitleMap := make(map[string]string)
	for _, note := range notes {
		noteTitleMap[note.ID] = note.Title
	}

	if len(links) == 0 {
		fmt.Println("No matching links found.")
		return nil
	}

	fmt.Println(strings.Repeat("=", 30))
	fmt.Printf("Zettelkasten: %v links shown\n", len(links))
	fmt.Println(strings.Repeat("=", 30))

	// テーブル作成
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDouble)
	t.Style().Options.SeparateRows = false

	// ヘッダー
	t.AppendHeader(table.Row{"SOURCE NOTE ID", "TARGET NOTE ID", "SOURCE TITLE", "TARGET TITLE"})

	// リンクをテーブルに追加
	for _, link := range links {
		sourceTitle := noteTitleMap[link.SourceNoteID]
		targetTitle := noteTitleMap[link.TargetNoteID]
		t.AppendRow([]interface{}{link.SourceNoteID, link.TargetNoteID, sourceTitle, targetTitle})
	}

	t.Render()
	return nil
}

// linkCmd represents the link command
var linkCmd = &cobra.Command{
	Use:     "link",
	Short:   "List all note links",
	Aliases: []string{"ln"},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("link called")
	},
}

var linkListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all note links",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		// `links.json` を最新の状態に更新
		err = UpdateLinksJson(*config)
		if err != nil {
			log.Fatalf("❌ Failed to update links.json: %v", err)
		}

		// `links.json` をロード
		links, _, err := store.LoadLinks(*config)
		if err != nil {
			log.Fatalf("❌ Failed to load links.json: %v", err)
		}

		// タグでフィルタリング（指定がある場合）
		if filterTag != "" {
			links, err = filterLinksByTag(links, filterTag, *config)
			if err != nil {
				log.Fatalf("❌ Failed to filter links: %v", err)
			}
		}

		// テーブル表示
		displayLinks(links, *config)
	},
}

func init() {
	linkCmd.AddCommand(linkListCmd)
	rootCmd.AddCommand(linkCmd)
	linkListCmd.Flags().StringVar(&filterTag, "tag", "", "Filter links by tag")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// linkCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// linkCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
