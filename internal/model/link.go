package model

type Link struct {
	SourceNoteID string `json:"source_note_id"` //yyyymmddhhmmss
	TargetNoteID string `json:"target_note_id"` //yyyymmddhhmmss
}
