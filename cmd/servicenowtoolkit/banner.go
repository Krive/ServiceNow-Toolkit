package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var bannerCmd = &cobra.Command{
	Use:   "banner",
	Short: "Display ServiceNow Toolkit ASCII banner",
	Long:  `Display the ServiceNow Toolkit ASCII banner and version information.`,
	Run: func(cmd *cobra.Command, args []string) {
		banner := `
 ███████ ███████ ██████  ██    ██ ██  ██████ ███████ ███    ██  ██████  ██     ██
 ██      ██      ██   ██ ██    ██ ██ ██      ██      ████   ██ ██    ██ ██     ██
 ███████ █████   ██████  ██    ██ ██ ██      █████   ██ ██  ██ ██    ██ ██  █  ██
      ██ ██      ██   ██  ██  ██  ██ ██      ██      ██  ██ ██ ██    ██ ██ ███ ██
 ███████ ███████ ██   ██   ████   ██  ██████ ███████ ██   ████  ██████   ███ ███ 

████████  ██████   ██████  ██      ██   ██ ██ ████████ 
   ██    ██    ██ ██    ██ ██      ██  ██  ██    ██    
   ██    ██    ██ ██    ██ ██      █████   ██    ██    
   ██    ██    ██ ██    ██ ██      ██  ██  ██    ██    
   ██     ██████   ██████  ███████ ██   ██ ██    ██    

                 🚀 ServiceNow Toolkit - Interactive CLI & SDK 🚀
                              Version 1.0.0-beta
                      https://github.com/Krive/ServiceNow-Toolkit
`
		fmt.Print(banner)
		fmt.Println("\nFeatures:")
		fmt.Println("  • 📋 Table Browser with pagination and filtering")
		fmt.Println("  • 🏗️ CMDB Explorer for configuration items")
		fmt.Println("  • 👥 Identity Management for users and groups")
		fmt.Println("  • 🔍 Global Search across multiple tables")
		fmt.Println("  • 📊 Analytics and aggregation capabilities")
		fmt.Println("  • 🛒 Service Catalog browser")
		fmt.Println("  • 📄 XML record viewer with proper reference field handling")
		fmt.Println("  • 🚀 Interactive TUI with keyboard navigation")
		fmt.Println("  • 🎭 Demo mode for testing without ServiceNow connection")
		fmt.Println("")
		fmt.Println("Run 'servicenowtoolkit explorer' to start the interactive interface!")
	},
}

func init() {
	rootCmd.AddCommand(bannerCmd)
}
