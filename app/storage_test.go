package app

import (
	"os"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	testApps := []App{
		{Name: "TestApp 1", Description: "Description 1"},
		{Name: "TestApp 2", Description: "Description 2"},
	}

	err := SaveApps(testApps)
	if err != nil {
		t.Fatalf("Failed to save apps: %v", err)
	}

	loadedApps, err := LoadApps()
	if err != nil {
		t.Fatalf("Failed to load apps: %v", err)
	}

	if len(loadedApps) != len(testApps) {
		t.Fatalf("Expected %d apps, got %d", len(testApps), len(loadedApps))
	}

	for i, a := range loadedApps {
		if a.Name != testApps[i].Name || a.Description != testApps[i].Description {
			t.Errorf("App %d: expected %+v, got %+v", i, testApps[i], a)
		}
	}

	// Cleanup
	os.Remove("apps.json")
}
