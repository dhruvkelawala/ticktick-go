package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"ticktick-go/internal/api"
	"ticktick-go/internal/config"
	"ticktick-go/internal/format"
)

func init() {
	taskCmd.AddCommand(taskListCmd, taskAddCmd, taskGetCmd, taskDoneCmd, taskDeleteCmd, taskEditCmd)
	
	// Add global json flag to task commands
	taskListCmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output in JSON format")
	taskGetCmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output in JSON format")
	
	// List flags
	taskListCmd.Flags().StringP("project", "p", "", "Filter by project name")
	taskListCmd.Flags().Bool("all", false, "Show all tasks across all projects")
	taskListCmd.Flags().String("due", "", "Filter by due date (today, overdue)")
	taskListCmd.Flags().String("priority", "", "Filter by priority (high, medium, low)")
	taskListCmd.Flags().String("tag", "", "Filter by tag")
	
	// Add flags
	taskAddCmd.Flags().StringP("project", "p", "inbox", "Project name")
	taskAddCmd.Flags().StringP("priority", "P", "", "Priority (high, medium, low)")
	taskAddCmd.Flags().StringP("due", "d", "", "Due date (natural language)")
	taskAddCmd.Flags().String("tag", "", "Tags (comma-separated)")
	taskAddCmd.Flags().StringP("note", "n", "", "Task notes")
	taskAddCmd.Flags().StringP("remind", "r", "", "Reminders (comma-separated: 15m, 1h, 1d, on-time)")
	
	// Edit flags
	taskEditCmd.Flags().String("title", "", "New title")
	taskEditCmd.Flags().String("due", "", "New due date")
	taskEditCmd.Flags().String("priority", "", "New priority")
	taskEditCmd.Flags().String("tag", "", "New tags (comma-separated)")
	taskEditCmd.Flags().StringP("remind", "r", "", "Reminders (comma-separated: 15m, 1h, 1d, on-time)")
}

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		showAll, _ := cmd.Flags().GetBool("all")
		projectName, _ := cmd.Flags().GetString("project")
		dueFilter, _ := cmd.Flags().GetString("due")
		priorityFilter, _ := cmd.Flags().GetString("priority")
		tagFilter, _ := cmd.Flags().GetString("tag")

		var tasks []api.Task
		var err error

		if showAll {
			tasks, err = client.GetAllTasks()
		} else if projectName != "" {
			projectID, err := client.GetProjectIDByName(projectName)
			if err != nil {
				return err
			}
			tasks, err = client.GetProjectTasks(projectID)
		} else {
			tasks, err = client.GetInboxTasks()
		}

		if err != nil {
			return err
		}

		// Apply filters
		if dueFilter != "" {
			tasks = filterByDue(tasks, dueFilter)
		}
		if priorityFilter != "" {
			tasks = filterByPriority(tasks, priorityFilter)
		}
		if tagFilter != "" {
			tasks = filterByTag(tasks, tagFilter)
		}

		if jsonFlag {
			return format.OutputJSON(tasks)
		}

		return format.OutputTaskList(tasks, client)
	},
}

var taskAddCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		title := args[0]
		projectName, _ := cmd.Flags().GetString("project")
		priorityStr, _ := cmd.Flags().GetString("priority")
		dueStr, _ := cmd.Flags().GetString("due")
		tagsStr, _ := cmd.Flags().GetString("tag")
		note, _ := cmd.Flags().GetString("note")
		remindStr, _ := cmd.Flags().GetString("remind")

		// Get project ID
		var projectID string
		var err error
		if projectName == "inbox" || projectName == "" {
			projectID, err = client.GetInboxProjectID()
		} else {
			projectID, err = client.GetProjectIDByName(projectName)
		}
		if err != nil {
			return err
		}

		// Parse due date
		dueDate, err := api.ParseDueDate(dueStr, cfg.Timezone)
		if err != nil {
			return fmt.Errorf("failed to parse due date: %w", err)
		}

		// Parse tags
		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		// Parse reminders
		reminders, err := api.ParseReminders(remindStr)
		if err != nil {
			return fmt.Errorf("failed to parse reminders: %w", err)
		}

		task := &api.Task{
			ProjectID: projectID,
			Title:     title,
			Content:   note,
			Priority:  api.ParsePriority(priorityStr),
			DueDate:   dueDate,
			Tags:      tags,
			IsAllDay:  dueStr != "" && !strings.ContainsAny(dueStr, "0123456789"),
			Status:    0,
			Reminders: reminders,
		}

		created, err := client.CreateTask(task)
		if err != nil {
			return err
		}

		if jsonFlag {
			return format.OutputJSON(created)
		}

		fmt.Println("✓ Task created successfully!")
		fmt.Printf("  ID: %s\n", created.ID)
		fmt.Printf("  Title: %s\n", created.Title)
		if len(created.Reminders) > 0 {
			for _, r := range created.Reminders {
				fmt.Printf("  🔔 Reminder: %s\n", api.ReminderToHuman(r.Trigger))
			}
		}
		return nil
	},
}

var taskGetCmd = &cobra.Command{
	Use:   "get [task-id]",
	Short: "Show task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		taskID := args[0]

		// Find the task across all projects
		tasks, err := client.GetAllTasks()
		if err != nil {
			return err
		}

		var task *api.Task
		var projectID string
		for _, t := range tasks {
			if t.ID == taskID {
				task = &t
				projectID = t.ProjectID
				break
			}
		}

		if task == nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if jsonFlag {
			return format.OutputJSON(task)
		}

		return format.OutputTaskDetail(task, projectID, client)
	},
}

var taskDoneCmd = &cobra.Command{
	Use:   "done [task-id]",
	Short: "Mark a task as complete",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		taskID := args[0]

		// Find the task to get project ID
		tasks, err := client.GetAllTasks()
		if err != nil {
			return err
		}

		var projectID string
		for _, t := range tasks {
			if t.ID == taskID {
				projectID = t.ProjectID
				break
			}
		}

		if projectID == "" {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if err := client.CompleteTask(projectID, taskID); err != nil {
			return err
		}

		fmt.Println("✓ Task marked as complete!")
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete [task-id]",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		taskID := args[0]

		// Find the task to get project ID
		tasks, err := client.GetAllTasks()
		if err != nil {
			return err
		}

		var projectID string
		for _, t := range tasks {
			if t.ID == taskID {
				projectID = t.ProjectID
				break
			}
		}

		if projectID == "" {
			return fmt.Errorf("task not found: %s", taskID)
		}

		if err := client.DeleteTask(projectID, taskID); err != nil {
			return err
		}

		fmt.Println("✓ Task deleted!")
		return nil
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit [task-id]",
	Short: "Edit a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		taskID := args[0]
		title, _ := cmd.Flags().GetString("title")
		dueStr, _ := cmd.Flags().GetString("due")
		priorityStr, _ := cmd.Flags().GetString("priority")
		tagsStr, _ := cmd.Flags().GetString("tag")
		remindStr, _ := cmd.Flags().GetString("remind")

		// Find the task
		tasks, err := client.GetAllTasks()
		if err != nil {
			return err
		}

		var task *api.Task
		for i := range tasks {
			if tasks[i].ID == taskID {
				task = &tasks[i]
				break
			}
		}

		if task == nil {
			return fmt.Errorf("task not found: %s", taskID)
		}

		// Update fields
		if title != "" {
			task.Title = title
		}
		if dueStr != "" {
			dueDate, err := api.ParseDueDate(dueStr, cfg.Timezone)
			if err != nil {
				return err
			}
			task.DueDate = dueDate
		}
		if priorityStr != "" {
			task.Priority = api.ParsePriority(priorityStr)
		}
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
			task.Tags = tags
		}
		if remindStr != "" {
			reminders, err := api.ParseReminders(remindStr)
			if err != nil {
				return fmt.Errorf("failed to parse reminders: %w", err)
			}
			task.Reminders = reminders
		}

		_, err = client.UpdateTask(task)
		if err != nil {
			return err
		}

		fmt.Println("✓ Task updated!")
		return nil
	},
}

// Helper functions for filtering
func filterByDue(tasks []api.Task, filter string) []api.Task {
	var filtered []api.Task
	for _, t := range tasks {
		dueStr := api.FormatDueDate(t.DueDate)
		switch filter {
		case "today":
			if dueStr == "today" {
				filtered = append(filtered, t)
			}
		case "overdue":
			if dueStr == "overdue" {
				filtered = append(filtered, t)
			}
		case "tomorrow":
			if dueStr == "tomorrow" {
				filtered = append(filtered, t)
			}
		}
	}
	return filtered
}

func filterByPriority(tasks []api.Task, filter string) []api.Task {
	filterPriority := api.ParsePriority(filter)
	var filtered []api.Task
	for _, t := range tasks {
		if t.Priority == filterPriority {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func filterByTag(tasks []api.Task, filter string) []api.Task {
	var filtered []api.Task
	for _, t := range tasks {
		for _, tag := range t.Tags {
			if strings.EqualFold(tag, filter) {
				filtered = append(filtered, t)
				break
			}
		}
	}
	return filtered
}
