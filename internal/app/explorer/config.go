package explorer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// UserConfig represents the user's saved configuration
type UserConfig struct {
	Version          string                        `json:"version"`
	ViewConfigurations map[string]*ViewConfiguration `json:"view_configurations"`
	GlobalSettings   GlobalSettings                `json:"global_settings"`
}

// GlobalSettings represents global user preferences
type GlobalSettings struct {
	DefaultPageSize int    `json:"default_page_size"`
	Theme          string `json:"theme"`
	AutoSave       bool   `json:"auto_save"`
	ExportDirectory string `json:"export_directory"`
}

// ConfigManager handles loading and saving user configuration
type ConfigManager struct {
	configPath string
	config     *UserConfig
}

// NewConfigManager creates a new configuration manager
func NewConfigManager() *ConfigManager {
	configDir := getConfigDir()
	configPath := filepath.Join(configDir, "servicenow-toolkit-config.json")
	
	// Set default export directory to Downloads folder if available, otherwise home directory
	defaultExportDir := getDefaultExportDirectory()
	
	return &ConfigManager{
		configPath: configPath,
		config: &UserConfig{
			Version:            "1.0",
			ViewConfigurations: make(map[string]*ViewConfiguration),
			GlobalSettings: GlobalSettings{
				DefaultPageSize: 20,
				Theme:          "default",
				AutoSave:       true,
				ExportDirectory: defaultExportDir,
			},
		},
	}
}

// getConfigDir returns the appropriate config directory for the OS
func getConfigDir() string {
	// Try XDG_CONFIG_HOME first (Linux/Unix standard)
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "servicenow-toolkit")
	}
	
	// Fall back to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Last resort: current directory
		return ".servicenow-toolkit"
	}
	
	// Platform-specific config directories
	switch {
	case os.Getenv("APPDATA") != "": // Windows
		return filepath.Join(os.Getenv("APPDATA"), "ServiceNowToolkit")
	case filepath.Base(homeDir) != "": // Unix-like systems
		return filepath.Join(homeDir, ".config", "servicenow-toolkit")
	default:
		return filepath.Join(homeDir, ".servicenow-toolkit")
	}
}

// getDefaultExportDirectory returns the appropriate default export directory for the OS
func getDefaultExportDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Last resort: current directory
		return "."
	}
	
	// Platform-specific default export directories
	switch {
	case os.Getenv("APPDATA") != "": // Windows
		// Try Downloads folder first, fall back to Documents, then home
		downloadsDir := filepath.Join(homeDir, "Downloads")
		if _, err := os.Stat(downloadsDir); err == nil {
			return downloadsDir
		}
		documentsDir := filepath.Join(homeDir, "Documents")
		if _, err := os.Stat(documentsDir); err == nil {
			return documentsDir
		}
		return homeDir
	default: // Unix-like systems (macOS, Linux)
		// Try Downloads folder first, fall back to home
		downloadsDir := filepath.Join(homeDir, "Downloads")
		if _, err := os.Stat(downloadsDir); err == nil {
			return downloadsDir
		}
		return homeDir
	}
}

// LoadConfig loads the configuration from disk
func (cm *ConfigManager) LoadConfig() error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Check if config file exists
	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		// Config file doesn't exist, use defaults and save
		return cm.SaveConfig()
	}
	
	// Read config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse JSON
	if err := json.Unmarshal(data, cm.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Ensure view configurations map is initialized
	if cm.config.ViewConfigurations == nil {
		cm.config.ViewConfigurations = make(map[string]*ViewConfiguration)
	}
	
	return nil
}

// SaveConfig saves the configuration to disk
func (cm *ConfigManager) SaveConfig() error {
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to JSON with pretty formatting
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file with appropriate permissions
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetViewConfigurations returns all view configurations
func (cm *ConfigManager) GetViewConfigurations() map[string]*ViewConfiguration {
	return cm.config.ViewConfigurations
}

// SaveViewConfiguration saves a view configuration
func (cm *ConfigManager) SaveViewConfiguration(name string, config *ViewConfiguration) error {
	cm.config.ViewConfigurations[name] = config
	return cm.SaveConfig()
}

// DeleteViewConfiguration deletes a view configuration
func (cm *ConfigManager) DeleteViewConfiguration(name string) error {
	delete(cm.config.ViewConfigurations, name)
	return cm.SaveConfig()
}

// GetGlobalSettings returns global settings
func (cm *ConfigManager) GetGlobalSettings() GlobalSettings {
	return cm.config.GlobalSettings
}

// UpdateGlobalSettings updates global settings
func (cm *ConfigManager) UpdateGlobalSettings(settings GlobalSettings) error {
	cm.config.GlobalSettings = settings
	return cm.SaveConfig()
}

// GetConfigPath returns the path to the config file
func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

// ResetConfig resets configuration to defaults
func (cm *ConfigManager) ResetConfig() error {
	cm.config = &UserConfig{
		Version:            "1.0",
		ViewConfigurations: make(map[string]*ViewConfiguration),
		GlobalSettings: GlobalSettings{
			DefaultPageSize: 20,
			Theme:          "default",
			AutoSave:       true,
		},
	}
	return cm.SaveConfig()
}

// BackupConfig creates a backup of the current configuration
func (cm *ConfigManager) BackupConfig() error {
	backupPath := cm.configPath + ".backup"
	
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config for backup: %w", err)
	}
	
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	
	return nil
}

// RestoreConfig restores configuration from backup
func (cm *ConfigManager) RestoreConfig() error {
	backupPath := cm.configPath + ".backup"
	
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist")
	}
	
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	
	if err := json.Unmarshal(data, cm.config); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}
	
	return cm.SaveConfig()
}