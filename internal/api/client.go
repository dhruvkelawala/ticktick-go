package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"ticktick-go/internal/auth"
	"ticktick-go/internal/config"
)

const baseURL = "https://api.ticktick.com/open/v1"

type Client struct {
	httpClient *http.Client
	config     *config.Config
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		httpClient: &http.Client{},
		config:     cfg,
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	token, err := auth.GetValidToken(c.config)
	if err != nil {
		return nil, err
	}

	var reqBody io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusUnauthorized {
		// Try to refresh token
		token, refreshErr := auth.RefreshToken(c.config)
		if refreshErr != nil {
			return nil, fmt.Errorf("authentication failed: %w", refreshErr)
		}

		// Retry with new token
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)
		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		data, _ = io.ReadAll(resp.Body)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API request failed (status %d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}
