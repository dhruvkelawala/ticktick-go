package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"ticktick-go/internal/config"
	"ticktick-go/internal/dateparse"
)

type Task struct {
	ID             string     `json:"id"`
	ProjectID      string     `json:"projectId"`
	Title          string     `json:"title"`
	Content        string     `json:"content,omitempty"`
	Priority       int        `json:"priority"` // 0=None, 1=Low, 2=Medium, 3=High
	DueDate        string     `json:"dueDate,omitempty"`
	StartDate      string     `json:"startDate,omitempty"`
	IsAllDay       bool       `json:"isAllDay"`
	Tags           []string   `json:"tags,omitempty"`
	Status         int        `json:"status"` // 0=Incomplete, 2=Complete
	CompletedTime  string     `json:"completedTime,omitempty"`
	CreatedTime    string     `json:"createdTime,omitempty"`
	ModifiedTime   string     `json:"modifiedTime,omitempty"`
}

type TaskListResponse struct {
	Tasks    []Task    `json:"tasks"`
	Projects []Project `json:"projects"`
}

// GetInboxTasks returns tasks from the inbox (default project)
func (c *Client) GetInboxTasks() ([]Task, error) {
	// Get all projects first
	data, err := c.doRequest("GET", "/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	// Find inbox project
	var inboxID string
	for _, p := range projects {
		if p.Inbox {
			inboxID = p.ID
			break
		}
	}

	// Fallback: use first project if no inbox found
	if inboxID == "" && len(projects) > 0 {
		inboxID = projects[0].ID
	}

	if inboxID == "" {
		return nil, fmt.Errorf("no projects found")
	}

	// Get tasks from inbox
	data, err = c.doRequest("GET", "/project/"+inboxID+"/data", nil)
	if err != nil {
		return nil, err
	}

	// Try to unmarshal as array first
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		// Try as object with tasks field
		var resp struct {
			Tasks []Task `json:"tasks"`
		}
		if err2 := json.Unmarshal(data, &resp); err2 == nil {
			return resp.Tasks, nil
		}
		return nil, fmt.Errorf("failed to parse tasks: %v", err)
	}

	return tasks, nil

	return tasks, nil
}

// GetProjectTasks returns tasks from a specific project
func (c *Client) GetProjectTasks(projectID string) ([]Task, error) {
	data, err := c.doRequest("GET", "/project/"+projectID+"/data", nil)
	if err != nil {
		return nil, err
	}

	// Try to unmarshal as array first
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		// Try as object with tasks field
		var resp struct {
			Tasks []Task `json:"tasks"`
		}
		if err2 := json.Unmarshal(data, &resp); err2 == nil {
			return resp.Tasks, nil
		}
		return nil, fmt.Errorf("failed to parse tasks: %v (data: %s)", err, string(data[:min(200, len(data))]))
	}

	return tasks, nil
}

// GetAllTasks returns all tasks across all projects
func (c *Client) GetAllTasks() ([]Task, error) {
	data, err := c.doRequest("GET", "/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	var allTasks []Task
	for _, p := range projects {
		tasks, err := c.GetProjectTasks(p.ID)
		if err != nil {
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}

// GetTask returns a single task by ID
func (c *Client) GetTask(projectID, taskID string) (*Task, error) {
	data, err := c.doRequest("GET", "/project/"+projectID+"/task/"+taskID, nil)
	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// CreateTask creates a new task
func (c *Client) CreateTask(task *Task) (*Task, error) {
	data, err := c.doRequest("POST", "/task", task)
	if err != nil {
		return nil, err
	}

	var created Task
	if err := json.Unmarshal(data, &created); err != nil {
		return nil, err
	}

	return &created, nil
}

// UpdateTask updates an existing task
func (c *Client) UpdateTask(task *Task) (*Task, error) {
	data, err := c.doRequest("POST", "/task/"+task.ID, task)
	if err != nil {
		return nil, err
	}

	var updated Task
	if err := json.Unmarshal(data, &updated); err != nil {
		return nil, err
	}

	return &updated, nil
}

// CompleteTask marks a task as complete
func (c *Client) CompleteTask(projectID, taskID string) error {
	_, err := c.doRequest("POST", "/project/"+projectID+"/task/"+taskID+"/complete", nil)
	return err
}

// DeleteTask deletes a task
func (c *Client) DeleteTask(projectID, taskID string) error {
	_, err := c.doRequest("DELETE", "/project/"+projectID+"/task/"+taskID, nil)
	return err
}

// GetInboxProjectID returns the inbox project ID
func (c *Client) GetInboxProjectID() (string, error) {
	projects, err := c.GetProjects()
	if err != nil {
		return "", err
	}

	for _, p := range projects {
		if p.Inbox {
			return p.ID, nil
		}
	}

	// Fallback: return first project if no inbox found
	if len(projects) > 0 {
		return projects[0].ID, nil
	}

	return "", fmt.Errorf("inbox project not found")
}

// ParsePriority converts string priority to int (TickTick API: 0=none, 1=low, 3=medium, 5=high)
func ParsePriority(p string) int {
	switch p {
	case "high":
		return 5
	case "medium", "med":
		return 3
	case "low":
		return 1
	default:
		return 0
	}
}

// PriorityToString converts int priority to string
func PriorityToString(p int) string {
	switch p {
	case 5:
		return "High"
	case 3:
		return "Medium"
	case 1:
		return "Low"
	default:
		return "-"
	}
}

// StatusToString converts status to readable string
func StatusToString(s int) string {
	if s == 2 {
		return "Completed"
	}
	return "Active"
}

// ParseDueDate parses a due date string to TickTick format
func ParseDueDate(due string, timezone string) (string, error) {
	if due == "" {
		return "", nil
	}

	// Try to parse with natural language
	parsed, err := dateparse.ParseDate(due, timezone)
	if err != nil {
		return "", err
	}

	// Return ISO format with timezone offset (like +0000)
	_, offset := parsed.Zone()
	offsetHours := offset / 3600
	offsetMins := (offset % 3600) / 60
	offsetStr := fmt.Sprintf("%+03d%02d", offsetHours, offsetMins)
	// Handle case where offset is 0 (UTC) - should be +0000 not +00
	if offset == 0 {
		offsetStr = "+0000"
	}
	return parsed.Format("2006-01-02T15:04:05") + offsetStr, nil
}

// ToLocalTime converts TickTick time to local time
func ToLocalTime(t string) time.Time {
	if t == "" {
		return time.Time{}
	}
	// Try parsing the TickTick format
	formats := []string{
		"2006-01-02T15:04:05.000+0000",
		"2006-01-02T15:04:05+0000",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	
	for _, f := range formats {
		if tm, err := time.Parse(f, t); err == nil {
			return tm
		}
	}
	return time.Time{}
}

// GetTaskStatus returns the completion status of a task
func (t *Task) IsCompleted() bool {
	return t.Status == 2
}

// FormatDueDate formats a due date for display
func FormatDueDate(due string) string {
	if due == "" {
		return "no due date"
	}

	tm := ToLocalTime(due)
	if tm.IsZero() {
		return due
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	taskDay := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())

	switch {
	case taskDay.Equal(today):
		return "today"
	case taskDay.Equal(tomorrow):
		return "tomorrow"
	case taskDay.Before(today):
		return "overdue"
	default:
		return tm.Format("Mon")
	}
}

// GetProjectName returns the project name from project ID
func (c *Client) GetProjectName(projectID string) string {
	projects, err := c.GetProjects()
	if err != nil {
		return "Unknown"
	}

	for _, p := range projects {
		if p.ID == projectID {
			return p.Name
		}
	}

	return "Unknown"
}

// GetProjectByID returns a project by ID
func (c *Client) GetProjectByID(id string) (*Project, error) {
	data, err := c.doRequest("GET", "/project/"+id, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func init() {
	// Force use of strconv for task IDs
	_ = strconv.Itoa
}

// GetClient creates a new API client with the given config
func GetClient(cfg *config.Config) *Client {
	return NewClient(cfg)
}
