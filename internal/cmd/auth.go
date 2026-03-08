package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"tt/internal/auth"
	"tt/internal/config"
)

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to TickTick via OAuth2",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		return auth.OAuthLogin(cfg)
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := auth.LoadToken()
		if err != nil {
			fmt.Println("Not authenticated. Run 'tt auth login' first.")
			return nil
		}

		fmt.Println("✓ Authenticated")

		if token.ExpiresAt > 0 {
			expiresAt := time.Unix(token.ExpiresAt, 0)
			fmt.Printf("Token expires at: %s\n", expiresAt.Format("2006-01-02 15:04:05"))

			if token.IsExpired() {
				fmt.Println("⚠ Token is expired. Run 'tt auth login' to refresh.")
			} else {
				remaining := time.Until(expiresAt)
				fmt.Printf("Time remaining: %v\n", remaining.Round(time.Minute))
			}
		}

		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and delete stored token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteToken(); err != nil {
			return fmt.Errorf("failed to delete token: %w", err)
		}
		fmt.Println("✓ Logged out successfully.")
		return nil
	},
}
