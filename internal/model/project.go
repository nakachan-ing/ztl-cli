package model

type Project struct {
	ProjectID string `json:"project_id"` // p001...
	SeqID     string `json:"seq_id"`     // 1...
	Name      string `json:"name"`
}
