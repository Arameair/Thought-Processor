// Package tassmd reads, edits and merges tass.md: who TASS is, his memory and what he's doing now.
//
// The file is markdown split into sections by "## " headings. A three-way merge works
// section by section: a section changed on only one side takes that side; a section changed
// on both sides is a conflict, except "## Memory", whose bullet lines are merged (TASS and
// Dustin can both add memories while offline without a conflict).
package tassmd

import (
	"strings"
	"time"
)

// MemorySection and NowSection are the parts TASS writes himself.
const (
	MemorySection = "Memory"
	NowSection    = "Now"
)

// Section is one "## Name" block. The text before the first heading is the section "".
type Section struct {
	Name string
	Body string // everything after the heading line, including its trailing newline(s)
}

// Parse splits a document into sections, keeping their order.
func Parse(doc string) []Section {
	doc = strings.ReplaceAll(doc, "\r\n", "\n")
	var out []Section
	cur := Section{}
	var b strings.Builder
	for _, line := range strings.SplitAfter(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			cur.Body = b.String()
			out = append(out, cur)
			b.Reset()
			cur = Section{Name: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			continue
		}
		b.WriteString(line)
	}
	cur.Body = b.String()
	return append(out, cur)
}

// Render is the inverse of Parse.
func Render(secs []Section) string {
	var b strings.Builder
	for _, s := range secs {
		if s.Name != "" {
			b.WriteString("## " + s.Name + "\n")
		}
		b.WriteString(s.Body)
	}
	return b.String()
}

func find(secs []Section, name string) (string, bool) {
	for _, s := range secs {
		if s.Name == name {
			return s.Body, true
		}
	}
	return "", false
}

func norm(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n")) }

// Merge combines base (last synced), local and remote. Conflicting sections take the remote
// text in merged; mine is the same document with TASS's (local) text in those sections instead,
// so the choice can be offered to Dustin. conflicts lists the section names.
func Merge(base, local, remote string) (merged, mine string, conflicts []string) {
	if norm(local) == norm(remote) {
		return local, local, nil
	}
	if norm(local) == norm(base) {
		return remote, remote, nil
	}
	if norm(remote) == norm(base) {
		return local, local, nil
	}
	b, l, r := Parse(base), Parse(local), Parse(remote)
	// Section order: remote's, then sections only local has (TASS added them).
	names := []string{}
	seen := map[string]bool{}
	for _, s := range r {
		names, seen[s.Name] = append(names, s.Name), true
	}
	for _, s := range l {
		if !seen[s.Name] {
			names, seen[s.Name] = append(names, s.Name), true
		}
	}
	var outM, outL []Section
	for _, n := range names {
		bb, inB := find(b, n)
		lb, inL := find(l, n)
		rb, inR := find(r, n)
		var pick, alt string
		keep := true
		switch {
		case inL && inR && norm(lb) == norm(rb):
			pick, alt = rb, rb
		case !inR: // only local has it (or remote deleted it)
			if inB && norm(lb) == norm(bb) {
				keep = false // remote deleted an unchanged section
			}
			pick, alt = lb, lb
		case !inL: // only remote has it (or local deleted it)
			if inB && norm(rb) == norm(bb) {
				keep = false
			}
			pick, alt = rb, rb
		case norm(lb) == norm(bb):
			pick, alt = rb, rb
		case norm(rb) == norm(bb):
			pick, alt = lb, lb
		case n == MemorySection:
			pick = mergeLines(bb, lb, rb)
			alt = pick
		default:
			conflicts = append(conflicts, n)
			pick, alt = rb, lb
		}
		if keep {
			outM = append(outM, Section{n, pick})
			outL = append(outL, Section{n, alt})
		}
	}
	return Render(outM), Render(outL), conflicts
}

// mergeLines keeps remote's lines (minus any line local removed), then adds lines only local added.
func mergeLines(base, local, remote string) string {
	in := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, l := range strings.Split(s, "\n") {
			if t := strings.TrimSpace(l); t != "" {
				m[t] = true
			}
		}
		return m
	}
	bs, ls, rs := in(base), in(local), in(remote)
	var out []string
	for _, l := range strings.Split(strings.TrimRight(remote, "\n"), "\n") {
		t := strings.TrimSpace(l)
		if t != "" && bs[t] && !ls[t] {
			continue // TASS removed it
		}
		out = append(out, l)
	}
	for _, l := range strings.Split(strings.TrimRight(local, "\n"), "\n") {
		t := strings.TrimSpace(l)
		if t != "" && !bs[t] && !rs[t] {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n") + "\n\n"
}

// AddMemory appends "- YYYY-MM-DD: text" to ## Memory (created if missing). Duplicates are skipped.
func AddMemory(doc, text string, now time.Time) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return doc
	}
	secs := Parse(doc)
	line := "- " + now.Format("2006-01-02") + ": " + text
	for i, s := range secs {
		if s.Name == MemorySection {
			for _, l := range strings.Split(s.Body, "\n") {
				if idx := strings.Index(l, ": "); strings.HasPrefix(l, "- ") && idx > 0 && strings.TrimSpace(l[idx+2:]) == text {
					return doc
				}
			}
			secs[i].Body = strings.TrimRight(s.Body, "\n") + "\n" + line + "\n\n"
			if strings.TrimSpace(s.Body) == "" {
				secs[i].Body = "\n" + line + "\n\n"
			}
			return Render(secs)
		}
	}
	return Render(insertBefore(secs, NowSection, Section{MemorySection, "\n" + line + "\n\n"}))
}

// SetNow replaces ## Now (what TASS is doing / thinking about right now).
func SetNow(doc, text string, now time.Time) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return doc
	}
	body := "\n" + text + "\n\n_updated " + now.Format("2006-01-02 15:04") + "_\n"
	secs := Parse(doc)
	for i, s := range secs {
		if s.Name == NowSection {
			secs[i].Body = body
			return Render(secs)
		}
	}
	secs[len(secs)-1].Body = strings.TrimRight(secs[len(secs)-1].Body, "\n") + "\n\n"
	return Render(append(secs, Section{NowSection, body}))
}

func insertBefore(secs []Section, name string, s Section) []Section {
	for i, x := range secs {
		if x.Name == name {
			out := append([]Section{}, secs[:i]...)
			out = append(out, s)
			return append(out, secs[i:]...)
		}
	}
	secs[len(secs)-1].Body = strings.TrimRight(secs[len(secs)-1].Body, "\n") + "\n\n"
	return append(secs, s)
}
