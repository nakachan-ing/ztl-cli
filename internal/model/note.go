package model

type Note struct {
	ID        string `json:"id"`     // yyyymmddhhmmss
	SeqID     string `json:"seq_id"` // 1...
	Title     string `json:"title"`
	NoteType  string `json:"note_type"` // fleeting, permanent, literature
	ProjectID string `json:"project_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"` // yyyy-mm-dd hh:mm:ss
	UpdatedAt string `json:"updated_at"` // yyyy-mm-dd hh:mm:ss
	Archived  bool   `json:"archived"`
	Deleted   bool   `json:"deleted"`
}
