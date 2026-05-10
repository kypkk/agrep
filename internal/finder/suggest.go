package finder

import (
	"sort"
	"strings"
)

// Suggestion is one "did you mean" entry.
type Suggestion struct {
	Field string // "method", "implements", "receiver"
	Value string // the suggested value
	Hint  string // optional usage hint, e.g. "(used by 3 types)"
}

// Suggest returns up to 3 suggestions per relevant flag whose value didn't
// match. Drawn from the corpus's name indexes.
func Suggest(f Filter, c *Corpus) []Suggestion {
	var out []Suggestion
	if f.HasMethod != "" {
		for _, s := range nearestNames(f.HasMethod, c.allMethodNames, 3) {
			out = append(out, Suggestion{Field: "method", Value: s})
		}
	}
	if f.Implements != "" {
		if _, ok := c.interfacesByName[f.Implements]; !ok {
			for _, s := range nearestNames(f.Implements, c.allInterfaceNames, 3) {
				out = append(out, Suggestion{Field: "implements", Value: s})
			}
		}
	}
	if f.HasReceiver != "" {
		for _, s := range nearestNames(f.HasReceiver, c.allReceiverNames, 3) {
			out = append(out, Suggestion{Field: "receiver", Value: s})
		}
	}
	return out
}

// nearestNames ranks every candidate by Levenshtein distance (with substring
// containment as tiebreaker) and returns the top `limit`. Always returns
// something when candidates is non-empty — the spec's no-results page is
// more useful with even loose suggestions than with silence.
func nearestNames(needle string, candidates []string, limit int) []string {
	type scored struct {
		name string
		dist int
		sub  bool
	}
	pool := make([]scored, 0, len(candidates))
	lowerNeedle := strings.ToLower(needle)
	for _, c := range candidates {
		pool = append(pool, scored{
			name: c,
			dist: levenshtein(needle, c),
			sub:  strings.Contains(strings.ToLower(c), lowerNeedle),
		})
	}
	sort.Slice(pool, func(i, j int) bool {
		if pool[i].dist != pool[j].dist {
			return pool[i].dist < pool[j].dist
		}
		if pool[i].sub != pool[j].sub {
			return pool[i].sub
		}
		return pool[i].name < pool[j].name
	})
	if len(pool) > limit {
		pool = pool[:limit]
	}
	out := make([]string, len(pool))
	for i, p := range pool {
		out[i] = p.name
	}
	return out
}

// levenshtein computes the edit distance between a and b. Standard DP, used
// only for short identifiers so quadratic is fine.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
