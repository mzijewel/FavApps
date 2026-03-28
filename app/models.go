package app

type App struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Cmd         string `json:"cmd,omitempty"`
}

type Config struct {
	AppsFile string `json:"apps_file"`
}
