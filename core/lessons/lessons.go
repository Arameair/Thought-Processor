// Package lessons reads lesson files and grades an entity's answers to them.
//
// A lesson the app can teach has checkable steps:
//
//	# Lesson title
//	by: Claude
//
//	One line: what the entity will be able to do afterwards.
//
//	## Step 1: short name
//	Teach: what this step explains (one or more lines).
//	Check: a question the entity answers in its own words.
//	Answer: the expected answer (a word, a phrase or a number; "a | b" means either)
//
// The Answer lines are never shown to the entity: they're removed from what it studies, and
// the app (not the model) decides whether each answer is right. Older lessons made of Python
// cells for the workspace have no Answer lines; they're listed as "needs the workspace".
package lessons

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

type Step struct {
	N      int    `json:"n"`
	Name   string `json:"name"`
	Teach  string `json:"teach,omitempty"`
	Check  string `json:"check,omitempty"`
	Answer string `json:"-"` // never sent to the entity or the page
}

type Lesson struct {
	Title     string `json:"title"`
	By        string `json:"by,omitempty"`
	Goal      string `json:"goal,omitempty"`
	Steps     []Step `json:"steps"`
	Retests   []Step `json:"-"` // "## Retest N" sections: fresh questions for after approval, never shown in study
	Checkable bool   `json:"checkable"` // every lesson the app can grade has at least one Check with an Answer
	HasCode   bool   `json:"has_code"`
}

var (
	stepRe   = regexp.MustCompile(`(?i)^##\s*step\s*(\d+)\s*[:.)-]?\s*(.*)$`)
	retestRe = regexp.MustCompile(`(?i)^##\s*retest\s*(\d+)\s*[:.)-]?\s*(.*)$`)
	fieldRe  = regexp.MustCompile(`(?i)^(teach|check|answer)\s*:\s*(.*)$`)
	answerRe = regexp.MustCompile(`(?i)^[\s*_>-]*answer\s*(\d+)\s*[:.)-]\s*(.*)$`)
	numRe    = regexp.MustCompile(`-?\d+(?:\.\d+)?`)
	wordRe   = regexp.MustCompile(`[a-z0-9]+`)
)

// Parse reads a lesson file.
func Parse(text string) Lesson {
	var l Lesson
	var cur *Step
	field := ""
	inCode := false
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimRight(raw, " \t")
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCode, l.HasCode = !inCode, true
			continue
		}
		if inCode {
			continue
		}
		t := strings.TrimSpace(line)
		switch {
		case l.Title == "" && strings.HasPrefix(t, "# "):
			l.Title = strings.TrimSpace(t[2:])
			continue
		case cur == nil && strings.HasPrefix(strings.ToLower(t), "by:"):
			l.By = strings.TrimSpace(t[3:])
			continue
		}
		if m := retestRe.FindStringSubmatch(t); m != nil {
			n, _ := strconv.Atoi(m[1])
			l.Retests = append(l.Retests, Step{N: n, Name: strings.TrimSpace(m[2])})
			cur, field = &l.Retests[len(l.Retests)-1], ""
			continue
		}
		if m := stepRe.FindStringSubmatch(t); m != nil {
			n, _ := strconv.Atoi(m[1])
			l.Steps = append(l.Steps, Step{N: n, Name: strings.TrimSpace(m[2])})
			cur, field = &l.Steps[len(l.Steps)-1], ""
			continue
		}
		if cur == nil {
			if l.Goal == "" && t != "" && !strings.HasPrefix(t, "#") {
				l.Goal = t
			}
			continue
		}
		if m := fieldRe.FindStringSubmatch(t); m != nil {
			field = strings.ToLower(m[1])
			set(cur, field, strings.TrimSpace(m[2]))
			continue
		}
		if field == "teach" || (field == "" && t != "") {
			field = "teach"
			set(cur, "teach", t)
		}
	}
	for _, s := range l.Steps {
		if s.Check != "" && s.Answer != "" {
			l.Checkable = true
		}
	}
	return l
}

func set(s *Step, field, v string) {
	if v == "" {
		return
	}
	switch field {
	case "teach":
		s.Teach = strings.TrimSpace(s.Teach + "\n" + v)
	case "check":
		s.Check = strings.TrimSpace(s.Check + " " + v)
	case "answer":
		s.Answer = strings.TrimSpace(s.Answer + " " + v)
	}
}

// ForStudy is the lesson as the entity sees it: everything except the expected answers.
func (l Lesson) ForStudy() string {
	var b strings.Builder
	b.WriteString("# " + l.Title + "\n")
	if l.Goal != "" {
		b.WriteString(l.Goal + "\n")
	}
	for _, s := range l.Steps {
		b.WriteString("\n## Step " + strconv.Itoa(s.N) + ": " + s.Name + "\n")
		if s.Teach != "" {
			b.WriteString(s.Teach + "\n")
		}
		if s.Check != "" {
			b.WriteString("Check " + strconv.Itoa(s.N) + ": " + s.Check + "\n")
		}
	}
	return b.String()
}

// Checks returns the steps that have a check and an expected answer.
func (l Lesson) Checks() []Step {
	var out []Step
	for _, s := range l.Steps {
		if s.Check != "" && s.Answer != "" {
			out = append(out, s)
		}
	}
	return out
}

// Result of grading one check.
type Result struct {
	N      int    `json:"n"`
	Check  string `json:"check"`
	Given  string `json:"given"`
	Right  bool   `json:"right"`
	Reason string `json:"reason"`
}

// Grade compares the entity's "ANSWER n: …" lines with the lesson's expected answers.
func Grade(l Lesson, reply string) (results []Result, right, total int) {
	return GradeSteps(l.Checks(), reply)
}

// GradeSteps grades "ANSWER n: …" lines against any list of checks (a lesson's, or a retest's).
func GradeSteps(checks []Step, reply string) (results []Result, right, total int) {
	given := map[int]string{}
	for _, line := range strings.Split(reply, "\n") {
		if m := answerRe.FindStringSubmatch(line); m != nil {
			n, _ := strconv.Atoi(m[1])
			if _, seen := given[n]; !seen {
				given[n] = strings.TrimSpace(m[2])
			}
		}
	}
	for _, s := range checks {
		total++
		g, ok := given[s.N]
		r := Result{N: s.N, Check: s.Check, Given: g}
		switch {
		case !ok || g == "":
			r.Reason = "no ANSWER " + strconv.Itoa(s.N) + " line"
		case matches(s.Answer, g):
			r.Right, r.Reason = true, "matches the lesson's answer"
			right++
		default:
			r.Reason = "doesn't match the lesson's answer"
		}
		results = append(results, r)
	}
	return results, right, total
}

// matches: the expected answer (or one of its "|" alternatives) appears in what was given.
// Numbers must be equal; words must all appear, in any order, ignoring case and punctuation.
func matches(expected, given string) bool {
	g := strings.ToLower(given)
	gw := map[string]bool{}
	for _, w := range wordRe.FindAllString(g, -1) {
		gw[w] = true
	}
	for _, alt := range strings.Split(expected, "|") {
		alt = strings.ToLower(strings.TrimSpace(alt))
		if alt == "" {
			continue
		}
		if f, err := strconv.ParseFloat(alt, 64); err == nil {
			for _, n := range numRe.FindAllString(g, -1) {
				if v, err := strconv.ParseFloat(n, 64); err == nil && v == f {
					return true
				}
			}
			if word, ok := numberWords[alt]; ok && gw[word] {
				return true
			}
			continue
		}
		all := true
		for _, w := range wordRe.FindAllString(alt, -1) {
			if !gw[w] {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

var numberWords = map[string]string{"0": "zero", "1": "one", "2": "two", "3": "three", "4": "four", "5": "five",
	"6": "six", "7": "seven", "8": "eight", "9": "nine", "10": "ten", "20": "twenty"}

// Notebook reads a notebook lesson's title, goal and code cells (with their THOUGHT) for listing.
// The app never runs them.
func Notebook(text string) Lesson {
	var nb struct {
		Cells []struct {
			Type     string          `json:"cell_type"`
			Source   json.RawMessage `json:"source"`
			Metadata struct {
				Tass struct{ Title, Thought, Expect string } `json:"tass"`
			} `json:"metadata"`
		} `json:"cells"`
	}
	l := Lesson{HasCode: true}
	if json.Unmarshal([]byte(text), &nb) != nil {
		return l
	}
	for _, c := range nb.Cells {
		src := sourceText(c.Source)
		if c.Type == "markdown" && l.Title == "" {
			md := Parse(src)
			l.Title, l.By, l.Goal = md.Title, md.By, md.Goal
		}
		if c.Type == "code" {
			l.Steps = append(l.Steps, Step{N: len(l.Steps) + 1, Name: c.Metadata.Tass.Title, Teach: c.Metadata.Tass.Thought})
		}
	}
	return l
}

func sourceText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []string
	_ = json.Unmarshal(raw, &parts)
	return strings.Join(parts, "")
}
