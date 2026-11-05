package tass

import (
	"fmt"
	"path"
	"strings"
)

// handleLessonChat recognizes a tiny, owner-only command vocabulary.
// It is not an LLM tool call and makes no assumptions about a model reply.
// Every real operation delegates to the same Identity-checked DiscoverLessons
// and StudyLesson methods as the buttons; no arbitrary commands, URLs or code.
func (c *Core) handleLessonChat(m Incoming) bool {
	if c.cfg.IdentitySvc == nil || !(m.From == "local" || (m.From == "github" && c.relayOn())) {
		return false
	}
	s := strings.ToLower(strings.TrimSpace(m.Text))
	for _, prefix := range []string{"hey tass, ", "hey tass ", "tass, ", "tass "} {
		s = strings.TrimPrefix(s, prefix)
	}
	s = strings.TrimSpace(strings.TrimRight(s, " .!?"))

	var study string
	switch s {
	case "find lessons", "find the lessons", "show lessons", "show me lessons",
		"show me the lessons", "show me your lessons", "what lessons are available",
		"what lessons can you learn", "can you see any lessons",
		"can you see any of the lessons", "good can you see any of the lessons":
		// Just list owner-approved lesson paths and actual states.
	case "can we try to learn any of them", "can we try to learn one",
		"can we try to learn a lesson", "try learning a lesson",
		"try to learn a lesson", "study one lesson", "run one lesson test",
		"try to learn one of them":
		study = "*"
	default:
		for _, prefix := range []string{"study lesson ", "study ", "learn lesson ", "learn "} {
			if strings.HasPrefix(s, prefix) {
				study = strings.TrimSpace(strings.TrimPrefix(s, prefix))
				break
			}
		}
		if study == "" {
			return false
		}
	}

	if err := c.DiscoverLessons(); err != nil {
		c.event("problem", "Lesson request: "+err.Error())
		c.lessonSay(m, "I could not verify the lessons through Identity and GitHub: "+err.Error()+". No lesson job was started.")
		return true
	}
	ls := c.Learning().Lessons
	if study == "" {
		if len(ls) == 0 {
			c.lessonSay(m, "I checked the configured GitHub lessons folder, but found no lessons. No test was run.")
			return true
		}
		var b strings.Builder
		fmt.Fprintf(&b, "I checked GitHub %s@%s and found %d lesson(s):", c.cfg.Repo, c.cfg.Branch, len(ls))
		for _, l := range ls {
			fmt.Fprintf(&b, "\n- %s [%s] (%s)", l.Title, l.Status, l.Path)
		}
		b.WriteString("\nTo run one supervised test, say 'study My work pulse' or click Study (manual) under Learning. Autonomous can stay OFF.")
		c.lessonSay(m, b.String())
		return true
	}
	if c.GetSettings().Autonomous {
		c.lessonSay(m, "Autonomous is ON. Turn it OFF before requesting a supervised lesson test; otherwise a passed lesson may be auto-approved.")
		return true
	}

	selected := ""
	if study == "*" {
		for _, l := range ls {
			if l.Status == "NEW" || l.Status == "RETRY" {
				selected = l.Path
				break // sorted by path; notebooks cannot enter this state
			}
		}
	} else {
		for _, l := range ls {
			if strings.EqualFold(l.Title, study) ||
				strings.EqualFold(l.Path, study) ||
				strings.EqualFold(path.Base(l.Path), study) {
				if selected != "" {
					c.lessonSay(m, "Several lessons match that name. Please use the full lesson path shown in Learning.")
					return true
				}
				selected = l.Path
			}
		}
	}
	if selected == "" {
		c.lessonSay(m, "No matching NEW or RETRY lesson can be started. Open Learning to inspect lesson names and states; no test was run.")
		return true
	}
	issue := 0
	if m.From == "github" {
		issue = m.Issue
	}
	id, err := c.studyLesson(selected, issue)
	if err != nil {
		c.event("problem", "Lesson study: "+err.Error())
		c.lessonSay(m, "I could not start the lesson: "+err.Error()+". No completed test is claimed.")
		return true
	}
	c.lessonSay(m, fmt.Sprintf("I submitted lesson %s to the existing Thought Engine: job %s. One manual pulse step was requested. Watch Learning for the actual answers, PASSED, RETRY or STUCK. I will not approve it for you.%s", selected, id, map[bool]string{true: " I'll post my graded answers here when I'm done; only my owner can approve on the local page.", false: ""}[issue > 0]))
	return true
}
