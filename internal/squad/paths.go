package squad

import "path/filepath"

func SquadDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".squad")
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
