package model

type Note struct {
	ID          string `json:"id"`     // yyyymmddhhmmss
	SeqID       string `json:"seq_id"` // n001...
	Title       string `json:"title"`
	NoteType    string `json:"note_type"` // fleeting, permanent, literature
	ProjectName string `json:"project_name"`
	Status      string `json:"status"`
	Content     string `json:"content"`
	CreatedAt   string `json:"created_at"` // yyyy-mm-dd hh:mm:ss
	UpdatedAt   string `json:"updated_at"` // yyyy-mm-dd hh:mm:ss
	Archived    bool   `json:"archived"`
	Deleted     bool   `json:"deleted"`
}

type NoteFrontMatter struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	NoteType    string   `yaml:"note_type"`
	Tags        []string `yaml:"tags"`
	Links       []string `yaml:"links"`
	ProjectName string   `yaml:"project_name"`
	CreatedAt   string   `yaml:"created_at"`
	UpdatedAt   string   `yaml:"updated_at"`
	Archived    bool     `yaml:"archived"`
	Deleted     bool     `yaml:"deleted"`
}

func (n *NoteFrontMatter) SetDeleted() {
	n.Deleted = true
}

func (n *NoteFrontMatter) ResetDeleted() {
	n.Deleted = false
}

func (n *NoteFrontMatter) SetArchived() {
	n.Archived = true
}

func (n *NoteFrontMatter) ResetArchived() {
	n.Archived = false
}
