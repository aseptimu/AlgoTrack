// Package catalog holds curated lists of LeetCode problems used by the
// recommendation engine. Each catalog is an ordered slice — the first
// problem the user has not yet solved AND has not yet been recommended is
// what /next will surface.
package catalog

import "fmt"

// Difficulty buckets used by the recommendation engine.
const (
	Easy   = "Easy"
	Medium = "Medium"
	Hard   = "Hard"
)

// Problem describes a single LeetCode problem entry in a curated list.
type Problem struct {
	Number     int    // LeetCode frontend question id
	TitleSlug  string // canonical leetcode slug, e.g. "two-sum"
	Title      string
	Difficulty string // Easy | Medium | Hard
	Topic      string // human-readable bucket
}

// Link returns the canonical leetcode.com problem URL.
func (p Problem) Link() string {
	return fmt.Sprintf("https://leetcode.com/problems/%s/", p.TitleSlug)
}

// Catalog is a named, ordered list of curated problems.
type Catalog struct {
	Name     string
	Problems []Problem
}

// Chain returns the ordered chain of catalogs the recommender should walk
// for a given user mode. The first catalog with an eligible problem wins.
//
//   - "default" → NeetCode 150 → Popular fallback
//   - "js"      → LeetCode 30 Days JS → NeetCode 150 → Popular fallback
//
// Unknown modes fall through to the default chain.
func Chain(mode string) []Catalog {
	switch mode {
	case "js":
		return []Catalog{LeetCodeJS30, NeetCode150, Popular}
	default:
		return []Catalog{NeetCode150, Popular}
	}
}
