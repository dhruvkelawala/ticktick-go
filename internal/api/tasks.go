package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"ticktick-go/internal/config"
	"ticktick-go/internal/dateparse"
)

type ChecklistItem struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Status        int    `json:"status"` // 0=open, 2=done
	CompletedTime string `json:"completedTime,omitempty"`
	SortOrder     int64  `json:"sortOrder,omitempty"`
}

type Task struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"projectId"`
	Title          string          `json:"title"`
	Content        string          `json:"content,omitempty"`
	Priority       int             `json:"priority"` // 0=None, 1=Low, 2=Medium, 3=High
	DueDate        string          `json:"dueDate,omitempty"`
	StartDate      string          `json:"startDate,omitempty"`
	Repeat         string          `json:"repeat,omitempty"`
	RepeatFlag     string          `json:"repeatFlag,omitempty"`
	IsAllDay       bool            `json:"isAllDay"`
	Tags           []string        `json:"tags,omitempty"`
	Status         int             `json:"status"` // 0=Incomplete, 2=Complete
	CompletedTime  string          `json:"completedTime,omitempty"`
	CreatedTime    string          `json:"createdTime,omitempty"`
	ModifiedTime   string          `json:"modifiedTime,omitempty"`
	Reminders      []Reminder      `json:"reminders,omitempty"`
	Items          []ChecklistItem `json:"items,omitempty"`
	Kind           string          `json:"kind,omitempty"`
	Progress       int             `json:"progress,omitempty"` // 0-100 for checklist tasks
}

type Reminder struct {
	Trigger string `json:"trigger"`
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

// GetChecklistItems returns checklist items for a task
func (c *Client) GetChecklistItems(projectID, taskID string) ([]ChecklistItem, error) {
	task, err := c.GetTask(projectID, taskID)
	if err != nil {
		return nil, err
	}

	return task.Items, nil
}

// AddChecklistItem adds a new checklist item to a task
func (c *Client) AddChecklistItem(projectID, taskID string, title string) (*ChecklistItem, error) {
	// Get the current task to get existing items
	task, err := c.GetTask(projectID, taskID)
	if err != nil {
		return nil, err
	}

	// Create new item with a sort order
	newItem := ChecklistItem{
		Title:     title,
		Status:    0,
		SortOrder: int64(len(task.Items) * 1000),
	}

	// Add to items array
	task.Items = append(task.Items, newItem)

	// Update the task with new items
	updated, err := c.UpdateTask(task)
	if err != nil {
		return nil, err
	}

	// Return the newly added item
	if len(updated.Items) > 0 {
		return &updated.Items[len(updated.Items)-1], nil
	}

	return nil, fmt.Errorf("failed to add checklist item")
}

// UpdateChecklistItem updates a checklist item
func (c *Client) UpdateChecklistItem(projectID, taskID string, item *ChecklistItem) (*ChecklistItem, error) {
	// Get the current task
	task, err := c.GetTask(projectID, taskID)
	if err != nil {
		return nil, err
	}

	// Find and update the item
	found := false
	for i, existingItem := range task.Items {
		if existingItem.ID == item.ID {
			task.Items[i] = *item
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("item not found: %s", item.ID)
	}

	// Update the task
	_, err = c.UpdateTask(task)
	if err != nil {
		return nil, err
	}

	// Return the updated item (use the one we sent since IDs may change)
	return item, nil
}

// DeleteChecklistItem deletes a checklist item
func (c *Client) DeleteChecklistItem(projectID, taskID, itemID string) error {
	// Get the current task
	task, err := c.GetTask(projectID, taskID)
	if err != nil {
		return err
	}

	// Find and remove the item
	found := false
	var newItems []ChecklistItem
	for _, existingItem := range task.Items {
		if existingItem.ID != itemID {
			newItems = append(newItems, existingItem)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("item not found: %s", itemID)
	}

	task.Items = newItems

	// Update the task
	_, err = c.UpdateTask(task)
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

// ChecklistItemStatusToString converts checklist item status to readable string
func ChecklistItemStatusToString(s int) string {
	if s == 2 {
		return "Completed"
	}
	return "Open"
}

// ParseReminders parses comma-separated reminder strings into Reminder structs
// Supported: on-time, 0, 5m, 15m, 30m, 1h, 2h, 1d, 2d, Nm, Nh, Nd
func ParseReminders(s string) ([]Reminder, error) {
	if s == "" {
		return nil, nil
	}

	parts := strings.Split(s, ",")
	var reminders []Reminder

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		trigger, err := parseReminderString(part)
		if err != nil {
			return nil, err
		}
		reminders = append(reminders, Reminder{Trigger: trigger})
	}

	return reminders, nil
}

// parseReminderString converts a single reminder string to TRIGGER format
func parseReminderString(s string) (string, error) {
	// Normalize
	lower := strings.ToLower(s)

	// on-time or 0
	if lower == "on-time" || lower == "0" {
		return "TRIGGER:PT0S", nil
	}

	// Pattern matching for Nm, Nh, Nd
	if len(s) >= 2 {
		n := s[:len(s)-1]
		unit := s[len(s)-1]

		num, err := strconv.Atoi(n)
		if err == nil && num > 0 {
			switch unit {
			case 'm', 'M':
				return fmt.Sprintf("TRIGGER:-PT%dM", num), nil
			case 'h', 'H':
				return fmt.Sprintf("TRIGGER:-PT%dH", num), nil
			case 'd', 'D':
				return fmt.Sprintf("TRIGGER:-P%dD", num), nil
			}
		}
	}

	// Known shortcuts
	switch lower {
	case "5m":
		return "TRIGGER:-PT5M", nil
	case "10m":
		return "TRIGGER:-PT10M", nil
	case "15m":
		return "TRIGGER:-PT15M", nil
	case "30m":
		return "TRIGGER:-PT30M", nil
	case "45m":
		return "TRIGGER:-PT45M", nil
	case "1h":
		return "TRIGGER:-PT1H", nil
	case "2h":
		return "TRIGGER:-PT2H", nil
	case "3h":
		return "TRIGGER:-PT3H", nil
	case "6h":
		return "TRIGGER:-PT6H", nil
	case "12h":
		return "TRIGGER:-PT12H", nil
	case "1d":
		return "TRIGGER:-P1D", nil
	case "2d":
		return "TRIGGER:-P2D", nil
	case "3d":
		return "TRIGGER:-P3D", nil
	case "1w":
		return "TRIGGER:-P1W", nil
	}

	return "", fmt.Errorf("unknown reminder format: %s", s)
}

// ReminderToHuman converts TRIGGER format to human-readable string
func ReminderToHuman(trigger string) string {
	if trigger == "" {
		return ""
	}

	// Remove TRIGGER: prefix
	if strings.HasPrefix(trigger, "TRIGGER:") {
		trigger = trigger[8:]
	}

	// PT0S = at due time
	if trigger == "PT0S" {
		return "at due time"
	}

	// Negative duration (before due)
	if strings.HasPrefix(trigger, "-") {
		trigger = trigger[1:]

		// Parse duration
		if strings.HasPrefix(trigger, "PT") {
			// Time duration: PT1H, PT30M, etc.
			duration := trigger[2:]
			switch duration {
			case "5M":
				return "5 min before"
			case "10M":
				return "10 min before"
			case "15M":
				return "15 min before"
			case "30M":
				return "30 min before"
			case "45M":
				return "45 min before"
			case "1H":
				return "1 hour before"
			case "2H":
				return "2 hours before"
			case "3H":
				return "3 hours before"
			case "6H":
				return "6 hours before"
			case "12H":
				return "12 hours before"
			}
			// Dynamic parse Nm
			if len(duration) >= 2 && duration[len(duration)-1] == 'M' {
				if n, err := strconv.Atoi(duration[:len(duration)-1]); err == nil {
					return fmt.Sprintf("%d min before", n)
				}
			}
			if len(duration) >= 2 && duration[len(duration)-1] == 'H' {
				if n, err := strconv.Atoi(duration[:len(duration)-1]); err == nil {
					return fmt.Sprintf("%d hour%s before", n, plural(n))
				}
			}
		} else if strings.HasPrefix(trigger, "P") {
			// Date duration: P1D, P2D, etc.
			duration := trigger[1:]
			switch duration {
			case "1D":
				return "1 day before"
			case "2D":
				return "2 days before"
			case "3D":
				return "3 days before"
			case "1W":
				return "1 week before"
			}
			if len(duration) >= 2 && duration[len(duration)-1] == 'D' {
				if n, err := strconv.Atoi(duration[:len(duration)-1]); err == nil {
					return fmt.Sprintf("%d days before", n)
				}
			}
			if len(duration) >= 2 && duration[len(duration)-1] == 'W' {
				if n, err := strconv.Atoi(duration[:len(duration)-1]); err == nil {
					return fmt.Sprintf("%d week%s before", n, plural(n))
				}
			}
		}
	}

	return trigger
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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

// ParseRepeat parses repeat string to TickTick RRULE format
// Supports: daily, weekly, monthly, yearly, or raw RRULE strings
func ParseRepeat(repeat string) (string, error) {
	if repeat == "" {
		return "", nil
	}

	lower := strings.ToLower(repeat)

	// If it already looks like an RRULE, return as-is
	if strings.HasPrefix(lower, "rrule:") {
		return strings.ToUpper(repeat), nil
	}

	// Parse shorthand values
	switch lower {
	case "daily", "every day", "1d":
		return "RRULE:FREQ=DAILY;INTERVAL=1", nil
	case "weekly", "every week", "1w":
		return "RRULE:FREQ=WEEKLY;INTERVAL=1", nil
	case "monthly", "every month", "1m":
		return "RRULE:FREQ=MONTHLY;INTERVAL=1", nil
	case "yearly", "every year", "1y":
		return "RRULE:FREQ=YEARLY;INTERVAL=1", nil
	}

	// Try to parse "every X days/weeks/months"
	// e.g., "every 3 days" -> RRULE:FREQ=DAILY;INTERVAL=3
	everyRe := regexp.MustCompile(`^every (\d+) (day|week|month|year)s?$`)
	matches := everyRe.FindStringSubmatch(lower)
	if len(matches) == 3 {
		amount := 1
		fmt.Sscanf(matches[1], "%d", &amount)

		switch matches[2] {
		case "day":
			return fmt.Sprintf("RRULE:FREQ=DAILY;INTERVAL=%d", amount), nil
		case "week":
			return fmt.Sprintf("RRULE:FREQ=WEEKLY;INTERVAL=%d", amount), nil
		case "month":
			return fmt.Sprintf("RRULE:FREQ=MONTHLY;INTERVAL=%d", amount), nil
		case "year":
			return fmt.Sprintf("RRULE:FREQ=YEARLY;INTERVAL=%d", amount), nil
		}
	}

	// Try to parse "every Mon/Tue/Wed/Thu/Fri/Sat/Sun"
	weekdayRe := regexp.MustCompile(`^every (mon|tue|wed|thu|fri|sat|sun)$`)
	weekdayMatches := weekdayRe.FindStringSubmatch(lower)
	if len(weekdayMatches) == 2 {
		dayMap := map[string]string{
			"mon": "MO", "tue": "TU", "wed": "WE",
			"thu": "TH", "fri": "FR", "sat": "SA", "sun": "SU",
		}
		if day, ok := dayMap[weekdayMatches[1]]; ok {
			return fmt.Sprintf("RRULE:FREQ=WEEKLY;BYDAY=%s", day), nil
		}
	}

	// Try to parse weekdays (Mon-Fri)
	if lower == "weekdays" || lower == "weekday" {
		return "RRULE:FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", nil
	}

	// Try to parse weekends
	if lower == "weekends" || lower == "weekend" {
		return "RRULE:FREQ=WEEKLY;BYDAY=SA,SU", nil
	}

	return "", fmt.Errorf("unknown repeat format: %s", repeat)
}

// RepeatToHuman converts RRULE format to human-readable string
func RepeatToHuman(repeat string) string {
	if repeat == "" {
		return ""
	}

	// Remove RRULE: prefix
	if strings.HasPrefix(strings.ToUpper(repeat), "RRULE:") {
		repeat = repeat[6:]
	}

	// Parse common patterns
	lower := strings.ToLower(repeat)

	if strings.Contains(lower, "freq=daily") {
		// Check for interval
		if strings.Contains(lower, "interval=1") || !strings.Contains(lower, "interval=") {
			return "Daily"
		}
		// Try to extract interval
		intervalRe := regexp.MustCompile(`interval=(\d+)`)
		matches := intervalRe.FindStringSubmatch(lower)
		if len(matches) == 2 {
			return fmt.Sprintf("Every %s days", matches[1])
		}
		return "Daily"
	}

	if strings.Contains(lower, "freq=weekly") {
		if byday := extractByDay(lower); byday != "" {
			return fmt.Sprintf("Weekly on %s", byday)
		}
		if strings.Contains(lower, "interval=1") || !strings.Contains(lower, "interval=") {
			return "Weekly"
		}
		intervalRe := regexp.MustCompile(`interval=(\d+)`)
		matches := intervalRe.FindStringSubmatch(lower)
		if len(matches) == 2 {
			return fmt.Sprintf("Every %s weeks", matches[1])
		}
		return "Weekly"
	}

	if strings.Contains(lower, "freq=monthly") {
		if strings.Contains(lower, "interval=1") || !strings.Contains(lower, "interval=") {
			return "Monthly"
		}
		intervalRe := regexp.MustCompile(`interval=(\d+)`)
		matches := intervalRe.FindStringSubmatch(lower)
		if len(matches) == 2 {
			return fmt.Sprintf("Every %s months", matches[1])
		}
		return "Monthly"
	}

	if strings.Contains(lower, "freq=yearly") {
		return "Yearly"
	}

	return repeat
}

func extractByDay(rrule string) string {
	bydayRe := regexp.MustCompile(`byday=([A-Z,]+)`)
	matches := bydayRe.FindStringSubmatch(rrule)
	if len(matches) == 2 {
		dayMap := map[string]string{
			"MO": "Monday", "TU": "Tuesday", "WE": "Wednesday",
			"TH": "Thursday", "FR": "Friday", "SA": "Saturday", "SU": "Sunday",
		}
		days := strings.Split(matches[1], ",")
		var result []string
		for _, d := range days {
			if name, ok := dayMap[d]; ok {
				result = append(result, name)
			}
		}
		if len(result) > 0 {
			return strings.Join(result, ", ")
		}
	}
	return ""
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
