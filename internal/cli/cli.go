package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xeaser/squad-opencode/internal/brief"
	"github.com/xeaser/squad-opencode/internal/doctor"
	"github.com/xeaser/squad-opencode/internal/githubissues"
	"github.com/xeaser/squad-opencode/internal/mcpconfig"
	"github.com/xeaser/squad-opencode/internal/opencodeclient"
	"github.com/xeaser/squad-opencode/internal/selfupdate"
	"github.com/xeaser/squad-opencode/internal/share"
	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/traces"
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
	case "doctor", "heartbeat":
		return cmdDoctor()
	case "status":
		return cmdStatus()
	case "brief":
		return cmdBrief(rest)
	case "cast":
		return cmdCast(rest)
	case "recast":
		return cmdRecast()
	case "upgrade":
		return cmdUpgrade(rest)
	case "run":
		return cmdRun(rest)
	case "watch", "triage", "loop":
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
		return cmdUpdateCheck(rest)
	case "traces":
		return cmdTraces(rest)
	case "mcp":
		return cmdMCP(rest)
	case "marketplace":
		return cmdMarketplace(rest)
	case "plugin":
		return cmdPlugin(rest)
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
  init [--preset default] [--description <text>] [--global] [--theme office|none]
  upgrade [--dry-run] [--force] [--global] [--self]
  doctor | heartbeat
  status | cast
  brief [--json]
  cast --add <name> [--role <role>] [--model <provider/id>]
  cast --remove <name>
  cast --theme office|none
  cast --model <name> <provider/id>
  cast --model squad <provider/id>
  cast --model <name|-> -
  recast
  run -p <prompt> | --file <path> [--agent name] [--url]
  watch | triage | loop [--execute] [--interval minutes] [--once] [--health] [--url]
      [--overnight-start HH:MM] [--overnight-end HH:MM] [--label name]
      [--project N] [--column name] [--force] [--retry-label name]
      [--log-file path] [--verbose] [--notify-level all|important|none]
      [--state-backend memory|git-notes|orphan-branch]
  export [file]
  import <file> [--with-host]
  externalize [--key name]
  internalize
  nap [--dry-run] [--deep]
  scrub-emails [directory]
  upstream add <name> <path|git-url>
  upstream list | remove <name> | sync <name>
  pack <path|git-url>
  link <team-dir|git-url>
  link --sync
  link --off
  update-check [--json] [--refresh]
  traces [--last N] [--json] [--export file] [--follow]   # JSONL + OpenCode SQLite ingest; OTEL optional
  mcp apply | list | init
  marketplace add <name> <path|git-url> | list | remove <name> | browse [name] | install <plugin> [--from <name>]
  plugin install <name>@<marketplace> | list | uninstall <name>
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
	var theme string
	interactive := true
	global := false
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
		case a == "--theme" && i+1 < len(args):
			i++
			theme = args[i]
		case a == "--theme":
			fmt.Fprintln(os.Stderr, "init --theme requires office or none")
			return 2
		case a == "--global":
			global = true
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
	if theme != "" {
		norm, err := squad.NormalizeTheme(theme)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		theme = norm
	}
	if interactive && description == "" {
		fmt.Print("What are you building? (optional, Enter to skip): ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		description = strings.TrimSpace(line)
	}
	result, err := squad.WriteDefaultPreset(squad.InitOptions{
		ProjectRoot: root, Preset: preset, ProjectDescription: description, Global: global, Theme: theme,
	})
	if err != nil {
		if errors.Is(err, squad.ErrUnknownTheme) {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println(result.Message)
	if result.AlreadyInitialized {
		if global {
			fmt.Println(result.ProjectRoot)
		}
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
	out, err := squad.StatusReport(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Print(out)
	return 0
}

var newBriefSources = func(root string) (brief.TicketSource, brief.PRSource) {
	gh := brief.GitHub{Dir: root}
	return brief.GitHubTickets{GitHub: gh}, brief.GitHubPRs{GitHub: gh}
}

func cmdBrief(args []string) int {
	asJSON := false
	for _, a := range args {
		switch a {
		case "--json":
			asJSON = true
		case "--help", "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown brief flag: %s\n", a)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if !squad.IsInitialized(root) {
		fmt.Fprintln(os.Stderr, "Not initialized. Run: squad-oc init --preset default")
		return 1
	}
	tickets, prs := newBriefSources(root)
	rep, err := brief.Collect(context.Background(), brief.Options{
		ProjectRoot: root,
		Tickets:     tickets,
		PRs:         prs,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if asJSON {
		raw, err := brief.FormatJSON(rep)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(raw))
		return 0
	}
	fmt.Print(brief.Format(rep))
	return 0
}

func cmdCast(args []string) int {
	var add, role, remove, theme, model, modelName string
	addMode := false
	for _, a := range args {
		if a == "--add" {
			addMode = true
			break
		}
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--add" && i+1 < len(args):
			i++
			add = args[i]
		case a == "--remove" && i+1 < len(args):
			i++
			remove = args[i]
		case a == "--role" && i+1 < len(args):
			i++
			role = args[i]
		case a == "--theme" && i+1 < len(args):
			i++
			theme = args[i]
		case a == "--theme":
			fmt.Fprintln(os.Stderr, "cast --theme requires office or none")
			return 2
		case a == "--model":
			if addMode {
				if i+1 >= len(args) {
					fmt.Fprintln(os.Stderr, "cast --model requires a model id")
					return 2
				}
				i++
				model = args[i]
			} else {
				if i+2 >= len(args) {
					fmt.Fprintln(os.Stderr, "cast --model requires <name> <provider/id>")
					return 2
				}
				i++
				modelName = args[i]
				i++
				model = args[i]
			}
		case a == "--help" || a == "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown cast flag: %s\n", a)
			return 2
		}
	}
	haveModel := model != "" || modelName != ""
	if theme != "" && (add != "" || remove != "" || haveModel) {
		fmt.Fprintln(os.Stderr, "cast: use --theme alone")
		return 2
	}
	if add != "" && remove != "" {
		fmt.Fprintln(os.Stderr, "cast: use --add or --remove, not both")
		return 2
	}
	if haveModel && remove != "" {
		fmt.Fprintln(os.Stderr, "cast: use --model or --remove, not both")
		return 2
	}
	if theme != "" {
		return cmdCastTheme(theme)
	}
	if add == "" && remove == "" && !haveModel {
		return cmdStatus()
	}
	if haveModel {
		if _, err := squad.ValidateModelID(model); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if remove != "" {
		if err := squad.RemoveMember(root, remove); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		res, err := squad.Recast(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("removed %s; recast %d agent file(s)\n", remove, res.Written)
		return 0
	}
	if add != "" {
		if err := squad.AddMember(root, add, role, model); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		res, err := squad.Recast(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("added %s; recast %d agent file(s)\n", add, res.Written)
		return 0
	}
	if strings.EqualFold(modelName, "squad") {
		if err := squad.SetSquadModel(root, model); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	} else {
		if err := squad.SetMemberModel(root, modelName, model); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	res, err := squad.Recast(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("model %s; recast %d agent file(s)\n", modelName, res.Written)
	return 0
}

func cmdCastTheme(theme string) int {
	if _, err := squad.NormalizeTheme(theme); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if err := squad.ApplyTheme(root, theme); err != nil {
		if errors.Is(err, squad.ErrUnknownTheme) {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	res, err := squad.Recast(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("theme %s; recast %d agent file(s)\n", strings.ToLower(theme), res.Written)
	return 0
}

func cmdRecast() int {
	root, code := cwd()
	if code != 0 {
		return code
	}
	res, err := squad.Recast(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	theme, origin := "", ""
	if det := squad.Detect(root); det.Config != nil {
		theme = det.Config.Theme
		origin = det.Config.ThemeOrigin
	}
	fmt.Printf("recast %d agent file(s)\n", res.Written)
	for _, id := range res.IDs {
		fmt.Printf("  ~ .opencode/agents/%s.md\n", squad.HostAgentID(id, theme, origin))
	}
	return 0
}

func cmdUpgrade(args []string) int {
	dry, force, global := false, false, false
	for _, a := range args {
		switch a {
		case "--dry-run":
			dry = true
		case "--force":
			force = true
		case "--global":
			global = true
		case "--self":
			msg, err := selfupdate.UpgradeSelf(nil, version.Repo, version.Version)
			if err != nil && !errors.Is(err, selfupdate.ErrReplacedOnNextStart) {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Println(msg)
			if errors.Is(err, selfupdate.ErrReplacedOnNextStart) {
				fmt.Println("replaced on next start")
			}
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
	if global {
		g, err := squad.GlobalSquadDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		root = g
	}
	res, err := squad.UpgradeHostFiles(squad.UpgradeOptions{ProjectRoot: root, DryRun: dry, Force: force})
	if err != nil {
		if global && !squad.IsInitialized(root) {
			fmt.Fprintln(os.Stderr, "not initialized — run: squad-oc init --global")
			return 1
		}
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
	var prompt, file, agent, apiURL string
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
		case a == "--url" && i+1 < len(args):
			i++
			apiURL = args[i]
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
	if _, err := traces.ResolveSettings(squad.Detect(root).Config, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	ensured, code := ensureAPI(apiURL, root)
	if code != 0 {
		return code
	}
	res, err := (opencodeclient.SDKRunner{BaseURL: ensured.BaseURL}).Run(context.Background(), opencodeclient.RunRequest{
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
	health := false
	interval := 10
	verbose := false
	notifyLevel := watch.NotifyImportant
	var overnightStart, overnightEnd, apiURL, logFile, stateBackend, retryLabel, column string
	var labels []string
	force := false
	project := 0
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--execute":
			exec = true
		case a == "--once":
			once = true
		case a == "--health":
			health = true
		case a == "--verbose":
			verbose = true
		case a == "--force":
			force = true
		case a == "--interval" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				fmt.Fprintln(os.Stderr, "invalid --interval")
				return 2
			}
			interval = n
		case a == "--overnight-start" && i+1 < len(args):
			i++
			overnightStart = args[i]
		case a == "--overnight-end" && i+1 < len(args):
			i++
			overnightEnd = args[i]
		case a == "--url" && i+1 < len(args):
			i++
			apiURL = args[i]
		case a == "--label" && i+1 < len(args):
			i++
			labels = append(labels, args[i])
		case a == "--project" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 1 {
				fmt.Fprintln(os.Stderr, "invalid --project")
				return 2
			}
			project = n
		case a == "--column" && i+1 < len(args):
			i++
			column = args[i]
		case a == "--retry-label" && i+1 < len(args):
			i++
			retryLabel = args[i]
		case a == "--log-file" && i+1 < len(args):
			i++
			logFile = args[i]
		case a == "--notify-level" && i+1 < len(args):
			i++
			lvl, err := watch.ParseNotifyLevel(args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 2
			}
			notifyLevel = lvl
		case a == "--state-backend" && i+1 < len(args):
			i++
			stateBackend = args[i]
		default:
			fmt.Fprintf(os.Stderr, "Unknown watch flag: %s\n", a)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if _, err := traces.ResolveSettings(squad.Detect(root).Config, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	backend, err := watch.ParseStateBackend(stateBackend, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if column != "" && project == 0 {
		fmt.Fprintln(os.Stderr, "--column requires --project")
		return 2
	}
	if health {
		return cmdWatchHealth(root, backend)
	}
	opts := watch.Options{
		ProjectRoot:    root,
		Execute:        exec,
		Once:           once,
		Interval:       time.Duration(interval) * time.Minute,
		OvernightStart: overnightStart,
		OvernightEnd:   overnightEnd,
		Labels:         labels,
		LogFile:        logFile,
		Verbose:        verbose,
		Notify:         notifyLevel,
		Logger: func(_ watch.NotifyLevel, msg string) {
			fmt.Println(msg)
		},
		Backend:    backend,
		Lister:     githubissues.GHLister{Dir: root, Labels: labels},
		PRChecker:  githubissues.GHPRChecker{Dir: root},
		Project:    project,
		Column:     column,
		Force:      force,
		RetryLabel: retryLabel,
	}
	if project > 0 {
		opts.ProjectSource = githubissues.GHProjectSource{Dir: root}
	}
	if exec {
		ensured, code := ensureAPI(apiURL, root)
		if code != 0 {
			return code
		}
		opts.Runner = opencodeclient.SDKRunner{BaseURL: ensured.BaseURL}
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

func cmdWatchHealth(root string, backend watch.StateBackend) int {
	var h watch.Health
	var err error
	if backend != nil {
		h, err = backend.Load(context.Background())
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if h.PID == 0 && h.Round == 0 && h.StartedAt.IsZero() {
			fmt.Println("no watch status (start: squad-oc watch)")
			return 1
		}
	} else {
		h, err = watch.ReadHealth(root)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("no watch status (start: squad-oc watch)")
				return 1
			}
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	fmt.Print(watch.FormatHealth(h, time.Now()))
	return 0
}

func ensureAPI(apiURL, root string) (opencodeclient.EnsureResult, int) {
	res, err := opencodeclient.EnsureAPI(context.Background(), apiURL, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return res, 1
	}
	fmt.Println(res.Message)
	return res, 0
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
	withHost := false
	src := ""
	for _, a := range args {
		switch {
		case a == "--with-host":
			withHost = true
		case a == "--help" || a == "-h":
			printHelp()
			return 0
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "Unknown import flag: %s\n", a)
			return 2
		default:
			if src != "" {
				fmt.Fprintln(os.Stderr, "import accepts one file")
				return 2
			}
			src = a
		}
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "import requires a file")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if err := squad.Import(root, src, withHost); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("imported", src)
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
		fmt.Fprintln(os.Stderr, "link <team-directory|git-url> | link --sync | link --off")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	switch args[0] {
	case "--help", "-h":
		printHelp()
		return 0
	case "--off":
		if err := share.Unlink(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("unlinked")
		return 0
	case "--sync":
		dest, err := share.SyncLink(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("synced →", dest)
		return 0
	}
	dest, err := share.Link(root, args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("linked →", dest)
	return 0
}

func cmdUpdateCheck(args []string) int {
	var asJSON, refresh bool
	for _, a := range args {
		switch a {
		case "--json":
			asJSON = true
		case "--refresh":
			refresh = true
		case "--help", "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown update-check flag: %s\n", a)
			return 2
		}
	}
	res, err := updatecheck.Check(nil, refresh)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(res); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Println(res.Message)
	return 0
}

func cmdTraces(args []string) int {
	last := 20
	asJSON := false
	exportPath := ""
	follow := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			asJSON = true
		case a == "--follow":
			follow = true
		case a == "--last" && i+1 < len(args):
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil || n < 0 {
				fmt.Fprintln(os.Stderr, "invalid --last")
				return 2
			}
			last = n
		case a == "--export" && i+1 < len(args):
			i++
			exportPath = args[i]
		case a == "--export":
			fmt.Fprintln(os.Stderr, "traces --export requires a file")
			return 2
		case a == "--last":
			fmt.Fprintln(os.Stderr, "traces --last requires N")
			return 2
		case a == "--help" || a == "-h":
			printHelp()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "Unknown traces flag: %s\n", a)
			return 2
		}
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	if follow {
		return tracesFollowFn(root)
	}
	if code := tracesIngestOnce(root); code != 0 {
		return code
	}
	spans, err := traces.List(root, last)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if exportPath != "" {
		if !filepath.IsAbs(exportPath) {
			exportPath = filepath.Join(root, exportPath)
		}
		if err := traces.ExportOTLPFile(spans, exportPath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("exported", exportPath)
		if !asJSON {
			return 0
		}
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(spans); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}
	fmt.Print(traces.FormatTable(spans))
	return 0
}

var tracesFollowFn = tracesFollow

// tracesIngestPush is the OTLP export hook for CLI ingest; tests replace it.
var tracesIngestPush = traces.Push

func tracesIngestOnce(root string) int {
	_, err := traces.Ingest(root, squad.Detect(root).Config, os.Getenv, tracesIngestPush)
	if err != nil {
		if traces.IsOTLPPushError(err) {
			fmt.Fprintln(os.Stderr, "traces:", err)
			return 0
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func tracesFollow(root string) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	for {
		n, err := traces.Ingest(root, squad.Detect(root).Config, os.Getenv, tracesIngestPush)
		if err != nil {
			if traces.IsOTLPPushError(err) {
				fmt.Fprintln(os.Stderr, "traces:", err)
				// JSONL kept; keep polling
			} else {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if n > 0 {
			fmt.Printf("ingested %d\n", n)
		}
		select {
		case <-ctx.Done():
			return 0
		case <-time.After(2 * time.Second):
		}
	}
}

func cmdMCP(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "mcp apply|list|init")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	switch args[0] {
	case "apply":
		if err := mcpconfig.Apply(root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("applied MCP → opencode.json")
		return 0
	case "list":
		items, err := mcpconfig.List(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(items) == 0 {
			fmt.Println("(no MCP servers)")
			return 0
		}
		fmt.Printf("%-16s %-8s %-8s %s\n", "name", "source", "enabled", "applied")
		for _, it := range items {
			src := it.Source
			if src == "" {
				src = "-"
			}
			fmt.Printf("%-16s %-8s %-8t %t\n", it.Name, src, it.Enabled, it.Applied)
		}
		return 0
	case "init":
		created, path, err := mcpconfig.InitExample(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		rel := path
		if r, err := filepath.Rel(root, path); err == nil {
			rel = filepath.ToSlash(r)
		}
		if !created {
			fmt.Println(rel, "already exists")
			return 0
		}
		fmt.Println("wrote", rel)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "mcp apply|list|init")
		return 2
	}
}

func cmdMarketplace(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "marketplace add|list|remove|browse|install")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	switch args[0] {
	case "list":
		list, err := share.ListMarketplaces(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(list) == 0 {
			fmt.Println("(no marketplaces)")
			return 0
		}
		for _, m := range list {
			fmt.Printf("%s\t%s\n", m.Name, m.Path)
		}
		return 0
	case "add":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "marketplace add <name> <path|git-url>")
			return 2
		}
		if err := share.AddMarketplace(root, args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("added", args[1])
		return 0
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "marketplace remove <name>")
			return 2
		}
		if err := share.RemoveMarketplace(root, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("removed", args[1])
		return 0
	case "browse":
		name := ""
		if len(args) > 1 {
			name = args[1]
		}
		plugins, err := share.BrowsePlugins(root, name)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(plugins) == 0 {
			fmt.Println("(no plugins)")
			return 0
		}
		fmt.Printf("%-20s %-36s %s\n", "plugin", "description", "triggers")
		for _, p := range plugins {
			fmt.Printf("%-20s %-36s %s\n", p.Name, p.Description, p.Triggers)
		}
		return 0
	case "install":
		plugin, from, err := parsePluginInstall(args[1:], "marketplace install <plugin> [--from <name>]")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		n, err := share.InstallPlugin(root, plugin, from)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("installed %s (%d file(s))\n", plugin, n)
		return 0
	default:
		fmt.Fprintln(os.Stderr, "marketplace add|list|remove|browse|install")
		return 2
	}
}

func cmdPlugin(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "plugin install|list|uninstall")
		return 2
	}
	root, code := cwd()
	if code != 0 {
		return code
	}
	switch args[0] {
	case "list":
		list, err := share.ListInstalledPlugins(root)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if len(list) == 0 {
			fmt.Println("(no plugins)")
			return 0
		}
		fmt.Printf("%-20s %-16s %s\n", "plugin", "source", "version")
		for _, p := range list {
			fmt.Printf("%-20s %-16s %s\n", p.Name, p.Source, p.Version)
		}
		return 0
	case "install":
		plugin, from, err := parsePluginInstall(args[1:], "plugin install <name>@<marketplace>")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		n, err := share.InstallPlugin(root, plugin, from)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Printf("installed %s (%d file(s))\n", plugin, n)
		return 0
	case "uninstall":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "plugin uninstall <name>")
			return 2
		}
		if err := share.UninstallPlugin(root, args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println("uninstalled", args[1])
		return 0
	default:
		fmt.Fprintln(os.Stderr, "plugin install|list|uninstall")
		return 2
	}
}

func parsePluginInstall(args []string, usage string) (plugin, from string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--from" && i+1 < len(args):
			i++
			from = args[i]
		case strings.HasPrefix(a, "--from="):
			from = strings.TrimPrefix(a, "--from=")
		case a == "--from":
			return "", "", fmt.Errorf("%s", usage)
		case a == "--help" || a == "-h":
			return "", "", fmt.Errorf("%s", usage)
		case strings.HasPrefix(a, "-"):
			return "", "", fmt.Errorf("unknown install flag: %s", a)
		default:
			if plugin != "" {
				return "", "", fmt.Errorf("%s", usage)
			}
			plugin = a
		}
	}
	if plugin == "" {
		return "", "", fmt.Errorf("%s", usage)
	}
	name, market, err := share.ParsePluginSpec(plugin)
	if err != nil {
		return "", "", err
	}
	if market != "" {
		if from != "" && from != market {
			return "", "", fmt.Errorf("conflicting marketplace %q vs %q", market, from)
		}
		from = market
	}
	return name, from, nil
}
