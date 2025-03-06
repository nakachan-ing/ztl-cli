package model

type LockFile struct {
	ID        string `yaml:"id"`
	User      string `yaml:"user"`
	Pid       int    `yaml:"pid"`
	TimeStamp string `yaml:"timestamp"`
}
