package model

type Task struct {
	ID     string `json:"id"`      // task-001...
	NoteID string `json:"note_id"` // yyyymmddhhmmss
	Status string `json:"status"`  // Not started, In progress, Waiting, On hold, Done
}

type TaskFrontMatter struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	NoteType    string   `yaml:"note_type"` // fleeting, permanent, literature
	Tags        []string `yaml:"tags"`
	Links       []string `yaml:"links"`
	ProjectName string   `yaml:"project_name"`
	Status      string   `yaml:"status"`
	CreatedAt   string   `yaml:"created_at"`
	UpdatedAt   string   `yaml:"updated_at"`
	Archived    bool     `yaml:"archived"`
	Deleted     bool     `yaml:"deleted"`
}

func (t *TaskFrontMatter) SetDeleted() {
	t.Deleted = true
}

func (n *TaskFrontMatter) SetArchived() {
	n.Archived = true
}
