package model

type Task struct {
	ID     string `json:"id"`      // task-001...
	NoteID string `json:"note_id"` // yyyymmddhhmmss
	Status string `json:"status"`  // Not started, In progress, Waiting, On hold, Done
}
