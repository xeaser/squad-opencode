package squad

import (
	"os"
	"path/filepath"
)

func SquadDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".squad")
}

// GlobalSquadDir is the personal squad root: $HOME/.squad-oc/global.
func GlobalSquadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".squad-oc", "global"), nil
}

// LinksCacheDir is ~/.squad-oc/links (cloned remote teams).
func LinksCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".squad-oc", "links"), nil
}

func ConfigPath(projectRoot string) string {
	return filepath.Join(SquadDir(projectRoot), "config.json")
}

func TeamPath(projectRoot string) string {
	return filepath.Join(ResolveDir(projectRoot), "team.md")
}

func OpencodeAgentsDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".opencode", "agents")
}

func SquadAgentPath(projectRoot string) string {
	return filepath.Join(OpencodeAgentsDir(projectRoot), "squad.md")
}
