package main

import (
	"github.com/spf13/cobra"

	"github.com/Krive/ServiceNow-Toolkit/internal/app/handlers"
)

var explorerCmd = &cobra.Command{
	Use:   "explorer",
	Short: "Launch interactive ServiceNow explorer",
	RunE: func(cmd *cobra.Command, args []string) error {
		config := handlers.ExplorerConfig{
			InstanceURL:  getCredential(instanceURL, "SERVICENOW_INSTANCE_URL"),
			Username:     getCredential(username, "SERVICENOW_USERNAME"),
			Password:     getCredential(password, "SERVICENOW_PASSWORD"),
			APIKey:       getCredential(apiKey, "SERVICENOW_API_KEY"),
			AuthMethod:   authMethod,
			ClientID:     getCredential(clientID, "SERVICENOW_CLIENT_ID"),
			ClientSecret: getCredential(clientSecret, "SERVICENOW_CLIENT_SECRET"),
			RefreshToken: getCredential(refreshToken, "SERVICENOW_REFRESH_TOKEN"),
			DemoMode:     demoMode,
		}

		return handlers.RunExplorer(cmd.Context(), config)
	},
}

func init() {
	rootCmd.AddCommand(explorerCmd)
	explorerCmd.Flags().BoolVar(&demoMode, "demo", false, "Run in demo mode")
}