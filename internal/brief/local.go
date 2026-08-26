package brief

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xeaser/squad-opencode/internal/squad"
	"github.com/xeaser/squad-opencode/internal/watch"
)

func fillLocal(root string, rep *Report) {
	dir := squad.ResolveDir(root)
	comms := filepath.Join(dir, "comms")
	entries, err := os.ReadDir(comms)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".md") {
				continue
			}
			low := strings.ToLower(name)
			if strings.Contains(low, "design-review") || strings.Contains(low, "retro") {
				rep.InProgress.DesignReviews = append(rep.InProgress.DesignReviews, name)
			}
			body, err := os.ReadFile(filepath.Join(comms, name))
			if err != nil {
				continue
			}
			if n, ok := parseReviewNeed(name, string(body)); ok {
				rep.NeedsYou = append(rep.NeedsYou, n)
			}
		}
	}

	if h, err := watch.ReadHealth(root); err == nil {
		rep.Ralph.Present = true
		rep.Ralph.LastSummary = h.LastSummary
		rep.Ralph.LastError = h.LastError
		rep.Ralph.Overnight = h.Overnight
	}
	if _, err := os.Stat(watch.StopPath(root)); err == nil {
		rep.Ralph.Stop = true
		if !rep.Ralph.Present {
			rep.Ralph.Present = true
		}
	}

	cer := filepath.Join(dir, "ceremonies.md")
	rep.Ceremonies.Path = "ceremonies.md"
	if st, err := os.Stat(cer); err == nil && !st.IsDir() {
		rep.Ceremonies.Present = true
	}
}

func parseReviewNeed(file, body string) (ReviewNeed, bool) {
	idx := strings.Index(strings.ToLower(body), "## review")
	if idx < 0 {
		return ReviewNeed{}, false
	}
	block := body[idx:]
	if next := strings.Index(block[len("## review"):], "\n## "); next >= 0 {
		block = block[:len("## review")+next]
	}
	verdict := reviewField(block, "Verdict")
	if !strings.EqualFold(strings.TrimSpace(verdict), "reject") {
		return ReviewNeed{}, false
	}
	author := strings.TrimSpace(reviewField(block, "Author"))
	fix := strings.TrimSpace(reviewField(block, "Fix owner"))
	return ReviewNeed{
		File:      file,
		Author:    author,
		FixOwner:  fix,
		SameOwner: author != "" && strings.EqualFold(author, fix),
	}, true
}

func reviewField(block, key string) string {
	for _, line := range strings.Split(block, "\n") {
		s := strings.TrimSpace(line)
		s = strings.TrimPrefix(s, "- ")
		s = strings.TrimSpace(s)
		low := strings.ToLower(s)
		prefix := "**" + strings.ToLower(key) + ":**"
		plain := strings.ToLower(key) + ":"
		var rest string
		switch {
		case strings.HasPrefix(low, prefix):
			rest = strings.TrimSpace(s[len(prefix):])
		case strings.HasPrefix(low, plain):
			rest = strings.TrimSpace(s[len(plain):])
		default:
			continue
		}
		return strings.TrimSpace(strings.Trim(rest, "*"))
	}
	return ""
}
