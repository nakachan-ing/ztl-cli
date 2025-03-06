package model

type NoteTag struct {
	NoteID string `json:"note_id"` // yyyymmddhhmmss
	TagID  string `json:"tag_id"`  // t001...
}
