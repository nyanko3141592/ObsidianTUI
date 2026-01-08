package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	VaultPath    string `mapstructure:"vault_path"`
	LastOpenFile string `mapstructure:"last_open_file"`
	Theme        string `mapstructure:"theme"`
	EditorMode   string `mapstructure:"editor_mode"`
}

var AppConfig *Config

func Init() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.Getenv("HOME")
	}
	configPath := filepath.Join(configDir, "obsidiantui")

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)

	viper.SetDefault("vault_path", "")
	viper.SetDefault("last_open_file", "")
	viper.SetDefault("theme", "default")
	viper.SetDefault("editor_mode", "normal")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			configFile := filepath.Join(configPath, "config.yaml")
			if err := viper.SafeWriteConfigAs(configFile); err != nil {
				return err
			}
		}
	}

	AppConfig = &Config{}
	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	return nil
}

func Save() error {
	viper.Set("vault_path", AppConfig.VaultPath)
	viper.Set("last_open_file", AppConfig.LastOpenFile)
	viper.Set("theme", AppConfig.Theme)
	viper.Set("editor_mode", AppConfig.EditorMode)
	return viper.WriteConfig()
}

func SetVaultPath(path string) {
	AppConfig.VaultPath = path
}

func GetVaultPath() string {
	return AppConfig.VaultPath
}
