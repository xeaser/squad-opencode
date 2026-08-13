package squad

// Config is stored at .squad/config.json and marks initialization.
type Config struct {
	Version            int    `json:"version"`
	Host               string `json:"host"`
	Preset             string `json:"preset"`
	ProjectDescription string `json:"projectDescription,omitempty"`
	// ExternalPath, if set, is the real .squad directory (externalize).
	ExternalPath string `json:"externalPath,omitempty"`
	// LinkPath, if set, is a remote/shared team directory (link).
	LinkPath string `json:"linkPath,omitempty"`
}

// TeamMember is a parsed row from .squad/team.md.
type TeamMember struct {
	ID     string
	Name   string
	Role   string
	Status string
}

// DetectResult describes whether a project has been initialized.
type DetectResult struct {
	Initialized bool
	ProjectRoot string
	SquadDir    string
	ConfigPath  string
	Config      *Config
}

// InitOptions controls writeDefaultPreset.
type InitOptions struct {
	ProjectRoot        string
	Preset             string
	ProjectDescription string
	Force              bool
}

// InitResult is the outcome of init.
type InitResult struct {
	AlreadyInitialized bool
	ProjectRoot        string
	FilesWritten       []string
	Message            string
}
