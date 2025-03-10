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
var filterOnlyLinked bool
var filterQuery string

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

	uniqueLinks := make(map[string]model.Link)

	for _, note := range notes {
		mdFilePath := filepath.Join(config.ZettelDir, note.ID+".md")
		content, err := os.ReadFile(mdFilePath)
		if err != nil {
			log.Printf("⚠️ Failed to read note file: %s (%v)", mdFilePath, err)
			continue
		}

		// フロントマターを解析
		frontMatter, body, err := store.ParseFrontMatter[model.NoteFrontMatter](string(content))
		if err != nil {
			log.Printf("⚠️ Failed to parse front matter: %s (%v)", mdFilePath, err)
			continue
		}

		// フロントマターの `links:` を取得
		for _, targetID := range frontMatter.Links {
			key := fmt.Sprintf("%s-%s", note.ID, targetID)
			uniqueLinks[key] = model.Link{
				SourceNoteID: note.ID,
				TargetNoteID: targetID,
			}
		}

		// 本文の `[タイトル](yyyymmddhhmmss.md)` 形式のリンクを取得
		markdownLinks := extractMarkdownLinks(string(body))
		for _, targetID := range markdownLinks {
			key := fmt.Sprintf("%s-%s", note.ID, targetID)
			uniqueLinks[key] = model.Link{
				SourceNoteID: note.ID,
				TargetNoteID: targetID,
			}
		}
	}

	// `links.json` に保存
	linksJsonPath := filepath.Join(config.JsonDataDir, "links.json")
	links := make([]model.Link, 0, len(uniqueLinks))
	for _, link := range uniqueLinks {
		links = append(links, link)
	}

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

	// ノートID → タイトルのマッピング
	noteTitleMap := make(map[string]string)
	for _, note := range notes {
		noteTitleMap[note.ID] = note.Title
	}

	// ノートごとにリンクを整理
	noteLinks := make(map[string][]string)
	noteBacklinks := make(map[string][]string)

	for _, link := range links {
		// 順方向リンク
		noteLinks[link.SourceNoteID] = append(noteLinks[link.SourceNoteID], fmt.Sprintf("%s (%s)", link.TargetNoteID, noteTitleMap[link.TargetNoteID]))

		// 逆方向リンク（バックリンク）
		noteBacklinks[link.TargetNoteID] = append(noteBacklinks[link.TargetNoteID], fmt.Sprintf("%s (%s)", link.SourceNoteID, noteTitleMap[link.SourceNoteID]))
	}

	if len(links) == 0 {
		fmt.Println("No matching links found.")
		return nil
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Zettelkasten: %v links shown\n", len(links))
	fmt.Println(strings.Repeat("=", 80))

	// テーブル作成
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDouble)
	t.Style().Options.SeparateRows = false

	// ヘッダー
	t.AppendHeader(table.Row{"NOTE ID", "TITLE", "LINKS (→)", "BACKLINKS (←)"})

	// ノートごとにテーブルに追加
	for noteID, title := range noteTitleMap {
		linksStr := formatMultiline(noteLinks[noteID])
		backlinksStr := formatMultiline(noteBacklinks[noteID])

		// **フィルタリング処理**
		if filterOnlyLinked && linksStr == "-" && backlinksStr == "-" {
			continue // リンクがないノートはスキップ
		}
		if filterQuery != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(filterQuery)) {
			continue // タイトル検索
		}

		t.AppendRow([]interface{}{noteID, title, linksStr, backlinksStr})
	}

	t.Render()
	return nil
}

// リンクを改行区切りでフォーマット
func formatMultiline(links []string) string {
	if len(links) == 0 {
		return "-"
	}
	return strings.Join(links, "\n")
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
	linkListCmd.Flags().StringVar(&filterQuery, "query", "", "Filter links by title query")
	linkListCmd.Flags().BoolVar(&filterOnlyLinked, "only-linked", false, "Show only notes with links")
}
