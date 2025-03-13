/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Model struct {
	cursor    int
	fields    []string
	config    model.Config
	textInput textinput.Model
	editMode  bool
}

func newModel(config model.Config) tea.Model {
	return &Model{
		cursor:    0,
		fields:    generateFieldList(),
		config:    config,
		textInput: textinput.New(),
		editMode:  false,
	}
}

func generateFieldList() []string {
	return []string{
		"ZettelDir", "Editor", "JsonDataDir", "ArchiveDir",
		"Backup.Frequency", "Backup.Retention", "Backup.BackupDir",
		"Trash.Frequency", "Trash.Retention", "Trash.TrashDir",
		"Sync.Platform", "Sync.Bucket", "Sync.AWSProfile", "Sync.AWSRegion",
		"Save & Exit",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) forceRedraw() tea.Msg {
	return tea.WindowSizeMsg{}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editMode {
			switch msg.String() {
			case "enter":
				m.updateConfig()
				m.editMode = false
				m.textInput.Blur()
				// **Bubble Tea の `tea.Batch()` を使ってリフレッシュ！**
				return m, tea.Batch(tea.ClearScreen, m.forceRedraw)
			case "esc":
				m.editMode = false
				m.textInput.Blur()
			default:
				m.textInput, _ = m.textInput.Update(msg)
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor == len(m.fields)-1 {
				if err := store.SaveConfig(m.config); err != nil {
					log.Printf("⚠️ Failed to save config file: %v", err)
				}
				return m, tea.Quit
			}
			m.editMode = true
			m.textInput.SetValue(m.getFieldValue(m.fields[m.cursor]))
			m.textInput.Focus()
		}
	}

	return m, nil
}

func (m Model) View() string {
	var s strings.Builder
	s.WriteString("\033[H\033[2J")
	s.WriteString("📄 Configure zk\n\n")

	// 余計な重複を防ぐために `fields` を固定してループ
	for i, field := range generateFieldList() {
		cursor := "  "
		if m.cursor == i {
			cursor = "👉"
		}

		// 設定値を取得
		value := m.getFieldValue(field)

		// 1つの項目ごとに表示
		s.WriteString(fmt.Sprintf("%s %s: %s\n", cursor, field, value))
	}

	if m.editMode {
		s.WriteString("\n✏️  Editing: " + generateFieldList()[m.cursor] + "\n")
		s.WriteString(m.textInput.View() + "\n")
		s.WriteString("(Enter to save, ESC to cancel)\n")
	} else {
		s.WriteString("\n⬆️⬇️ で移動, Enter で編集, Q で終了\n")
	}

	return s.String()
}

// 設定値を取得（修正後）
func (m Model) getFieldValue(field string) string {
	switch field {
	case "ZettelDir":
		return m.config.ZettelDir
	case "Editor":
		return m.config.Editor
	case "JsonDataDir":
		return m.config.JsonDataDir
	case "ArchiveDir":
		return m.config.ArchiveDir
	case "Backup.Frequency":
		return strconv.Itoa(m.config.Backup.Frequency)
	case "Backup.Retention":
		return strconv.Itoa(m.config.Backup.Retention)
	case "Backup.BackupDir":
		return m.config.Backup.BackupDir
	case "Trash.Frequency":
		return strconv.Itoa(m.config.Trash.Frequency)
	case "Trash.Retention":
		return strconv.Itoa(m.config.Trash.Retention)
	case "Trash.TrashDir":
		return m.config.Trash.TrashDir
	case "Sync.Platform":
		return m.config.Sync.Platform
	case "Sync.Bucket":
		return m.config.Sync.Bucket
	case "Sync.AWSProfile":
		return m.config.Sync.AWSProfile
	case "Sync.AWSRegion":
		return m.config.Sync.AWSRegion
	default:
		return "UNKNOWN"
	}
}

// 設定を更新
func (m *Model) updateConfig() {
	newValue := m.textInput.Value()

	// 選択された設定項目に応じて値を更新
	switch m.fields[m.cursor] {
	case "ZettelDir":
		m.config.ZettelDir = newValue
	case "Editor":
		m.config.Editor = newValue
	case "JsonDataDir":
		m.config.JsonDataDir = newValue
	case "ArchiveDir":
		m.config.ArchiveDir = newValue
	case "Backup.Frequency":
		if newInt, err := strconv.Atoi(newValue); err == nil {
			m.config.Backup.Frequency = newInt
		}
	case "Backup.Retention":
		if newInt, err := strconv.Atoi(newValue); err == nil {
			m.config.Backup.Retention = newInt
		}
	case "Backup.BackupDir":
		m.config.Backup.BackupDir = newValue
	case "Trash.Frequency":
		if newInt, err := strconv.Atoi(newValue); err == nil {
			m.config.Trash.Frequency = newInt
		}
	case "Trash.Retention":
		if newInt, err := strconv.Atoi(newValue); err == nil {
			m.config.Trash.Retention = newInt
		}
	case "Trash.TrashDir":
		m.config.Trash.TrashDir = newValue
	case "Sync.Platform":
		m.config.Sync.Platform = newValue
	case "Sync.Bucket":
		m.config.Sync.Bucket = newValue
	case "Sync.AWSProfile":
		m.config.Sync.AWSProfile = newValue
	case "Sync.AWSRegion":
		m.config.Sync.AWSRegion = newValue
	}

	// 設定を保存
	if err := store.SaveConfig(m.config); err != nil {
		log.Printf("⚠️ Failed to save config file: %v", err)
	}

}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure config.yaml interactively",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := store.GetConfigPath()
		if err != nil {
			log.Printf("failed to get config path: %v", err)
		}

		fmt.Println(configPath)

		configByte, err := os.ReadFile(configPath)
		if err != nil {
			log.Printf("❌ Failed to read config file: %v", err)
			os.Exit(1)
		}

		var config model.Config

		if err = yaml.Unmarshal(configByte, &config); err != nil {
			log.Printf("failed to parse YAML: %v", err)
		}

		if _, err := tea.NewProgram(newModel(config)).Run(); err != nil {
			log.Fatalf("❌ Error running TUI: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
