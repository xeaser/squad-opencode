package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/doctor"
	"github.com/xeaser/squad-opencode/internal/githubissues"
	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/share"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/updatecheck"
	"github.com/xeaser/squad-opencode/internal/version"
	"github.com/xeaser/squad-opencode/internal/watch"
)

// Execute is the CLI entry. Returns process exit code.
func Execute(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "--help", "-h":
		printHelp()
		return 0
	case "version", "--version", "-v":
		fmt.Println(version.Version)
		return 0
	case "init":
		return cmdInit(rest)
	case "doctor":
		return cmdDoctor()
	case "status", "cast":
		return cmdStatus()
	case "upgrade":
		return cmdUpgrade(rest)
	case "run":
		return cmdRun(rest)
	case "watch":
		return cmdWatch(rest)
	case "export":
		return cmdExport(rest)
	case "import":
		return cmdImport(rest)
	case "externalize":
		return cmdExternalize(rest)
	case "internalize":
		return cmdInternalize()
	case "nap":
		return cmdNap(rest)
	case "scrub-emails":
		return cmdScrub(rest)
	case "upstream":
		return cmdUpstream(rest)
	case "pack":
		return cmdPack(rest)
	case "link":
		return cmdLink(rest)
	case "update-check":
		return cmdUpdateCheck()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		return 2
	}
}

func printHelp() {
	fmt.Printf(`squad-oc v%s

Human-led AI agent teams for OpenCode.

Usage:
  squad-oc <command> [options]

Commands:
  init [--preset default] [--description <text>]
  upgrade [--dry-run] [--force]
  doctor
  status | cast
  run -p <prompt> | --file <path> [--agent name]   # needs: opencode serve
  watch [--execute] [--interval minutes] [--once]
  export [file]
  import <file>
  externalize [--key name]
  internalize
  nap [--dry-run] [--deep]
  scrub-emails [directory]
  upstream add <name> <path|git-url>
  upstream list | remove <name> | sync <name>
  pack <path|git-url>
  link <team-dir>
  update-check
  help | version

Team state (.squad/) is never wiped by upgrade.
`, version.Version)
}

func cwd() (string, int) {
	d, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", 1
	}
	return d, 0
}

func cmdInit(args []string) int {
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
				fmt.Fprintf(os.Stderr, "Unknown preset: %s\n", preset)
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
	for _, a := range args {
		if a == "--preset" {
			interactive = false
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if interactive && description == "" {
		fmt.Print("What are you building? (optional, Enter to skip): ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		description = strings.TrimSpace(line)
	}
	result, err := squad.WriteDefaultPreset(squad.InitOptions{
		ProjectRoot: root, Preset: preset, ProjectDescription: description,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(result.Message)
	if result.AlreadyInitialized {
		return 0
	}
	fmt.Printf("Project: %s\nFiles written: %d\n", result.ProjectRoot, len(result.FilesWritten))
	for i, f := range result.FilesWritten {
		if i >= 30 {
			fmt.Printf("  … and %d more\n", len(result.FilesWritten)-30)
			break
		}
		fmt.Printf("  + %s\n", f)
	}
	fmt.Print("\nNext: opencode → agent squad → confirm the team with yes\n")
	return 0
}

func cmdDoctor() int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	return doctor.PrintAndExitCode(root)
}

func cmdStatus() int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	if !squad.IsInitialized(root) {
		fmt.Fprintln(os.Stderr, "Not initialized. Run: squad-oc init --preset default")
		return 1
	}
	det := squad.Detect(root)
	members, err := squad.ReadTeam(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Squad status — %s\n", root)
	if det.Config != nil {
		fmt.Printf("Host: %s  Preset: %s\n", det.Config.Host, det.Config.Preset)
		if det.Config.ProjectDescription != "" {
			fmt.Printf("Project: %s\n", det.Config.ProjectDescription)
		}
		if det.Config.ExternalPath != "" {
			fmt.Printf("External: %s\n", det.Config.ExternalPath)
		}
		if det.Config.LinkPath != "" {
			fmt.Printf("Link: %s\n", det.Config.LinkPath)
		}
	}
	fmt.Println()
	if len(members) == 0 {
		fmt.Println("(no members parsed from team.md)")
		return 0
	}
	width := 4
	for _, m := range members {
		if len(m.Name) > width {
			width = len(m.Name)
		}
	}
	fmt.Printf("%-*s  Role\n%s  ----\n", width, "Name", strings.Repeat("-", width))
	for _, m := range members {
		fmt.Printf("%-*s  %s  [%s]\n", width, m.Name, m.Role, m.Status)
	}
	return 0
}

func cmdUpgrade(args []string) int {
	dry, force := false, false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dry = true
		case "--force":
			force = true
		case "--self":
			fmt.Println("upgrade --self is not wired yet. Build from source:")
			fmt.Println("  go install github.com/xeaser/squad-opencode/cmd/squad-oc@latest")
			return 0
		case "--help", "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown upgrade flag: %s\n", a)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	res, err := squad.UpgradeHostFiles(squad.UpgradeOptions{ProjectRoot: root, DryRun: dry, Force: force})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(res.Message)
	for _, f := range res.Updated {
		fmt.Printf("  ~ %s\n", f)
	}
	for _, f := range res.Created {
		fmt.Printf("  + %s\n", f)
	}
	return 0
}

func cmdRun(args []string) int {
	var prompt, file, agent string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case (a == "-p" || a == "--prompt") && i+1 < len(args):
			i++
			prompt = args[i]
		case a == "--file" && i+1 < len(args):
			i++
			file = args[i]
		case a == "--agent" && i+1 < len(args):
			i++
			agent = args[i]
		default:
			fmt.Fprintf(os.Stderr, "Unknown run flag: %s\n", a)
			return 2
		}
	}
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		prompt = string(b)
	}
	if strings.TrimSpace(prompt) == "" {
		fmt.Fprintln(os.Stderr, "run requires -p <prompt> or --file <path>")
		return 2
	}
	if agent == "" {
		agent = "squad"
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	probe := opencodeclient.ProbeServer(context.Background(), "")
	if !probe.Reachable {
		fmt.Fprintf(os.Stderr, "OpenCode HTTP API not reachable at %s.\n%s\n\n%s\n",
			opencodeclient.DefaultBaseURL, probe.Detail, opencodeclient.StartHelp)
		return 1
	}
	res, err := (opencodeclient.SDKRunner{}).Run(context.Background(), opencodeclient.RunRequest{
		Directory: root, Agent: agent, Prompt: prompt, Title: "squad-oc run",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(res.Text)
	return 0
}

func cmdWatch(args []string) int {
	exec := false
	once := false
	interval := 10
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--execute":
			exec = true
		case a == "--once":
			once = true
		case a == "--interval" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				fmt.Fprintln(os.Stderr, "invalid --interval")
				return 2
			}
			interval = n
		default:
			fmt.Fprintf(os.Stderr, "Unknown watch flag: %s\n", a)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	opts := watch.Options{
		ProjectRoot: root,
		Execute:     exec,
		Once:        once,
		Interval:    time.Duration(interval) * time.Minute,
		Lister:      githubissues.GHLister{Dir: root},
	}
	if exec {
		opts.Runner = opencodeclient.SDKRunner{}
	}
	if once || !exec {
		_, summary, err := watch.Pass(context.Background(), opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(summary)
		return 0
	}
	fmt.Printf("watch loop every %d min (stop: touch .squad/ralph-stop)\n", interval)
	return watchLoop(opts)
}

func watchLoop(opts watch.Options) int {
	if err := watch.Loop(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdExport(args []string) int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	dest := "squad-export.json"
	if len(args) > 0 {
		dest = args[0]
	}
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(root, dest)
	}
	if err := squad.Export(root, dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("exported", dest)
	return 0
}

func cmdImport(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "import requires a file")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if err := squad.Import(root, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("imported", args[0])
	return 0
}

func cmdExternalize(args []string) int {
	key := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--key" && i+1 < len(args) {
			i++
			key = args[i]
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	dest, err := squad.Externalize(root, key)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("externalized →", dest)
	return 0
}

func cmdInternalize() int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	if err := squad.Internalize(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("internalized")
	return 0
}

func cmdNap(args []string) int {
	dry, deep := false, false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dry = true
		case "--deep":
			deep = true
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	res, err := squad.Nap(root, dry, deep)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(res.Message)
	return 0
}

func cmdScrub(args []string) int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	dir := ""
	if len(args) > 0 {
		dir = args[0]
	}
	n, err := squad.ScrubEmails(root, dir, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("scrubbed %d file(s)\n", n)
	return 0
}

func cmdUpstream(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "upstream add|list|remove|sync")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	switch args[0] {
	case "list":
		list, err := share.ListUpstreams(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(list) == 0 {
			fmt.Println("(no upstreams)")
			return 0
		}
		for _, u := range list {
			fmt.Printf("%s\t%s\n", u.Name, u.Path)
		}
		return 0
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "upstream add <name> <path|git-url>")
			return 2
		}
		if err := share.AddUpstream(root, args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("added", args[1])
		return 0
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "upstream remove <name>")
			return 2
		}
		if err := share.RemoveUpstream(root, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("removed", args[1])
		return 0
	case "sync":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "upstream sync <name>")
			return 2
		}
		n, err := share.SyncUpstream(root, args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("synced %d file(s)\n", n)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "upstream add|list|remove|sync")
		return 2
	}
}

func cmdPack(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "pack <path|git-url>")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	n, err := share.InstallPack(root, args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("installed %d file(s)\n", n)
	return 0
}

func cmdLink(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "link <team-directory>")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if err := share.Link(root, args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("linked", args[0])
	return 0
}

func cmdUpdateCheck() int {
	res, err := updatecheck.Check(nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(res.Message)
	return 0
}
