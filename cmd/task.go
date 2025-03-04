/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/nakachan-ing/ztl-cli/internal/model"
	"github.com/nakachan-ing/ztl-cli/internal/store"
	"github.com/nakachan-ing/ztl-cli/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var taskTags []string
var status string
var taskFrom string
var taskTo string
var taskSearchQuery string
var taskPageSize int
var taskTrash bool
var taskArchive bool

func createNewTask(taskTitle string, config model.Config) (string, model.Note, error) {
	t := time.Now()
	noteId := fmt.Sprintf("%d%02d%02d%02d%02d%02d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())
	createdAt := t.Format("2006-01-02 15:04:05")

	// Create front matter
	frontMatter := model.TaskFrontMatter{
		ID:        noteId,
		Title:     taskTitle,
		NoteType:  "task",
		Tags:      taskTags,
		Status:    "Not started",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	// Convert to YAML format
	frontMatterBytes, err := yaml.Marshal(frontMatter)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Create Markdown content
	content := fmt.Sprintf("---\n%s---\n\n## %s", string(frontMatterBytes), frontMatter.Title)

	// Write to file
	filePath := fmt.Sprintf("%s/%s.md", config.ZettelDir, noteId)
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to create note file (%s): %w", filePath, err)
	}

	// Write to JSON file
	note := model.Note{
		ID:        noteId,
		SeqID:     "",
		Title:     taskTitle,
		NoteType:  "task",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Archived:  false,
		Deleted:   false,
	}

	err = store.InsertNoteToJson(note, config)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	task := model.Task{
		ID:     "",
		NoteID: noteId,
		Status: "Not Started",
	}

	err = store.InsertTaskToJson(task, config)
	if err != nil {
		return "", model.Note{}, fmt.Errorf("failed to write to JSON file: %w", err)
	}

	fmt.Printf("✅ Task %s has been created successfully.\n", filePath)
	return filePath, note, nil
}

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:     "task",
	Short:   "A brief description of your command",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"t"},
}

var newTaskCmd = &cobra.Command{
	Use:     "new [title]",
	Short:   "Add a new task",
	Args:    cobra.ExactArgs(1),
	Aliases: []string{"n"},
	Run: func(cmd *cobra.Command, args []string) {
		taskTitle := args[0]

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

		if len(taskTags) > 0 {
			if err := store.CreateNewTag(taskTags, *config); err != nil {
				log.Printf("❌ Failed to create tag: %v\n", err)
				return
			}
		}

		newTaskStr, note, err := createNewTask(taskTitle, *config)
		if err != nil {
			log.Printf("❌ Failed to create note: %v\n", err)
			return
		}

		for _, tagID := range taskTags {
			if err := store.InsertNoteTag(note.ID, tagID, *config); err != nil {
				log.Printf("❌ Failed to insert note-tag relation: %v\n", err)
			}
		}

		log.Printf("Opening %q (Title: %q)...", newTaskStr, taskTitle)
		time.Sleep(2 * time.Second)

		err = util.OpenEditor(newTaskStr, *config)
		if err != nil {
			log.Printf("❌ Failed to open editor: %v\n", err)
		}
	},
}

var listTaskCmd = &cobra.Command{
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

		// `tasks.json` をロード
		tasks, _, err := store.LoadTasks(*config)
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// `notes.json` をロード
		notes, _, err := store.LoadNotes(*config)
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// `note_tag.json` をロード
		noteTags, _, err := store.LoadNoteTags(*config)
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// `tags.json` をロード
		tags, _, err := store.LoadTags(*config)
		if err != nil {
			log.Printf("❌ Error loading config: %v\n", err)
			os.Exit(1)
		}

		// // ノートIDごとにノートデータをマッピング
		// noteMap := make(map[string]model.Note)
		// for _, note := range notes {
		// 	noteMap[note.ID] = note
		// }

		// // タスクIDごとにタスクデータをマッピング（ノート情報も含める）
		// taskMap := make(map[string]model.Task)
		// for _, task := range tasks {
		// 	if note, exists := noteMap[task.NoteID]; exists {
		// 		task.NoteID = note.ID // ノートIDをセット
		// 		taskMap[task.ID] = task
		// 	} else {
		// 		fmt.Printf("⚠️ Warning: Note with ID %s not found for task %s\n", task.NoteID, task.ID)
		// 	}
		// }

		// // ノートIDごとにタイトル、作成日、更新日をマッピング
		// noteMetaMap := make(map[string]struct {
		// 	Title     string
		// 	CreatedAt string
		// 	UpdatedAt string
		// })

		// for _, note := range notes {
		// 	noteMetaMap[note.ID] = struct {
		// 		Title     string
		// 		CreatedAt string
		// 		UpdatedAt string
		// 	}{
		// 		Title:     note.Title,
		// 		CreatedAt: note.CreatedAt,
		// 		UpdatedAt: note.UpdatedAt,
		// 	}
		// }

		// // タグID → タグ名のマッピング
		// tagMap := make(map[string]string)
		// for _, tag := range tags {
		// 	tagMap[tag.ID] = tag.Name
		// }

		// // ノートIDごとにタグ名のリストを作成
		// noteTagMap := make(map[string][]string)
		// for _, noteTag := range noteTags {
		// 	if tagName, exists := tagMap[noteTag.TagID]; exists {
		// 		noteTagMap[noteTag.NoteID] = append(noteTagMap[noteTag.NoteID], tagName)
		// 	}
		// }

		// // テーブル作成
		// t := table.NewWriter()
		// t.SetOutputMirror(os.Stdout)
		// t.SetStyle(table.StyleDouble)
		// t.AppendHeader(table.Row{
		// 	text.FgGreen.Sprintf("Task ID"), text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Title")),
		// 	text.FgGreen.Sprintf("Tags"),
		// 	text.FgGreen.Sprintf("Status"),
		// 	text.FgGreen.Sprintf("Created"),
		// 	text.FgGreen.Sprintf("Updated"),
		// })

		// // マッピングしたデータを表示
		// for _, task := range taskMap {
		// 	// `note_id` からタイトル, 作成日, 更新日を取得
		// 	meta, exists := noteMetaMap[task.NoteID]
		// 	if !exists {
		// 		meta.Title = "(Unknown Note)"
		// 		meta.CreatedAt = "-"
		// 		meta.UpdatedAt = "-"
		// 	}

		// 	// `note_id` からタグ名を取得
		// 	tagStr := strings.Join(noteTagMap[task.NoteID], ", ")

		// 	t.AppendRow(table.Row{task.ID, meta.Title, tagStr, task.Status, meta.CreatedAt, meta.UpdatedAt})
		// }

		// t.Render()

		// ノートIDごとに `model.Note` をマッピング
		noteMap := make(map[string]model.Note)
		for _, note := range notes {
			noteMap[note.ID] = note
		}

		// ノートIDごとに `Task` をマッピング
		taskMap := make(map[string]model.Task)
		for _, task := range tasks {
			taskMap[task.NoteID] = task
		}

		// タグID → タグ名のマッピング
		tagMap := make(map[string]string)
		for _, tag := range tags {
			tagMap[tag.ID] = tag.Name
		}

		// ノートID → タグ名のマッピング
		noteTagMap := make(map[string][]string)
		for _, noteTag := range noteTags {
			if tagName, exists := tagMap[noteTag.TagID]; exists {
				noteTagMap[noteTag.NoteID] = append(noteTagMap[noteTag.NoteID], tagName)
			}
		}

		// **タスクのみに絞った `filteredTasks` を作成**
		filteredTasks := []struct {
			Task model.Task
			Note model.Note
		}{}

		noteTagDisplay := make(map[string][]string)

		for _, note := range notes {
			// タスクでないノートはスキップ
			if note.NoteType != "task" {
				continue
			}

			// `taskMap` に存在しない `note_id` はスキップ
			task, exists := taskMap[note.ID]
			if !exists {
				continue
			}

			// `--trash`, `--archive` のフィルタ
			if taskTrash && !note.Deleted {
				continue
			} else if taskArchive && !note.Archived {
				continue
			} else {
				if note.Deleted {
					continue
				}
			}

			// `--tag` のフィルタリング
			tagNames := noteTagMap[note.ID]
			noteTagDisplay[note.ID] = tagNames
			if len(taskTags) > 0 && !util.HasTags(tagNames, taskTags) {
				continue
			}

			// `--from` / `--to` の日付フィルタ
			if !util.IsWithinDateRange(note.CreatedAt, taskFrom, taskTo) {
				continue
			}

			// `filteredTasks` に `Task` と `Note` をセット
			filteredTasks = append(filteredTasks, struct {
				Task model.Task
				Note model.Note
			}{Task: task, Note: note})
		}

		// ページネーションの準備
		reader := bufio.NewReader(os.Stdin)
		page := 0

		fmt.Println(strings.Repeat("=", 30))
		fmt.Printf("Tasks: %v tasks shown\n", len(filteredTasks))
		fmt.Println(strings.Repeat("=", 30))

		// `--limit` がない場合は全件表示
		if taskPageSize == -1 {
			taskPageSize = len(filteredTasks)
		}

		for {
			start := page * taskPageSize
			end := start + taskPageSize

			// 範囲チェック
			if start >= len(filteredTasks) {
				fmt.Println("No more tasks to display.")
				break
			}
			if end > len(filteredTasks) {
				end = len(filteredTasks)
			}

			// テーブル作成
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.SetStyle(table.StyleDouble)
			t.Style().Options.SeparateRows = false

			t.AppendHeader(table.Row{
				text.FgGreen.Sprintf("Task ID"), text.FgGreen.Sprintf("%s", text.Bold.Sprintf("Title")),
				text.FgGreen.Sprintf("Tags"),
				text.FgGreen.Sprintf("Status"),
				text.FgGreen.Sprintf("Created"), text.FgGreen.Sprintf("Updated"),
			})

			for _, row := range filteredTasks[start:end] {
				note := row.Note
				tagStr := strings.Join(noteTagDisplay[note.ID], ", ")
				taskStatus := row.Task.Status
				statusColored := taskStatus

				switch taskStatus {
				case "Not started":
					statusColored = text.FgHiRed.Sprintf("%s", taskStatus)
				case "In progress":
					statusColored = text.FgHiYellow.Sprintf("%s", taskStatus)
				case "Waiting":
					statusColored = text.FgHiBlue.Sprintf("%s", taskStatus)
				case "On hold":
					statusColored = text.FgHiMagenta.Sprintf("%s", taskStatus)
				case "Done":
					statusColored = text.FgHiGreen.Sprintf("%s", taskStatus)
				default:
					statusColored = taskStatus
				}

				t.AppendRow(table.Row{
					row.Task.ID,
					note.Title,
					tagStr,
					statusColored,
					note.CreatedAt,
					note.UpdatedAt,
				})
			}

			t.Render()

			// すべてのタスクを表示し終えたら終了
			if end >= len(filteredTasks) {
				break
			}

			// 次のページへ進む
			fmt.Print("\nPress Enter for the next page (q to quit): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "q" {
				break
			}

			page++
		}

	},
}

func init() {
	taskCmd.AddCommand(newTaskCmd)
	taskCmd.AddCommand(listTaskCmd)
	rootCmd.AddCommand(taskCmd)
	newTaskCmd.Flags().StringSliceVarP(&taskTags, "tag", "t", []string{}, "Specify tags")
	listTaskCmd.Flags().StringVar(&status, "status", "", "Filter by status")
	listTaskCmd.Flags().StringSliceVarP(&taskTags, "tag", "t", []string{}, "Filter by tags")
	listTaskCmd.Flags().StringVar(&taskFrom, "from", "", "Filter by start date (YYYY-MM-DD)")
	listTaskCmd.Flags().StringVar(&taskTo, "to", "", "Filter by end date (YYYY-MM-DD)")
	listTaskCmd.Flags().StringVarP(&taskSearchQuery, "search", "q", "", "Search by title or content")
	listTaskCmd.Flags().IntVar(&taskPageSize, "limit", 20, "Set the number of notes to display per page (-1 for all)")
	listTaskCmd.Flags().BoolVar(&taskTrash, "trash", false, "Show deleted notes")
	listTaskCmd.Flags().BoolVar(&taskArchive, "archive", false, "Show archived notes")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// taskCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// taskCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
