package app

import "runtime"

type App struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CmdMac      string `json:"cmd_mac,omitempty"`
	CmdLinux    string `json:"cmd_linux,omitempty"`
}

type Config struct {
	AppsFile string `json:"apps_file"`
}

// Cmd returns the install command for the OS currently running the app.
// If the command for the current OS is missing, the other one is used.
func (a App) Cmd() string {
	if runtime.GOOS == "linux" {
		if a.CmdLinux != "" {
			return a.CmdLinux
		}
		return a.CmdMac
	}
	if a.CmdMac != "" {
		return a.CmdMac
	}
	return a.CmdLinux
}
