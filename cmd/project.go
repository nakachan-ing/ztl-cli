/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/text"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/spf13/cobra"
)

func createNewProject(projectName string, config model.Config) error {
	project := model.Project{
		ProjectID: "",
		Name:      projectName,
	}

	err := store.InsertProjectToJson(project, config)
	if err != nil {
		return fmt.Errorf("failed to write to JSON file: %w", err)
	}

	fmt.Printf("✅ Project %s has been created successfully.\n", projectName)
	return nil
}

func addNoteToProject(noteID, projectID string, config model.Config) (model.Note, model.Project, error) {
	projects, _, err := store.LoadProjects(config)
	if err != nil {
		return model.Note{}, model.Project{}, fmt.Errorf("❌ Failed to load to projects.json: %w", err)
	}

	notes, _, err := store.LoadNotes(config)
	if err != nil {
		return model.Note{}, model.Project{}, fmt.Errorf("❌ Failed to load to notes.json: %w", err)
	}

	projectNote := model.ProjectNote{}

	var matchedProject model.Project
	foundProject := false
	for _, project := range projects {
		if projectID == project.ProjectID {
			projectNote.ProjectID = project.ProjectID
			matchedProject = project // マッチしたノートを格納
			foundProject = true
			break // マッチしたらループを抜ける
		}
	}

	var matchedNote model.Note
	foundNote := false
	for _, note := range notes {
		if noteID == note.SeqID {
			projectNote.NoteID = note.ID
			matchedNote = note // マッチしたノートを格納
			foundNote = true
			break // マッチしたらループを抜ける
		}
	}

	// プロジェクトまたはノートが見つからなかった場合のエラーハンドリング
	if !foundProject {
		return model.Note{}, model.Project{}, fmt.Errorf("❌ Error: Project with SeqID %s not found", projectID)
	}
	if !foundNote {
		return model.Note{}, model.Project{}, fmt.Errorf("❌ Error: Note with SeqID %s not found", noteID)
	}

	err = store.InsertProjectNoteToJson(projectNote, config)
	if err != nil {
		return model.Note{}, model.Project{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	return matchedNote, matchedProject, nil
}

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:     "project",
	Short:   "A brief description of your command",
	Aliases: []string{"pj"},
}

var newProjectCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "create a new project",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {
		projectName := args[0]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		if err = createNewProject(projectName, *config); err != nil {
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}
	},
}

var addProjectCmd = &cobra.Command{
	Use:     "add [noteID] [projectID]",
	Short:   "Add note to project",
	Args:    cobra.ExactArgs(2),
	Aliases: []string{"a"},
	Run: func(cmd *cobra.Command, args []string) {
		noteID := args[0]
		projectID := args[1]

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		// Load notes from JSON
		notes, noteJsonPath, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		for i := range notes {
			if noteID == notes[i].SeqID {
				note, project, err := addNoteToProject(noteID, projectID, *config)
				if err != nil {
					log.Printf("❌ Failed to associate note & project: %v\n", err)
					return
				}

				content, err := os.ReadFile(filepath.Join(config.ZettelDir, note.ID+".md"))
				if err != nil {
					log.Printf("❌ Failed to read updated note file: %v", err)
					os.Exit(1)
				}

				frontMatter, body, err := store.ParseFrontMatter(string(content))
				if err != nil {
					log.Printf("❌ Error parsing front matter: %v", err)
					os.Exit(1)
				}

				notes[i].ProjectName = project.Name
				frontMatter.ProjectName = project.Name

				updatedContent := store.UpdateFrontMatter(&frontMatter, body)

				err = os.WriteFile(filepath.Join(config.ZettelDir, note.ID+".md"), []byte(updatedContent), 0644)
				if err != nil {
					log.Printf("❌ Error writing updated note file: %v", err)
					return
				}

				err = store.SaveUpdatedJson(notes, noteJsonPath)
				if err != nil {
					log.Printf("❌ Error updating JSON file: %v", err)
					return
				}
			}
		}

	},
}

var listProjectCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	// Args:    cobra.ExactArgs(2),
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {

		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		projects, _, err := store.LoadProjects(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		// プロジェクトに紐づくノート数を取得
		projectNotes, _, err := store.LoadProjectNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		// プロジェクトごとのノート数をマッピング
		projectNoteCount := make(map[string]int)
		for _, pn := range projectNotes {
			projectNoteCount[pn.ProjectID]++
		}

		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf("Zettelkasten: %v projects shown\n", len(projectNotes))
		fmt.Println(strings.Repeat("=", 30))

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleDouble)
		t.Style().Options.SeparateRows = false

		t.AppendHeader(table.Row{
			text.FgGreen.Sprintf("Project ID"),
			text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Project Name")),
			text.FgGreen.Sprintf("Notes Count"),
		})

		// プロジェクトをテーブルに追加
		for _, project := range projects {
			t.AppendRow(table.Row{
				project.ProjectID,
				project.Name,
				projectNoteCount[project.ProjectID],
			})
		}

		t.Render()

	},
}

var showProjectCmd = &cobra.Command{
	Use:     "show",
	Short:   "Show project details",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {
		projectID := args[0]
		config, err := store.LoadConfig()
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // Perform cleanup tasks
		// if err := internal.CleanupBackups(config.Backup.BackupDir, time.Duration(config.Backup.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Backup cleanup failed: %v", err)
		// }
		// if err := internal.CleanupTrash(config.Trash.TrashDir, time.Duration(config.Trash.Retention)*24*time.Hour); err != nil {
		// 	log.Printf("⚠️ Trash cleanup failed: %v", err)
		// }

		projects, _, err := store.LoadProjects(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		// プロジェクトに紐づくノート数を取得
		projectNotes, _, err := store.LoadProjectNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		notes, _, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading notes from JSON: %v", err)
			os.Exit(1)
		}

		var project model.Project
		foundProject := false
		for _, p := range projects {
			if p.ProjectID == projectID {
				project = p
				foundProject = true
				break
			}
		}

		if !foundProject {
			log.Printf("❌ Error: Project with ID %s not found", projectID)
			os.Exit(1)
		}

		// プロジェクトに紐づくノートを取得
		var noteIDs []string
		for _, pn := range projectNotes {
			if pn.ProjectID == projectID {
				noteIDs = append(noteIDs, pn.NoteID)
			}
		}

		// ノートID に対応するノート名を取得
		noteMap := make(map[string]string)
		for _, note := range notes {
			noteMap[note.ID] = note.Title
		}

		fmt.Printf("📖 Project: %s (%s)\n", project.Name, project.ProjectID)
		fmt.Println("   🏷 Notes:")
		if len(noteIDs) == 0 {
			fmt.Println("   - No notes assigned to this project.")
		} else {
			for _, noteID := range noteIDs {
				if title, exists := noteMap[noteID]; exists {
					fmt.Printf("   - [%s] %s\n", noteID, title)
				} else {
					fmt.Printf("   - [%s] (Note not found)\n", noteID)
				}
			}
		}
	},
}

func init() {
	projectCmd.AddCommand(newProjectCmd)
	projectCmd.AddCommand(addProjectCmd)
	projectCmd.AddCommand(listProjectCmd)
	projectCmd.AddCommand(showProjectCmd)
	rootCmd.AddCommand(projectCmd)
}
