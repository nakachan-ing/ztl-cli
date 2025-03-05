package model

type Config struct {
	ZettelDir   string `yaml:"zettel_dir"`
	Editor      string `yaml:"editor"`
	JsonDataDir string `yaml:"json_data_dir"`
	ArchiveDir  string `yaml:"archive_dir"`
	Backup      struct {
		Enable    bool   `yaml:"enable"`
		Frequency int    `yaml:"frequency"`
		Retention int    `yaml:"retention"`
		BackupDir string `yaml:"backup_dir"`
	}
	Trash struct {
		Frequency int    `yaml:"frequency"`
		Retention int    `yaml:"retention"`
		TrashDir  string `yaml:"trash_dir"`
	}
	Sync struct {
		Enable     bool     `yaml:"enable"`
		Platform   string   `yaml:"platform"`
		Bucket     string   `yaml:"bucket"`
		AWSProfile string   `yaml:"aws_profile"`
		AWSRegion  string   `yaml:"aws_region"`
		Include    []string `yaml:"include"`
		Exclude    []string `yaml:"exclude"`
	}
}

func DefaultConfig() Config {
	return Config{
		ZettelDir:   "~/Zettelkasten",
		Editor:      "vim",
		JsonDataDir: "~/.config/ztl/data",
		ArchiveDir:  "~/.config/ztl/archive",
		Backup: struct {
			Enable    bool   `yaml:"enable"`
			Frequency int    `yaml:"frequency"`
			Retention int    `yaml:"retention"`
			BackupDir string `yaml:"backup_dir"`
		}{
			Enable:    true,
			Frequency: 7,
			Retention: 30,
			BackupDir: "~/.config/ztl/backup",
		},
		Trash: struct {
			Frequency int    `yaml:"frequency"`
			Retention int    `yaml:"retention"`
			TrashDir  string `yaml:"trash_dir"`
		}{
			Frequency: 7,
			Retention: 14,
			TrashDir:  "~/.config/ztl/trash",
		},
	}
}
