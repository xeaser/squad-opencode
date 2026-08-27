package brief

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xeaser/squad-opencode/internal/squad"
)

type Ticket struct {
	Source    string    `json:"source,omitempty"`
	ID        string    `json:"id"`
	Number    int       `json:"number,omitempty"`
	Title     string    `json:"title"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
}

type PR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      string `json:"author,omitempty"`
	Draft       bool   `json:"draft,omitempty"`
	Review      string `json:"review,omitempty"`
	MergedAt    string `json:"mergedAt,omitempty"`
	LinkedIssue []int  `json:"linkedIssue,omitempty"`
}

type TicketSource interface {
	ListOpen(ctx context.Context) ([]Ticket, error)
}

type PRSource interface {
	ListOpen(ctx context.Context) ([]PR, error)
	ListMerged(ctx context.Context, limit int) ([]PR, error)
}

type Options struct {
	ProjectRoot string
	Tickets     TicketSource
	PRs         PRSource
}

type Member struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type Team struct {
	Theme    string   `json:"theme"`
	Link     string   `json:"link"`
	External string   `json:"external"`
	Members  []Member `json:"members"`
}

type SourceList[T any] struct {
	OK    bool   `json:"ok"`
	Items []T    `json:"items"`
	Error string `json:"error,omitempty"`
}

type ReviewNeed struct {
	File      string `json:"file"`
	Author    string `json:"author"`
	FixOwner  string `json:"fixOwner"`
	SameOwner bool   `json:"sameOwner"`
}

type Report struct {
	Team       Team               `json:"team"`
	PRs        SourceList[PR]     `json:"prs"`
	Tickets    SourceList[Ticket] `json:"tickets"`
	InProgress struct {
		PRs           []PR     `json:"prs"`
		DesignReviews []string `json:"designReviews"`
	} `json:"inProgress"`
	LastDone struct {
		OK  bool `json:"ok"`
		PRs []PR `json:"prs"`
	} `json:"lastDone"`
	Next       *Ticket      `json:"next"`
	Ralph      RalphInfo    `json:"ralph"`
	NeedsYou   []ReviewNeed `json:"needsYou"`
	Ceremonies struct {
		Path    string `json:"path"`
		Present bool   `json:"present"`
	} `json:"ceremonies"`
}

type RalphInfo struct {
	LastSummary string `json:"lastSummary"`
	LastError   string `json:"lastError"`
	Overnight   bool   `json:"overnight"`
	Stop        bool   `json:"stop"`
	Present     bool   `json:"present"`
}

func Collect(ctx context.Context, opts Options) (Report, error) {
	var rep Report
	rep.Team.Members = []Member{}
	rep.PRs.Items = []PR{}
	rep.Tickets.Items = []Ticket{}
	rep.InProgress.PRs = []PR{}
	rep.InProgress.DesignReviews = []string{}
	rep.LastDone.PRs = []PR{}
	rep.NeedsYou = []ReviewNeed{}
	if !squad.IsInitialized(opts.ProjectRoot) {
		return rep, fmt.Errorf("not initialized")
	}
	det := squad.Detect(opts.ProjectRoot)
	theme := "none"
	if det.Config != nil {
		if det.Config.Theme != "" {
			theme = det.Config.Theme
		}
		rep.Team.Link = det.Config.LinkPath
		rep.Team.External = det.Config.ExternalPath
	}
	rep.Team.Theme = theme
	members, err := squad.ReadTeam(opts.ProjectRoot)
	if err != nil {
		return rep, err
	}
	for _, m := range members {
		rep.Team.Members = append(rep.Team.Members, Member{Name: m.Name, Role: m.Role, Status: m.Status})
	}

	rep.PRs = SourceList[PR]{Error: "unavailable", Items: []PR{}}
	rep.Tickets = SourceList[Ticket]{Error: "unavailable", Items: []Ticket{}}
	if opts.PRs != nil {
		open, err := opts.PRs.ListOpen(ctx)
		if err != nil {
			rep.PRs.Error = err.Error()
		} else {
			if open == nil {
				open = []PR{}
			}
			rep.PRs = SourceList[PR]{OK: true, Items: open}
			rep.InProgress.PRs = open
		}
		merged, err := opts.PRs.ListMerged(ctx, 5)
		if err == nil {
			if merged == nil {
				merged = []PR{}
			}
			rep.LastDone.OK = true
			rep.LastDone.PRs = merged
		}
	}
	if opts.Tickets != nil {
		items, err := opts.Tickets.ListOpen(ctx)
		if err != nil {
			rep.Tickets.Error = err.Error()
		} else {
			if items == nil {
				items = []Ticket{}
			}
			rep.Tickets = SourceList[Ticket]{OK: true, Items: items}
		}
	}
	if rep.PRs.OK && rep.Tickets.OK {
		rep.Next = pickNext(rep.Tickets.Items, rep.PRs.Items)
	}

	fillLocal(opts.ProjectRoot, &rep)
	return rep, nil
}

func pickNext(issues []Ticket, prs []PR) *Ticket {
	linked := map[int]bool{}
	for _, p := range prs {
		for _, n := range p.LinkedIssue {
			linked[n] = true
		}
	}
	var best *Ticket
	for i := range issues {
		n := issues[i].Number
		if n == 0 {
			if _, err := fmt.Sscanf(strings.TrimPrefix(issues[i].ID, "#"), "%d", &n); err != nil {
				n = 0
			}
		}
		if n != 0 && linked[n] {
			continue
		}
		cand := issues[i]
		if best == nil {
			best = &cand
			continue
		}
		if cand.CreatedAt.Before(best.CreatedAt) || (cand.CreatedAt.Equal(best.CreatedAt) && n < best.Number) {
			best = &cand
		}
	}
	return best
}

func Format(r Report) string {
	var b strings.Builder
	b.WriteString("Morning brief\n\n")
	b.WriteString("Team\n")
	fmt.Fprintf(&b, "  Theme: %s\n", r.Team.Theme)
	if r.Team.Link != "" {
		fmt.Fprintf(&b, "  Link: %s\n", r.Team.Link)
	}
	if r.Team.External != "" {
		fmt.Fprintf(&b, "  External: %s\n", r.Team.External)
	}
	for _, m := range r.Team.Members {
		fmt.Fprintf(&b, "  %s  %s  [%s]\n", m.Name, m.Role, m.Status)
	}
	b.WriteString("\nOpen PRs\n")
	writePRSection(&b, r.PRs)
	b.WriteString("\nTickets\n")
	if !r.Tickets.OK {
		fmt.Fprintf(&b, "  unavailable\n")
	} else if len(r.Tickets.Items) == 0 {
		fmt.Fprintf(&b, "  (none)\n")
	} else {
		for _, t := range r.Tickets.Items {
			fmt.Fprintf(&b, "  %s  %s\n", t.ID, t.Title)
		}
	}
	b.WriteString("\nIn progress\n")
	if len(r.InProgress.PRs) == 0 && len(r.InProgress.DesignReviews) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, p := range r.InProgress.PRs {
			fmt.Fprintf(&b, "  PR #%d  %s\n", p.Number, p.Title)
		}
		for _, f := range r.InProgress.DesignReviews {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}
	b.WriteString("\nLast done\n")
	if !r.LastDone.OK {
		b.WriteString("  unavailable\n")
	} else if len(r.LastDone.PRs) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, p := range r.LastDone.PRs {
			fmt.Fprintf(&b, "  PR #%d  %s\n", p.Number, p.Title)
		}
	}
	b.WriteString("\nNext\n")
	switch {
	case !r.Tickets.OK || !r.PRs.OK:
		b.WriteString("  unavailable\n")
	case r.Next == nil:
		b.WriteString("  (none)\n")
	default:
		fmt.Fprintf(&b, "  %s  %s\n", r.Next.ID, r.Next.Title)
	}
	b.WriteString("\nRalph\n")
	if !r.Ralph.Present {
		b.WriteString("  (none)\n")
	} else {
		fmt.Fprintf(&b, "  last: %s\n", r.Ralph.LastSummary)
		if r.Ralph.LastError != "" {
			fmt.Fprintf(&b, "  error: %s\n", r.Ralph.LastError)
		}
		fmt.Fprintf(&b, "  overnight: %v\n  stop: %v\n", r.Ralph.Overnight, r.Ralph.Stop)
	}
	b.WriteString("\nNeeds you\n")
	if len(r.NeedsYou) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, n := range r.NeedsYou {
			line := fmt.Sprintf("  %s  author=%s  fix=%s", n.File, n.Author, n.FixOwner)
			if n.SameOwner {
				line += "  FLAG author==fix-owner"
			}
			b.WriteString(line + "\n")
		}
	}
	if r.Ceremonies.Path != "" {
		state := "missing"
		if r.Ceremonies.Present {
			state = "present"
		}
		fmt.Fprintf(&b, "\nCeremonies: %s (%s)\n", r.Ceremonies.Path, state)
	}
	return b.String()
}

func writePRSection(b *strings.Builder, list SourceList[PR]) {
	if !list.OK {
		b.WriteString("  unavailable\n")
		return
	}
	if len(list.Items) == 0 {
		b.WriteString("  (none)\n")
		return
	}
	for _, p := range list.Items {
		draft := ""
		if p.Draft {
			draft = " draft"
		}
		fmt.Fprintf(b, "  #%d  %s  %s%s  %s\n", p.Number, p.Title, p.Author, draft, p.Review)
	}
}

func FormatJSON(r Report) ([]byte, error) {
	if r.Team.Members == nil {
		r.Team.Members = []Member{}
	}
	if r.PRs.Items == nil {
		r.PRs.Items = []PR{}
	}
	if r.Tickets.Items == nil {
		r.Tickets.Items = []Ticket{}
	}
	if r.InProgress.PRs == nil {
		r.InProgress.PRs = []PR{}
	}
	if r.InProgress.DesignReviews == nil {
		r.InProgress.DesignReviews = []string{}
	}
	if r.LastDone.PRs == nil {
		r.LastDone.PRs = []PR{}
	}
	if r.NeedsYou == nil {
		r.NeedsYou = []ReviewNeed{}
	}
	return json.MarshalIndent(r, "", "  ")
}
