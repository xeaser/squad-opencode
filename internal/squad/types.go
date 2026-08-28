package squad

// OTLP is optional live-export knobs. Missing key = unset. Never store API keys here.
type OTLPConfig struct {
	Endpoint       string `json:"endpoint,omitempty"`
	Protocol       string `json:"protocol,omitempty"`
	CaptureContent *bool  `json:"capture_content,omitempty"`
}

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
	// LinkURL, if set, is the git remote this project linked (remote link).
	LinkURL string `json:"linkUrl,omitempty"`
	// LinkRef is the branch checked out in the links cache (default branch).
	LinkRef string `json:"linkRef,omitempty"`
	// LinkSHA is the last fetched commit of LinkRef.
	LinkSHA string `json:"linkSha,omitempty"`
	// Theme is an optional display-name theme (e.g. "office"). IDs stay stable.
	Theme string `json:"theme,omitempty"`
	// ThemeOrigin is how the theme was set: "init" or "applied".
	ThemeOrigin string `json:"themeOrigin,omitempty"`
	// OTLP is optional live OTLP export settings (env wins at resolve time).
	OTLP *OTLPConfig `json:"otlp,omitempty"`
	// OpenCodeDB is an optional path to opencode.db (OPENCODE_DB wins).
	OpenCodeDB string `json:"opencode_db,omitempty"`
}

// MentionRow is one row in .squad/mentions.md (slugs without @).
type MentionRow struct {
	Role string
	Now  string
	Was  string
}

// TeamMember is a parsed row from .squad/team.md.
type TeamMember struct {
	ID     string
	Name   string
	Role   string
	Status string
	Model  string // raw cell; empty means inherit
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
	// Global writes into GlobalSquadDir() instead of ProjectRoot.
	Global bool
	// Theme is optional (office|none). Office mints native character IDs at birth.
	Theme string
}

// InitResult is the outcome of init.
type InitResult struct {
	AlreadyInitialized bool
	ProjectRoot        string
	FilesWritten       []string
	Message            string
}
