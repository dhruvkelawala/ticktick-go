package format

import (
	"encoding/json"
	"fmt"
	"strings"

	"tt/internal/api"
)

// OutputJSON outputs data as JSON
func OutputJSON(data interface{}) error {
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(output))
	return nil
}

// OutputTaskList outputs a list of tasks in a table format
func OutputTaskList(tasks []api.Task, client *api.Client) error {
	if len(tasks) == 0 {
		fmt.Println("No tasks found.")
		return nil
	}

	fmt.Println()
	for _, t := range tasks {
		projectName := client.GetProjectName(t.ProjectID)
		dueStr := api.FormatDueDate(t.DueDate)

		// Format priority (TickTick: 5=high, 3=medium, 1=low)
		priorityStr := "[ - ]"
		if t.Priority == 5 {
			priorityStr = "[HIGH]"
		} else if t.Priority == 3 {
			priorityStr = "[MED]"
		} else if t.Priority == 1 {
			priorityStr = "[LOW]"
		}

		// Format status prefix
		statusPrefix := " "
		if t.Status == 2 {
			statusPrefix = "✓"
		}

		fmt.Printf("%s %s  %-30s → %-15s due: %s\n", 
			statusPrefix,
			priorityStr, 
			truncate(t.Title, 30), 
			truncate(projectName, 15),
			dueStr)
	}
	fmt.Println()

	return nil
}

// OutputTaskDetail outputs a single task with full details
func OutputTaskDetail(task *api.Task, projectID string, client *api.Client) error {
	projectName := client.GetProjectName(projectID)
	priority := api.PriorityToString(task.Priority)
	status := api.StatusToString(task.Status)

	fmt.Println()
	fmt.Println("╭─ Task " + task.ID + strings.Repeat("─", 40-len(task.ID)) + "╮")
	fmt.Println("│ Title:    " + task.Title)
	fmt.Println("│ Project:  " + projectName)
	fmt.Println("│ Priority: " + priority)
	
	if task.DueDate != "" {
		fmt.Println("│ Due:      " + formatDueDateFull(task.DueDate))
	} else {
		fmt.Println("│ Due:      no due date")
	}
	
	if len(task.Tags) > 0 {
		fmt.Println("│ Tags:     " + strings.Join(task.Tags, ", "))
	} else {
		fmt.Println("│ Tags:     none")
	}
	
	fmt.Println("│ Status:   " + status)
	
	if task.Content != "" {
		fmt.Println("│ Notes:    " + task.Content)
	}
	
	fmt.Println("╰" + strings.Repeat("─", 50) + "╯")
	fmt.Println()

	return nil
}

// OutputProjectList outputs a list of projects
func OutputProjectList(projects []api.Project, client *api.Client) error {
	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return nil
	}

	fmt.Println()
	for _, p := range projects {
		inbox := ""
		if p.Inbox {
			inbox = " [INBOX]"
		}
		fmt.Printf("  %-30s%s\n", truncate(p.Name, 30), inbox)
	}
	fmt.Println()

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatDueDateFull(due string) string {
	tm := api.ToLocalTime(due)
	if tm.IsZero() {
		return due
	}
	return tm.Format("Mon, Jan 2, 2006 at 3:04 PM")
}
