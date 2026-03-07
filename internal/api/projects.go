package api

import "encoding/json"

// Project represents a TickTick project
type Project struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Color         string      `json:"color,omitempty"`
	Archived      bool        `json:"archived,omitempty"`
	ParentID      string      `json:"parentId,omitempty"`
	Kind          interface{} `json:"kind,omitempty"` // 0=Personal, 1=Business (can be string or int)
	Share         bool        `json:"share,omitempty"`
	OwnerID       string      `json:"ownerId,omitempty"`
	GroupID       string      `json:"groupId,omitempty"`
	Inbox         bool        `json:"inbox,omitempty"` // True if this is the inbox project
	SortOrder     int         `json:"sortOrder,omitempty"`
	TaskCount     int         `json:"taskCount,omitempty"` // Not from API, computed
}

// GetProjects returns all projects
func (c *Client) GetProjects() ([]Project, error) {
	data, err := c.doRequest("GET", "/project", nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

// GetProject returns a single project by ID
func (c *Client) GetProject(projectID string) (*Project, error) {
	data, err := c.doRequest("GET", "/project/"+projectID, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, err
	}

	return &project, nil
}
