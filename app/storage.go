package app

import (
	"encoding/json"
	"os"
)

var AppsFile = "apps.json"
const configFile = "config.json"

func LoadConfig() (Config, error) {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return Config{AppsFile: "apps.json"}, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{AppsFile: "apps.json"}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{AppsFile: "apps.json"}, err
	}

	if config.AppsFile == "" {
		config.AppsFile = "apps.json"
	}
	AppsFile = config.AppsFile
	return config, nil
}

func SaveConfig(config Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func LoadApps() ([]App, error) {
	if _, err := os.Stat(AppsFile); os.IsNotExist(err) {
		return []App{}, nil
	}

	data, err := os.ReadFile(AppsFile)
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
	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(AppsFile, data, 0644)
}
