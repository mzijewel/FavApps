package app

import (
	"encoding/json"
	"os"
)

const appsFile = "apps.json"

func LoadApps() ([]App, error) {
	if _, err := os.Stat(appsFile); os.IsNotExist(err) {
		return []App{}, nil
	}

	data, err := os.ReadFile(appsFile)
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

	return os.WriteFile(appsFile, data, 0644)
}
