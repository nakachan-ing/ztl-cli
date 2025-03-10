package util

import (
	"log"
	"strings"
	"time"

	"github.com/nakachan-ing/ztl-cli/internal/model"
)

func FullTextSearch(notes []model.Note, query string) []model.Note {
	if query == "" {
		return notes
	}

	query = strings.ToLower(query) // 大文字小文字を無視
	var filteredNotes []model.Note

	for _, note := range notes {
		// タイトルまたは本文に `query` が含まれているかチェック
		if strings.Contains(strings.ToLower(note.Title), query) ||
			strings.Contains(strings.ToLower(note.Content), query) {
			filteredNotes = append(filteredNotes, note)
		}
	}

	log.Printf("📌 Debug: Found %d matching notes in JSON\n", len(filteredNotes))
	return filteredNotes
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
