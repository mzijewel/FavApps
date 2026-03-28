package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var AppsFilePath = ".favapps/data.json"

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".favapps_config.json"
	}
	return filepath.Join(home, ".favapps_config.json")
}

func LoadConfig() (Config, error) {
	configPath := getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Config{AppsFile: AppsFilePath}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{AppsFile: AppsFilePath}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{AppsFile: AppsFilePath}, err
	}

	if config.AppsFile == "" {
		config.AppsFile = AppsFilePath
	}
	AppsFilePath = config.AppsFile
	return config, nil
}

func SaveConfig(config Config) error {
	configPath := getConfigPath()
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func LoadApps() ([]App, error) {
	if _, err := os.Stat(AppsFilePath); os.IsNotExist(err) {
		return []App{}, nil
	}

	data, err := os.ReadFile(AppsFilePath)
	if err != nil {
		return nil, err
	}

	var apps []App
	err = json.Unmarshal(data, &apps)
	if err != nil {
		return nil, err
	}

	return apps, nil
}

func SaveApps(apps []App) error {
	dir := filepath.Dir(AppsFilePath)
	if dir != "." {
		os.MkdirAll(dir, 0755)
	}

	var buf strings.Builder
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(apps); err != nil {
		return err
	}

	return os.WriteFile(AppsFilePath, []byte(buf.String()), 0o644)
}
