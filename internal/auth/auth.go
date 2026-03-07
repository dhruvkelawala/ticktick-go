package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tt/internal/config"
)

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

const (
	authURL     = "https://ticktick.com/oauth/authorize"
	tokenURL    = "https://ticktick.com/oauth/token"
	callbackURL = "http://localhost:18900/callback"
)

func TokenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tt", "token.json")
}

func LoadToken() (*Token, error) {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func SaveToken(token *Token) error {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "tt")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(TokenPath(), data, 0600)
}

func DeleteToken() error {
	return os.Remove(TokenPath())
}

func (t *Token) IsExpired() bool {
	if t.ExpiresAt == 0 {
		return false
	}
	return time.Now().Unix() >= t.ExpiresAt
}

func (t *Token) IsValid() bool {
	return t.AccessToken != ""
}

// RefreshToken refreshes the access token using the refresh token
func RefreshToken(cfg *config.Config) (*Token, error) {
	current, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("no token to refresh: %w", err)
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", current.RefreshToken)
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)

	req, err := http.NewRequest("POST", tokenURL, 
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokenResp Token
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, err
	}

	// Set expiry to 30 minutes from now (TickTick tokens typically last 30 min)
	tokenResp.ExpiresAt = time.Now().Add(30 * time.Minute).Unix()

	// Preserve refresh token if not in response
	if tokenResp.RefreshToken == "" {
		tokenResp.RefreshToken = current.RefreshToken
	}

	if err := SaveToken(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// OAuthLogin performs the OAuth2 authorization code flow
func OAuthLogin(cfg *config.Config) error {
	// Build authorization URL
	authURL := fmt.Sprintf(
		"%s?client_id=%s&scope=tasks:read%%20tasks:write&response_type=code&redirect_uri=%s",
		authURL,
		cfg.ClientID,
		url.QueryEscape(callbackURL),
	)

	fmt.Println("Please open this URL in your browser:")
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("After authorizing, you will be redirected to a localhost page.")
	fmt.Println("Paste the full redirect URL (or just the code) below.")
	fmt.Print("\nAuthorization code: ")

	var input string
	fmt.Scanln(&input)

	// Extract code from full URL if user pasted redirect URL
	code := input
	if strings.Contains(input, "code=") {
		u, err := url.Parse(input)
		if err == nil {
			code = u.Query().Get("code")
		}
	}

	if code == "" {
		return fmt.Errorf("authorization code is required")
	}

	// Exchange code for token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", cfg.ClientID)
	data.Set("client_secret", cfg.ClientSecret)
	data.Set("redirect_uri", callbackURL)

	req, err := http.NewRequest("POST", tokenURL, 
		bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp Token
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	// Set expiry to 30 minutes from now
	tokenResp.ExpiresAt = time.Now().Add(30 * time.Minute).Unix()

	if err := SaveToken(&tokenResp); err != nil {
		return err
	}

	fmt.Println("\n✓ Authentication successful! Token saved.")
	return nil
}

// GetValidToken returns a valid token, refreshing if necessary
func GetValidToken(cfg *config.Config) (*Token, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated. Run 'tt auth login' first")
	}

	if token.IsExpired() {
		token, err = RefreshToken(cfg)
		if err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
	}

	return token, nil
}
