package workstream

import (
	"fmt"
	"time"

	"github.com/Arameair/Thought-Processor/core/github"
)

// Source is the read-only slice of Core's GitHub client the observer uses. It has no
// write methods on purpose: S2-A never writes to GitHub.
type Source interface {
	PullHead(repo string, number int) (string, error)
	MailboxComments(repo string, number int, since time.Time, page int) ([]github.Comment, error)
}

var _ Source = (*github.Client)(nil)

const perPage = 100 // github.Client.MailboxComments requests 100 per page

// Observe reads the owning PR's head and every comment. It returns an error rather
// than a partial observation when it cannot read every page within maxPages, or when
// the head moves while it reads (the comments would then describe a different head).
func Observe(src Source, repo string, pr, maxPages int) (Observation, error) {
	obs := Observation{Repo: repo, PR: pr}
	before, err := src.PullHead(repo, pr)
	if err != nil {
		return obs, fmt.Errorf("workstream: read PR head: %w", err)
	}
	for page := 1; ; page++ {
		if page > maxPages {
			return obs, fmt.Errorf("workstream: %s#%d has more than %d pages of comments; refusing a partial read", repo, pr, maxPages)
		}
		cs, err := src.MailboxComments(repo, pr, time.Time{}, page)
		if err != nil {
			return obs, fmt.Errorf("workstream: read comments page %d: %w", page, err)
		}
		for _, c := range cs {
			obs.Comments = append(obs.Comments, Comment{ID: c.ID, Login: c.User.Login, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Body: c.Body})
		}
		if len(cs) < perPage {
			break
		}
	}
	after, err := src.PullHead(repo, pr)
	if err != nil {
		return obs, fmt.Errorf("workstream: re-read PR head: %w", err)
	}
	if after != before {
		return obs, fmt.Errorf("workstream: %s#%d head moved from %s to %s during the read", repo, pr, short(before), short(after))
	}
	obs.HeadSHA, obs.Complete = after, true
	return obs, nil
}
