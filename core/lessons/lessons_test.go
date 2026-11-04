package lessons

import (
	"strings"
	"testing"
)

const pulseLesson = `# How my pulse works
by: Claude

Know what happens every 20 seconds, so I can explain it to my owner.

## Step 1: The beat
Teach: My work pulse ticks every 20 seconds while my owner has it turned on.
Each tick looks at what's waiting and decides one thing.
Check: How many seconds are there between two ticks?
Answer: 20

## Step 2: Who decides
Teach: Before anything runs, Identity decides. If Identity is locked, nothing runs.
Check: Which service decides whether an action may run?
Answer: Identity

## Step 3: Say it back
Teach: Put it together in one sentence.
`

func TestParseKeepsAnswersOutOfWhatTheEntityStudies(t *testing.T) {
	l := Parse(pulseLesson)
	if l.Title != "How my pulse works" || l.By != "Claude" || !strings.HasPrefix(l.Goal, "Know what happens") {
		t.Fatalf("%+v", l)
	}
	if len(l.Steps) != 3 || len(l.Checks()) != 2 || !l.Checkable || l.HasCode {
		t.Fatalf("steps %+v", l.Steps)
	}
	if !strings.Contains(l.Steps[0].Teach, "Each tick looks") {
		t.Fatalf("multi-line teach: %q", l.Steps[0].Teach)
	}
	s := l.ForStudy()
	if strings.Contains(s, "Answer") || strings.Contains(s, "Identity decides.") == false || strings.Contains(s, "\n20\n") {
		t.Fatalf("the entity must never see the expected answers:\n%s", s)
	}
	if !strings.Contains(s, "Check 1: How many seconds") {
		t.Fatalf("checks are numbered for the reply:\n%s", s)
	}
}

func TestGrade(t *testing.T) {
	l := Parse(pulseLesson)
	res, right, total := Grade(l, "Here's what I learned.\nANSWER 1: twenty seconds\n**ANSWER 2:** the Identity service decides")
	if right != 2 || total != 2 {
		t.Fatalf("%d/%d %+v", right, total, res)
	}
	_, right, _ = Grade(l, "ANSWER 1: 30 seconds\nANSWER 2: the model")
	if right != 0 {
		t.Fatal("wrong answers must not count")
	}
	res, right, _ = Grade(l, "I think it's every 20 seconds and Identity decides.")
	if right != 0 || res[0].Reason != "no ANSWER 1 line" {
		t.Fatalf("answers count only on their ANSWER lines: %+v", res)
	}
}

func TestWorkspaceLessonIsNotCheckable(t *testing.T) {
	l := Parse("# Make a tool\nby: Claude\n\nBuild one.\n\n## Step 1: Write it\nWhy: a tool is a function.\n```python\nprint('hi')\n```\nYou should see: `hi`\n")
	if l.Checkable || !l.HasCode {
		t.Fatalf("%+v", l)
	}
}

func TestAlternatives(t *testing.T) {
	if !matches("locked | LOCKED state", "it stays locked") || matches("3", "33 things") || !matches("3", "three") {
		t.Fatal("alternatives / numbers")
	}
}
