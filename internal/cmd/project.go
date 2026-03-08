package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"ticktick-go/internal/api"
	"ticktick-go/internal/config"
	"ticktick-go/internal/format"
)

func init() {
	projectCmd.AddCommand(projectListCmd, projectGetCmd)
	
	projectListCmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output in JSON format")
	projectGetCmd.Flags().BoolVarP(&jsonFlag, "json", "j", false, "Output in JSON format")
}

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project management commands",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects with task counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		projects, err := client.GetProjects()
		if err != nil {
			return err
		}

		if jsonFlag {
			return format.OutputJSON(projects)
		}

		return format.OutputProjectList(projects, client)
	},
}

var projectGetCmd = &cobra.Command{
	Use:   "get [project-id]",
	Short: "Show project details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		client := api.NewClient(cfg)

		projectID := args[0]

		project, err := client.GetProject(projectID)
		if err != nil {
			return err
		}

		if jsonFlag {
			return format.OutputJSON(project)
		}

		fmt.Printf("Project: %s\n", project.Name)
		fmt.Printf("ID: %s\n", project.ID)
		if project.Color != "" {
			fmt.Printf("Color: %s\n", project.Color)
		}
		if project.Inbox {
			fmt.Println("Inbox: Yes")
		}

		return nil
	},
}
