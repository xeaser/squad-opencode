package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/squad-opencode/squad-opencode/internal/opencodeclient"
	"github.com/squad-opencode/squad-opencode/internal/squad"
)

// Check is one doctor row.
type Check struct {
	Name   string
	OK     bool
	Detail string
	// Hard means failure should set non-zero exit when required checks fail.
	Hard bool
}

// RunChecks evaluates project readiness for the E2E walkthrough.
func RunChecks(projectRoot string) []Check {
	var checks []Check

	checks = append(checks, Check{
		Name:   "Go runtime (squad-oc build host)",
		OK:     true,
		Detail: runtime.Version(),
		Hard:   false,
	})

	oc := commandExists("opencode")
	detail := "opencode found"
	if !oc {
		detail = "not found — install from https://opencode.ai/docs/"
	}
	checks = append(checks, Check{
		Name:   "OpenCode on PATH",
		OK:     oc,
		Detail: detail,
		Hard:   true,
	})

	// Soft: OpenCode server via Go SDK / health
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	probe := opencodeclient.ProbeServer(ctx, opencodeclient.DefaultBaseURL)
	checks = append(checks, Check{
		Name:   "OpenCode server (optional)",
		OK:     probe.Reachable,
		Detail: probe.Detail + " — start with: " + opencodeclient.StartHint + " (not the TUI)",
		Hard:   false,
	})

	gitRepo := dirExists(filepath.Join(projectRoot, ".git"))
	gitDetail := ".git present"
	if !gitRepo {
		gitDetail = "this folder is not a git repo (optional but recommended)"
	}
	checks = append(checks, Check{
		Name:   "Git repository",
		OK:     gitRepo,
		Detail: gitDetail,
		Hard:   false,
	})

	init := squad.IsInitialized(projectRoot)
	det := squad.Detect(projectRoot)
	initDetail := "missing — run: squad-oc init --preset default"
	if init {
		preset := "?"
		if det.Config != nil {
			preset = det.Config.Preset
		}
		initDetail = fmt.Sprintf(".squad/config.json (preset: %s)", preset)
	}
	checks = append(checks, Check{
		Name:   "Squad initialized",
		OK:     init,
		Detail: initDetail,
		Hard:   true,
	})

	agent := squad.SquadAgentPath(projectRoot)
	agentOK := fileExists(agent)
	agentDetail := ".opencode/agents/squad.md"
	if !agentOK {
		agentDetail = "missing .opencode/agents/squad.md"
	}
	checks = append(checks, Check{
		Name:   "OpenCode squad agent",
		OK:     agentOK,
		Detail: agentDetail,
		Hard:   true,
	})

	team := squad.TeamPath(projectRoot)
	teamOK := fileExists(team)
	teamDetail := ".squad/team.md"
	if !teamOK {
		teamDetail = "missing .squad/team.md"
	}
	checks = append(checks, Check{
		Name:   "Team file",
		OK:     teamOK,
		Detail: teamDetail,
		Hard:   true,
	})

	return checks
}

// PrintAndExitCode prints checks and returns process exit code.
func PrintAndExitCode(projectRoot string) int {
	fmt.Printf("squad-oc doctor — %s\n\n", projectRoot)
	checks := RunChecks(projectRoot)
	hardFails := 0
	for _, c := range checks {
		mark := "OK  "
		if !c.OK {
			mark = "FAIL"
			if c.Hard {
				hardFails++
			}
		}
		fmt.Printf("[%s] %s: %s\n", mark, c.Name, c.Detail)
	}
	fmt.Println()
	if hardFails == 0 {
		fmt.Println("All required checks passed.")
		return 0
	}
	fmt.Printf("%d required check(s) failed.\n", hardFails)
	return 1
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dirExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}
