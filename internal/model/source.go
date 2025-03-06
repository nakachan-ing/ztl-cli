package model

type Source struct {
	SourceID   string `json:"source_id"`   // s001...
	SourceType string `json:"source_type"` // book, web, paper, video
	Title      string `json:"title"`
	Author     string `json:"author"`
	Publisher  string `json:"publisher"`
	Year       int    `json:"year"`
	ISBN       string `json:"isbn"`
	URL        string `json:"url"`
}
