package workstream

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Arameair/Thought-Processor/core/github"
)

// fakePR serves one PR's head and paged comments. It fails the test on any non-GET.
type fakePR struct {
	t        *testing.T
	mu       sync.Mutex
	heads    []string // successive PullHead answers; the last one repeats
	comments []map[string]any
	reads    int
}

func (f *fakePR) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		f.t.Errorf("S2-A must not write to GitHub: %s %s", r.Method, r.URL)
		http.Error(w, "read-only", http.StatusMethodNotAllowed)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.URL.Path == fmt.Sprintf("/repos/%s/pulls/%d", repo, pr):
		h := f.heads[0]
		if len(f.heads) > 1 {
			f.heads = f.heads[1:]
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"head": map[string]string{"sha": h}})
	case r.URL.Path == fmt.Sprintf("/repos/%s/issues/%d/comments", repo, pr):
		f.reads++
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		lo, hi := (page-1)*100, page*100
		if lo > len(f.comments) {
			lo = len(f.comments)
		}
		if hi > len(f.comments) {
			hi = len(f.comments)
		}
		_ = json.NewEncoder(w).Encode(f.comments[lo:hi])
	default:
		http.NotFound(w, r)
	}
}

func client(f *fakePR) (*github.Client, func()) {
	srv := httptest.NewServer(f)
	c := github.New("")
	c.Base = srv.URL
	return c, srv.Close
}

func apiComment(c Comment) map[string]any {
	return map[string]any{"id": c.ID, "body": c.Body, "user": map[string]string{"login": c.Login},
		"created_at": c.CreatedAt, "updated_at": c.UpdatedAt,
		"issue_url": fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", repo, pr)}
}

func TestObserveThenEvaluateAgainstFakeGitHub(t *testing.T) {
	tr := trust()
	f := &fakePR{t: t, heads: []string{c2}}
	// 150 discussion comments force a second page; events sit on both pages.
	evs := map[int64]Comment{
		3:   post(&tr, 3, event("e1", Candidate, "Claude", c1, "")),
		40:  post(&tr, 40, event("e2", Audit, "Codex", c1, NeedsRepair)),
		120: post(&tr, 120, event("e3", Candidate, "Claude", c2, "")),
		149: post(&tr, 149, event("e4", Audit, "Codex", c2, Pass)),
	}
	for id := int64(1); id <= 150; id++ {
		c, ok := evs[id]
		if !ok {
			c = comment(id, owner, "Codex: PASS (plain text, ignored)")
		}
		f.comments = append(f.comments, apiComment(c))
	}
	gh, stop := client(f)
	defer stop()
	obs, err := Observe(gh, repo, pr, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(obs.Comments) != 150 || f.reads != 2 || !obs.Complete || obs.HeadSHA != c2 {
		t.Fatalf("comments=%d reads=%d complete=%v head=%s", len(obs.Comments), f.reads, obs.Complete, obs.HeadSHA)
	}
	r := Evaluate(tr, obs)
	if r.Phase != PhaseAuditPassed || !r.Current || strings.Join(transitions(r.Accepted), ",") != "BUILD,AUDIT,REPAIR,RE-AUDIT" {
		t.Fatalf("phase=%s current=%v transitions=%v blocked=%v rejected=%v", r.Phase, r.Current, transitions(r.Accepted), reasons(r.Blocked), reasons(r.Rejected))
	}
}

func TestObserveDetectsEditsFromUpdatedAt(t *testing.T) {
	tr := trust()
	c := post(&tr, 1, event("e1", Candidate, "Claude", c1, ""))
	c.UpdatedAt = c.CreatedAt.Add(1)
	f := &fakePR{t: t, heads: []string{c1}, comments: []map[string]any{apiComment(c)}}
	gh, stop := client(f)
	defer stop()
	obs, err := Observe(gh, repo, pr, 10)
	if err != nil {
		t.Fatal(err)
	}
	wantBlocked(t, Evaluate(tr, obs), RModified)
}

func TestObserveRefusesPartialOrMovingReads(t *testing.T) {
	t.Run("head moved during the read", func(t *testing.T) {
		f := &fakePR{t: t, heads: []string{c1, c2}}
		gh, stop := client(f)
		defer stop()
		if obs, err := Observe(gh, repo, pr, 10); err == nil || obs.Complete {
			t.Fatalf("moving head accepted: %+v", obs)
		}
	})
	t.Run("more pages than allowed", func(t *testing.T) {
		f := &fakePR{t: t, heads: []string{c1}}
		for id := int64(1); id <= 250; id++ {
			f.comments = append(f.comments, apiComment(comment(id, owner, "x")))
		}
		gh, stop := client(f)
		defer stop()
		if obs, err := Observe(gh, repo, pr, 2); err == nil || obs.Complete {
			t.Fatalf("partial enumeration accepted: %d comments", len(obs.Comments))
		}
	})
	t.Run("PR not found", func(t *testing.T) {
		f := &fakePR{t: t, heads: []string{c1}}
		gh, stop := client(f)
		defer stop()
		if _, err := Observe(gh, repo, pr+1, 2); err == nil {
			t.Fatal("missing PR accepted")
		}
	})
}
