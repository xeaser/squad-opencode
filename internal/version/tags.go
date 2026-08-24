package version

import (
	"sort"
	"strconv"
	"strings"
)

// IsStableTag reports vMAJOR.MINOR.PATCH with no prerelease suffix.
func IsStableTag(tag string) bool {
	_, ok := parseStable(tag)
	return ok
}

// PreviousStableTag is the stable tag immediately before current, or "".
func PreviousStableTag(current string, tags []string) string {
	cur, ok := parseStable(current)
	if !ok {
		return ""
	}
	var list []stableTag
	for _, t := range tags {
		st, ok := parseStable(t)
		if !ok || st.raw == cur.raw {
			continue
		}
		list = append(list, st)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].less(list[j]) })
	var prev string
	for _, st := range list {
		if !st.less(cur) {
			break
		}
		prev = st.raw
	}
	return prev
}

type stableTag struct {
	raw           string
	maj, min, pat int
}

func (a stableTag) less(b stableTag) bool {
	if a.maj != b.maj {
		return a.maj < b.maj
	}
	if a.min != b.min {
		return a.min < b.min
	}
	return a.pat < b.pat
}

func parseStable(tag string) (stableTag, bool) {
	if !strings.HasPrefix(tag, "v") {
		return stableTag{}, false
	}
	parts := strings.Split(tag[1:], ".")
	if len(parts) != 3 {
		return stableTag{}, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return stableTag{}, false
	}
	if parts[0] != strconv.Itoa(maj) || parts[1] != strconv.Itoa(min) || parts[2] != strconv.Itoa(pat) {
		return stableTag{}, false
	}
	return stableTag{raw: tag, maj: maj, min: min, pat: pat}, true
}
