package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
)

func FullTextSearch(notes []model.Note, query string) []model.Note {
	if query == "" {
		return notes
	}

	// ripgrep が使えるなら、それで全文検索（推奨）
	if isRipgrepAvailable() {
		return searchWithRipgrep(notes, query)
	}

	// ripgrep がない場合は、Go でファイルを1つずつ開いて検索
	return searchWithGo(notes, query)
}

// ripgrep の有無を確認
func isRipgrepAvailable() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// ripgrep を使った全文検索（高速）
func searchWithRipgrep(notes []model.Note, query string) []model.Note {
	config, err := store.LoadConfig()
	if err != nil {
		log.Printf("❌ Error loading config: %v", err)
		return nil
	}

	cmd := exec.Command("rg", "--ignore-case", "--files-with-matches", query)

	// メモファイルのパスを ripgrep に渡す
	var paths []string
	for _, note := range notes {
		paths = append(paths, filepath.Join(config.ZettelDir, note.ID+".md"))
	}
	cmd.Args = append(cmd.Args, paths...)

	out, err := cmd.Output()
	if err != nil {
		log.Printf("❌ Error running ripgrep: %v", err)
		return nil
	}

	// ripgrep の結果に含まれるファイルだけを `filteredNotes` に追加
	matchedPaths := strings.Split(strings.TrimSpace(string(out)), "\n")
	var filteredNotes []model.Note
	for _, note := range notes {
		if contains(matchedPaths, filepath.Join(config.ZettelDir, note.ID+".md")) {
			filteredNotes = append(filteredNotes, note)
		}
	}

	fmt.Printf("📌 Debug: Ripgrep found %d matching notes\n", len(filteredNotes))
	return filteredNotes
}

// Go でファイルを開いて検索（遅いが代替手段）
func searchWithGo(notes []model.Note, query string) []model.Note {
	var filteredNotes []model.Note
	config, err := store.LoadConfig()
	if err != nil {
		log.Printf("❌ Error loading config: %v", err)
		return nil
	}

	for _, note := range notes {
		content, err := os.ReadFile(filepath.Join(config.ZettelDir, note.ID+".md"))
		if err != nil {
			log.Printf("❌ Error reading file %s: %v", filepath.Join(config.ZettelDir, note.ID+".md"), err)
			continue
		}

		// タイトルまたは本文に `query` が含まれているかチェック
		if strings.Contains(strings.ToLower(note.Title), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
			filteredNotes = append(filteredNotes, note)
		}
	}

	fmt.Printf("📌 Debug: Go search found %d matching notes\n", len(filteredNotes))
	return filteredNotes
}

func contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

// ノートのフィルター処理（タグ + 日付）
func FilterNotes(notes []model.Note, tags []string, fromDate, toDate string, noteTagDisplay map[string][]string) []model.Note {
	var filteredNotes []model.Note

	for _, note := range notes {
		// タグのフィルタリング（noteTagDisplay を使用）
		if len(tags) > 0 && !HasTags(noteTagDisplay[note.ID], tags) {
			continue
		}

		// 日付のフィルタリング
		if !IsWithinDateRange(note.CreatedAt, fromDate, toDate) {
			continue
		}

		filteredNotes = append(filteredNotes, note)
	}

	return filteredNotes
}

// 指定されたタグが含まれているかチェック
func HasTags(noteTags []string, filterTags []string) bool {
	for _, filterTag := range filterTags {
		for _, noteTag := range noteTags {
			if strings.EqualFold(noteTag, filterTag) {
				return true
			}
		}
	}
	return false
}

// 日付が指定範囲内かチェック
func IsWithinDateRange(noteDateTime string, fromDate, toDate string) bool {
	noteDate := strings.Split(noteDateTime, " ")[0]

	// 日付が空の場合はフィルターしない
	if fromDate == "" && toDate == "" {
		return true
	}

	// メモの日付をパース
	noteTime, err := time.Parse("2006-01-02", noteDate)
	if err != nil {
		return false
	}

	// `from` の指定がある場合
	if fromDate != "" {
		fromTime, err := time.Parse("2006-01-02", fromDate)
		if err == nil && noteTime.Before(fromTime) {
			return false
		}
	}

	// `to` の指定がある場合
	if toDate != "" {
		toTime, err := time.Parse("2006-01-02", toDate)
		if err == nil && noteTime.After(toTime) {
			return false
		}
	}

	return true
}
