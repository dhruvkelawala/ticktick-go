package cmd

import (
	"github.com/spf13/cobra"
	"ticktick-go/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "ttg",
	Short: "TickTick CLI - Manage your tasks from the terminal",
	Long:  `A CLI tool for managing TickTick tasks with OAuth2 authentication.`,
}

var (
	jsonFlag bool
	cfg      *config.Config
)

func Execute() error {
	cfg = config.Load()
	return rootCmd.Execute()
}

func addGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&jsonFlag, "json", "j", false, "Output in JSON format")
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(taskCmd)
	rootCmd.AddCommand(projectCmd)
}
