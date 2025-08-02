package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Krive/ServiceNow-Toolkit/internal/app/explorer"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration settings",
	Long: `Manage ServiceNow Toolkit configuration settings.

This command allows you to view and manage your saved view configurations,
including column layouts, filters, and other user preferences.`,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show the path to the configuration file",
	Long: `Show the path to the configuration file where your saved views and settings are stored.

You can copy this file to other computers to transfer your configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		cm := explorer.NewConfigManager()
		configPath := cm.GetConfigPath()
		
		fmt.Printf("Configuration file location:\n%s\n\n", configPath)
		
		// Check if file exists and show size
		if info, err := os.Stat(configPath); err == nil {
			fmt.Printf("File exists: %d bytes\n", info.Size())
		} else {
			fmt.Printf("File does not exist yet (will be created when you save your first view)\n")
		}
		
		fmt.Printf("\nTo transfer your configuration to another computer:\n")
		fmt.Printf("1. Copy this file to the same location on the target computer\n")
		fmt.Printf("2. Ensure the directory exists on the target computer\n")
		fmt.Printf("3. Your saved views and settings will be available immediately\n")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration including saved views and global settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		cm := explorer.NewConfigManager()
		if err := cm.LoadConfig(); err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			return
		}
		
		configs := cm.GetViewConfigurations()
		settings := cm.GetGlobalSettings()
		
		fmt.Printf("Configuration file: %s\n\n", cm.GetConfigPath())
		
		fmt.Printf("Global Settings:\n")
		fmt.Printf("  Default Page Size: %d\n", settings.DefaultPageSize)
		fmt.Printf("  Theme: %s\n", settings.Theme)
		fmt.Printf("  Auto Save: %t\n", settings.AutoSave)
		fmt.Printf("  Export Directory: %s\n\n", settings.ExportDirectory)
		
		if len(configs) == 0 {
			fmt.Printf("No saved view configurations found.\n")
			fmt.Printf("Use the column customizer (press 'c' in table view) and save views (Ctrl+S) to create configurations.\n")
		} else {
			fmt.Printf("Saved View Configurations (%d):\n", len(configs))
			for name, config := range configs {
				fmt.Printf("\n  📋 %s\n", name)
				fmt.Printf("    Table: %s\n", config.TableName)
				fmt.Printf("    Columns: %d (%v)\n", len(config.Columns), config.Columns)
				if config.Query != "" {
					fmt.Printf("    Filter: %s\n", config.Query)
				}
				if config.Description != "" {
					fmt.Printf("    Description: %s\n", config.Description)
				}
				fmt.Printf("    Created: %s\n", config.CreatedAt)
			}
		}
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long:  `Reset all configuration to default values. This will delete all saved views and settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		cm := explorer.NewConfigManager()
		
		// Confirm with user
		fmt.Printf("This will delete all saved view configurations and reset settings to defaults.\n")
		fmt.Printf("Are you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Printf("Reset cancelled.\n")
			return
		}
		
		if err := cm.ResetConfig(); err != nil {
			fmt.Printf("Error resetting configuration: %v\n", err)
			return
		}
		
		fmt.Printf("Configuration reset to defaults.\n")
	},
}

var configBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of the current configuration",
	Long:  `Create a backup of the current configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		cm := explorer.NewConfigManager()
		
		if err := cm.BackupConfig(); err != nil {
			fmt.Printf("Error creating backup: %v\n", err)
			return
		}
		
		fmt.Printf("Configuration backed up to: %s.backup\n", cm.GetConfigPath())
	},
}

var configSetExportDirCmd = &cobra.Command{
	Use:   "set-export-dir [directory]",
	Short: "Set the export directory for exported files",
	Long: `Set the directory where exported CSV and JSON files will be saved.
	
If no directory is provided, the current directory will be used.
The directory will be created if it doesn't exist.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cm := explorer.NewConfigManager()
		
		// Load current config
		if err := cm.LoadConfig(); err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			return
		}
		
		var newDir string
		if len(args) > 0 {
			newDir = args[0]
		} else {
			// If no argument provided, use current directory
			if wd, err := os.Getwd(); err == nil {
				newDir = wd
			} else {
				fmt.Printf("Error getting current directory: %v\n", err)
				return
			}
		}
		
		// Expand ~ to home directory if needed
		if strings.HasPrefix(newDir, "~/") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				newDir = filepath.Join(homeDir, newDir[2:])
			}
		}
		
		// Convert to absolute path
		if absPath, err := filepath.Abs(newDir); err == nil {
			newDir = absPath
		}
		
		// Test if directory exists or can be created
		if err := os.MkdirAll(newDir, 0755); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", newDir, err)
			return
		}
		
		// Update config
		settings := cm.GetGlobalSettings()
		settings.ExportDirectory = newDir
		
		if err := cm.UpdateGlobalSettings(settings); err != nil {
			fmt.Printf("Error updating configuration: %v\n", err)
			return
		}
		
		fmt.Printf("Export directory set to: %s\n", newDir)
		fmt.Printf("All exported files will now be saved to this directory.\n")
	},
}

func init() {
	// Add subcommands to config command
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configResetCmd)
	configCmd.AddCommand(configBackupCmd)
	configCmd.AddCommand(configSetExportDirCmd)
	
	// Add config command to root
	rootCmd.AddCommand(configCmd)
}