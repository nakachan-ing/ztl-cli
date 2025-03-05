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

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
)

var sourceType, sourceTitle, sourceAuthor, sourcePublisher, sourceURL, sourceISBN string
var sourceYear int
var sourcePageSize int
var newTitle, newAuthor, newPublisher, newURL, newISBN string
var newYear int

// sourceCmd represents the source command
var sourceCmd = &cobra.Command{
	Use:   "source",
	Short: "A brief description of your command",
}

var sourceNewCmd = &cobra.Command{
	Use:   "new",
	Short: "Add a new source (book, web, paper, video)",
	Run: func(cmd *cobra.Command, args []string) {
		if sourceTitle == "" {
			log.Fatalf("❌ You must specify a title with --title")
		}

		if sourceType != "book" && sourceType != "web" && sourceType != "paper" && sourceType != "video" {
			log.Fatalf("❌ Invalid source type: %s. Must be 'book', 'web', 'paper', or 'video'", sourceType)
		}

		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		sources, _, err := store.LoadSources(*config)
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		source := model.Source{
			SourceID:   store.GetNextSourceID(sources),
			SourceType: sourceType,
			Title:      sourceTitle,
			Author:     sourceAuthor,
			Publisher:  sourcePublisher,
			Year:       sourceYear,
			ISBN:       sourceISBN,
			URL:        sourceURL,
		}

		err = store.InsertSourceToJson(source, *config)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		log.Printf("✅ Added new source: %s (%s)", source.Title, source.SourceType)
	},
}

var sourceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all sources (book, web, paper, video)",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		sources, _, err := store.LoadSources(*config)
		if err != nil {
			log.Printf("❌ Failed to load sources.json: %v", err)
		}

		// Handle case where no notes match
		if len(sources) == 0 {
			fmt.Println("No matching notes found.")
			return
		}

		reader := bufio.NewReader(os.Stdin)
		page := 0

		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf("Zettelkasten: %v notes shown\n", len(sources))
		fmt.Println(strings.Repeat("=", 30))

		if sourcePageSize == -1 {
			sourcePageSize = len(sources)
		}

		// ページネーションのループ
		for {
			start := page * sourcePageSize
			end := start + sourcePageSize

			// 範囲チェック
			if start >= len(sources) {
				fmt.Println("No more notes to display.")
				break
			}
			if end > len(sources) {
				end = len(sources)
			}

			// テーブル作成
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.SetStyle(table.StyleDouble)
			t.Style().Options.SeparateRows = false

			t.AppendHeader(table.Row{
				text.FgGreen.Sprintf("Source ID"),
				text.FgGreen.Sprintf("Source Type"),
				text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Title")),
				text.FgGreen.Sprintf("Author"),
				text.FgGreen.Sprintf("Publisher"),
				text.FgGreen.Sprintf("Year"), text.FgGreen.Sprintf("URL"),
			})

			// フィルタされたノートをテーブルに追加
			for _, row := range sources[start:end] {

				t.AppendRow(table.Row{
					row.SourceID,
					row.SourceType,
					row.Title,
					row.Author,
					row.Publisher,
					row.Year,
					row.URL,
				})
			}

			t.Render()

			if permanentPageSize == len(sources) {
				break
			}

			if end >= len(sources) {
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

var sourceShowCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show source detail (book, web, paper, video)",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceID := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		sources, _, err := store.LoadSources(*config)
		if err != nil {
			log.Printf("❌ Failed to load sources.json: %v", err)
		}

		var source model.Source
		found := false
		for i := range sources {
			if sources[i].SourceID == sourceID {
				source = sources[i]
				found = true
				break
			}
		}

		if !found {
			log.Printf("❌ Source ID '%s' not found", sourceID)
			os.Exit(1)
		}

		fmt.Printf("📖 %s\n", source.Title)
		fmt.Println(strings.Repeat("─", len(source.Title)+3))
		fmt.Printf("Type:      %s\n", source.SourceType)
		fmt.Printf("Author:    %s\n", source.Author)
		fmt.Printf("Publisher: %s\n", source.Publisher)
		fmt.Printf("Year:      %d\n", source.Year)
		if source.URL != "" {
			fmt.Printf("URL:       %s\n", source.URL)
		}
		fmt.Println()

	},
}

var sourceEditCmd = &cobra.Command{
	Use:     "edit",
	Short:   "edit source detail (book, web, paper, video)",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"e"},
	Run: func(cmd *cobra.Command, args []string) {
		sourceID := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Fatalf("❌ Error loading config: %v", err)
		}

		sources, sourcesJsonPath, err := store.LoadSources(*config)
		if err != nil {
			log.Printf("❌ Failed to load sources.json: %v", err)
		}

		found := false
		for i := range sources {
			if sources[i].SourceID == sourceID {
				found = true

				// 指定されたオプションのみ更新
				if cmd.Flags().Changed("title") {
					sources[i].Title = newTitle
				}
				if cmd.Flags().Changed("author") {
					sources[i].Author = newAuthor
				}
				if cmd.Flags().Changed("publisher") {
					sources[i].Publisher = newPublisher
				}
				if cmd.Flags().Changed("year") {
					sources[i].Year = newYear
				}
				if cmd.Flags().Changed("isbn") {
					sources[i].ISBN = newISBN
				}
				if cmd.Flags().Changed("url") {
					sources[i].URL = newURL
				}

				break
			}
		}

		if !found {
			log.Fatalf("❌ Source ID '%s' not found", sourceID)
		}

		// `sources.json` を更新
		err = store.SaveUpdatedJson(sources, sourcesJsonPath)
		if err != nil {
			log.Fatalf("❌ Failed to update sources.json: %v", err)
		}

		log.Printf("✅ Source '%s' updated successfully!", sourceID)
	},
}

func init() {
	sourceCmd.AddCommand(sourceNewCmd)
	sourceCmd.AddCommand(sourceListCmd)
	sourceCmd.AddCommand(sourceShowCmd)
	sourceCmd.AddCommand(sourceEditCmd)
	rootCmd.AddCommand(sourceCmd)
	sourceNewCmd.Flags().StringVar(&sourceType, "type", "", "Source type (book, web, paper, video)")
	sourceNewCmd.Flags().StringVar(&sourceTitle, "title", "", "Title of the source")
	sourceNewCmd.Flags().StringVar(&sourceAuthor, "author", "", "Author of the source")
	sourceNewCmd.Flags().StringVar(&sourcePublisher, "publisher", "", "Publisher")
	sourceNewCmd.Flags().IntVar(&sourceYear, "year", 0, "Publication year")
	sourceNewCmd.Flags().StringVar(&sourceISBN, "isbn", "", "ISBN (for books only)")
	sourceNewCmd.Flags().StringVar(&sourceURL, "url", "", "URL (for web, video)")
	sourceListCmd.Flags().IntVar(&sourcePageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	sourceEditCmd.Flags().StringVar(&newTitle, "title", "", "New title")
	sourceEditCmd.Flags().StringVar(&newAuthor, "author", "", "New author")
	sourceEditCmd.Flags().StringVar(&newPublisher, "publisher", "", "New publisher")
	sourceEditCmd.Flags().IntVar(&newYear, "year", 0, "New publication year")
	sourceEditCmd.Flags().StringVar(&newISBN, "isbn", "", "New ISBN")
	sourceEditCmd.Flags().StringVar(&newURL, "url", "", "New URL")

}
