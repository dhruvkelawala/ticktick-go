package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ticktick-go/internal/config"
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
	return filepath.Join(home, ".config", "ttg", "token.json")
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
	dir := filepath.Join(home, ".config", "ttg")
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

	if current.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available. Please run 'tt auth login'")
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

// OAuthLogin performs the OAuth2 authorization code flow with callback server
func OAuthLogin(cfg *config.Config) error {
	// Create channels
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start callback server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			codeChan <- code
			w.Write([]byte(`<html><head><meta name="viewport" content="width=device-width,initial-scale=1"><style>body{font-family:-apple-system,BlinkMacSystemFont,sans-serif;padding:40px;text-align:center;background:#1a1a2e;color:#eee}h1{color:#4ade80}code{background:#333;padding:8px 16px;border-radius:4px;display:inline-block;margin:20px 0;font-size:14px}</style></head><body><h1>✓ Authentication Successful!</h1><p>You can close this window and return to the terminal.</p><p>Code received: <code>` + code + `</code></p></body></html>`))
		} else {
			errChan <- fmt.Errorf("no code received in callback")
			w.Write([]byte(`<html><head><style>body{font-family:sans-serif;padding:40px;text-align:center;background:#1a1a2e;color:#eee}h1{color:#f87171}</style></head><body><h1>✗ Authentication Failed</h1></body></html>`))
		}
	})

	server := &http.Server{Addr: "localhost:18900", Handler: mux}
	
	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Build authorization URL
	loginURL := fmt.Sprintf(
		"%s?client_id=%s&scope=tasks:read%%20tasks:write&response_type=code&redirect_uri=%s",
		authURL,
		cfg.ClientID,
		url.QueryEscape(callbackURL),
	)

	fmt.Println("Opening browser for authentication...")
	fmt.Println("If browser doesn't open, visit this URL manually:")
	fmt.Println(loginURL)
	fmt.Println()
	fmt.Println("Waiting for callback from TickTick...")

	// Try to open browser
	_ = openBrowser(loginURL)

	// Wait for code from callback
	select {
	case code := <-codeChan:
		fmt.Println("\n✓ Received authorization code!")
		server.Shutdown(context.Background())
		return exchangeCodeForToken(code, cfg)
	case err := <-errChan:
		server.Shutdown(context.Background())
		return err
	case <-time.After(120 * time.Second):
		server.Shutdown(context.Background())
		return fmt.Errorf("authentication timed out - please try again")
	}
}

func openBrowser(url string) error {
	cmd := exec.Command("open", url)
	return cmd.Start()
}

func exchangeCodeForToken(code string, cfg *config.Config) error {
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

	fmt.Println("✓ Authentication successful! Token saved.")
	return nil
}

// GetValidToken returns a valid token, trying to use existing token first
func GetValidToken(cfg *config.Config) (*Token, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, fmt.Errorf("not authenticated. Run 'tt auth login' first")
	}

	// If token is expired and we have a refresh token, try to refresh
	if token.IsExpired() && token.RefreshToken != "" {
		token, err = RefreshToken(cfg)
		if err != nil {
			// If refresh fails but we have an access token, try using it anyway
			// (the expiry might be wrong but the token still works)
			if token.AccessToken != "" {
				return token, nil
			}
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}
	}

	// If not expired, or no refresh token but we have access token, return what we have
	return token, nil
}
