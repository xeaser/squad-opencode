package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/squad-opencode/squad-opencode/internal/doctor"
	"github.com/squad-opencode/squad-opencode/internal/squad"
)

const version = "0.1.0"

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	cmd := args[0]
	switch cmd {
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		fmt.Println(version)
	case "init":
		os.Exit(runInit(args[1:]))
	case "doctor":
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(doctor.PrintAndExitCode(cwd))
	case "status":
		os.Exit(runStatus())
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	fmt.Printf(`squad-oc v%s

Human-led AI agent teams for OpenCode (Go E2E MVP).

Usage:
  squad-oc <command> [options]

Commands:
  init [--preset default] [--description <text>]
      Scaffold .squad/ and .opencode/ in the current directory.
      Idempotent: safe if already initialized.

  doctor
      Check OpenCode, scaffold files, git, and optional server (Go SDK).

  status
      Print the team from .squad/team.md.

  help
      Show this help.

  version
      Print version.

Examples:
  squad-oc init --preset default
  squad-oc init --preset default --description "Recipe app with React"
  squad-oc doctor
  squad-oc status

Build:
  go build -o squad-oc ./cmd/squad-oc
`, version)
}

func runInit(args []string) int {
	preset := "default"
	var description string
	interactive := true

	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--preset" && i+1 < len(args):
			i++
			preset = args[i]
			interactive = false
			if preset != "default" {
				fmt.Fprintf(os.Stderr, "Unknown preset: %s. Only \"default\" is supported.\n", preset)
				return 2
			}
		case (a == "--description" || a == "-d") && i+1 < len(args):
			i++
			description = args[i]
			interactive = false
		case a == "--help" || a == "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown init flag: %s\n", a)
			return 2
		}
	}

	// --preset default alone still non-interactive for description
	for _, a := range args {
		if a == "--preset" {
			interactive = false
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if interactive && description == "" {
		fmt.Print("What are you building? (optional, Enter to skip): ")
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		description = strings.TrimSpace(line)
	}

	result, err := squad.WriteDefaultPreset(squad.InitOptions{
		ProjectRoot:        cwd,
		Preset:             preset,
		ProjectDescription: description,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if result.AlreadyInitialized {
		fmt.Println(result.Message)
		return 0
	}

	fmt.Println(result.Message)
	fmt.Printf("Project: %s\n", result.ProjectRoot)
	fmt.Printf("Files written: %d\n", len(result.FilesWritten))
	limit := 30
	for i, f := range result.FilesWritten {
		if i >= limit {
			fmt.Printf("  … and %d more\n", len(result.FilesWritten)-limit)
			break
		}
		fmt.Printf("  + %s\n", f)
	}
	fmt.Print(`
Next:
  1. opencode
  2. Switch to the "squad" agent (Tab)
  3. Say: Set up the team. Here's what I'm building: …
  4. Confirm with yes
`)
	return 0
}

func runStatus() int {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if !squad.IsInitialized(cwd) {
		fmt.Fprintln(os.Stderr, "Not initialized. Run: squad-oc init --preset default")
		return 1
	}
	det := squad.Detect(cwd)
	members, err := squad.ReadTeam(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Squad status — %s\n", cwd)
	host, preset := "?", "?"
	if det.Config != nil {
		host = det.Config.Host
		preset = det.Config.Preset
		if det.Config.ProjectDescription != "" {
			fmt.Printf("Project: %s\n", det.Config.ProjectDescription)
		}
	}
	fmt.Printf("Host: %s  Preset: %s\n\n", host, preset)

	if len(members) == 0 {
		fmt.Println("(no members parsed from .squad/team.md)")
		return 0
	}
	width := 4
	for _, m := range members {
		if len(m.Name) > width {
			width = len(m.Name)
		}
	}
	fmt.Printf("%-*s  Role\n", width, "Name")
	fmt.Printf("%s  ----\n", strings.Repeat("-", width))
	for _, m := range members {
		fmt.Printf("%-*s  %s  [%s]\n", width, m.Name, m.Role, m.Status)
	}
	return 0
}
