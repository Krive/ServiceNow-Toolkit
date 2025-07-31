package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/Krive/ServiceNow-Toolkit/pkg/servicenow"
)

var explorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Launch interactive ServiceNow explorer",
	RunE: func(cmd *cobra.Command, args []string) error {
		var client *servicenow.Client
		var err error

		if demoMode {
			client = nil             // Demo mode
			resolvedInstanceURL = "" // Clear for demo mode
		} else {
			// Capture the resolved instance URL before creating client
			resolvedInstanceURL = getCredentialLocal(instanceURL, "SERVICENOW_INSTANCE_URL")

			client, err = createClient()
			if err != nil {
				return fmt.Errorf("failed to create ServiceNow client: %w", err)
			}
		}

		model := newSimpleExplorer(client)
		program := tea.NewProgram(model, tea.WithAltScreen())

		_, err = program.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(explorerCmd)
	explorerCmd.Flags().BoolVar(&demoMode, "demo", false, "Run in demo mode")
}
