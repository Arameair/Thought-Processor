// Package github is the small slice of the GitHub REST API the core needs.
//
// Reads send If-None-Match with the last ETag: a "304 Not Modified" answer doesn't count
// against GitHub's rate limit, which is what makes a 20-second pulse affordable.
package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrNotFound is a 404: missing file, branch or repo.
var ErrNotFound = errors.New("not found on GitHub")

// ErrConflict is a 409/422 on write: someone changed the file since we read it.
var ErrConflict = errors.New("changed on GitHub since we read it")

type Client struct {
	Base  string // https://api.github.com
	Token string
	HTTP  *http.Client
}

func New(token string) *Client {
	return &Client{Base: "https://api.github.com", Token: token, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// Response of a conditional GET.
type Got struct {
	NotModified bool
	ETag        string
	Body        []byte
}

func (c *Client) do(method, path string, body any, etag string) (*Got, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, strings.TrimRight(c.Base, "/")+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "tass-core")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err // offline: the caller keeps going on the local copy
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	switch {
	case resp.StatusCode == http.StatusNotModified:
		return &Got{NotModified: true, ETag: etag}, nil
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusConflict || (resp.StatusCode == 422 && method == http.MethodPut):
		return nil, ErrConflict
	case resp.StatusCode >= 300:
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(data, &e)
		return nil, fmt.Errorf("GitHub %s %s: %d %s", method, path, resp.StatusCode, e.Message)
	}
	return &Got{ETag: resp.Header.Get("ETag"), Body: data}, nil
}

// File is a file's text and blob sha on a branch.
type File struct {
	Text string
	SHA  string
}

// GetFile reads path on branch. With a matching etag it returns NotModified and no content.
func (c *Client) GetFile(repo, branch, path, etag string) (*File, *Got, error) {
	g, err := c.do("GET", fmt.Sprintf("/repos/%s/contents/%s?ref=%s", repo, path, url.QueryEscape(branch)), nil, etag)
	if err != nil || g.NotModified {
		return nil, g, err
	}
	var f struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
		SHA      string `json:"sha"`
	}
	if err := json.Unmarshal(g.Body, &f); err != nil {
		return nil, nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(f.Content, "\n", ""))
	if err != nil {
		return nil, nil, err
	}
	return &File{Text: string(raw), SHA: f.SHA}, g, nil
}

// PutFile writes path on branch (sha = current blob sha, or "" for a new file). Returns the new blob sha.
func (c *Client) PutFile(repo, branch, path, text, sha, message string) (string, error) {
	body := map[string]any{"message": message, "branch": branch,
		"content": base64.StdEncoding.EncodeToString([]byte(text))}
	if sha != "" {
		body["sha"] = sha
	}
	g, err := c.do("PUT", fmt.Sprintf("/repos/%s/contents/%s", repo, path), body, "")
	if err != nil {
		return "", err
	}
	var r struct {
		Content struct {
			SHA string `json:"sha"`
		} `json:"content"`
	}
	_ = json.Unmarshal(g.Body, &r)
	return r.Content.SHA, nil
}

// BranchSHA is the commit sha at the tip of branch.
func (c *Client) BranchSHA(repo, branch string) (string, error) {
	g, err := c.do("GET", fmt.Sprintf("/repos/%s/git/ref/heads/%s", repo, branch), nil, "")
	if err != nil {
		return "", err
	}
	var r struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	return r.Object.SHA, json.Unmarshal(g.Body, &r)
}

func (c *Client) DefaultBranch(repo string) (string, error) {
	g, err := c.do("GET", "/repos/"+repo, nil, "")
	if err != nil {
		return "", err
	}
	var r struct {
		DefaultBranch string `json:"default_branch"`
	}
	return r.DefaultBranch, json.Unmarshal(g.Body, &r)
}

func (c *Client) CreateBranch(repo, branch, fromSHA string) error {
	_, err := c.do("POST", fmt.Sprintf("/repos/%s/git/refs", repo),
		map[string]string{"ref": "refs/heads/" + branch, "sha": fromSHA}, "")
	return err
}

// CreatePR opens a pull request and returns its URL.
func (c *Client) CreatePR(repo, head, base, title, body string) (string, error) {
	g, err := c.do("POST", fmt.Sprintf("/repos/%s/pulls", repo),
		map[string]string{"head": head, "base": base, "title": title, "body": body}, "")
	if err != nil {
		return "", err
	}
	var r struct {
		HTMLURL string `json:"html_url"`
	}
	return r.HTMLURL, json.Unmarshal(g.Body, &r)
}

// User is a GitHub login.
type User struct {
	Login string `json:"login"`
}

type Issue struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	User        User      `json:"user"`
	HTMLURL     string    `json:"html_url"`
	UpdatedAt   time.Time `json:"updated_at"`
	PullRequest *struct{} `json:"pull_request"`
	State       string    `json:"state"`
	Labels      []struct{ Name string `json:"name"` } `json:"labels"`
}

type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	IssueURL  string    `json:"issue_url"`
	CreatedAt time.Time `json:"created_at"`
}

// Issue number from a comment's issue_url.
func (c Comment) IssueNumber() int {
	var n int
	if i := strings.LastIndex(c.IssueURL, "/"); i >= 0 {
		fmt.Sscan(c.IssueURL[i+1:], &n)
	}
	return n
}

// Issues lists open issues with label (conditional on etag).
func (c *Client) Issues(repo, label, etag string) ([]Issue, *Got, error) {
	g, err := c.do("GET", fmt.Sprintf("/repos/%s/issues?state=open&labels=%s&per_page=50", repo, url.QueryEscape(label)), nil, etag)
	if err != nil || g.NotModified {
		return nil, g, err
	}
	var out []Issue
	return out, g, json.Unmarshal(g.Body, &out)
}

// Comments lists the repo's issue comments since t (conditional on etag).
func (c *Client) Comments(repo string, since time.Time, etag string) ([]Comment, *Got, error) {
	g, err := c.do("GET", fmt.Sprintf("/repos/%s/issues/comments?sort=created&direction=asc&per_page=100&since=%s",
		repo, url.QueryEscape(since.UTC().Format(time.RFC3339))), nil, etag)
	if err != nil || g.NotModified {
		return nil, g, err
	}
	var out []Comment
	return out, g, json.Unmarshal(g.Body, &out)
}

func (c *Client) Comment(repo string, issue int, body string) error {
	_, err := c.do("POST", fmt.Sprintf("/repos/%s/issues/%d/comments", repo, issue), map[string]string{"body": body}, "")
	return err
}

 // MailboxIssue validates the exact pilot issue before any new comment is read.
func (c *Client) MailboxIssue(repo string, number int) (Issue, error) {
    var is Issue
    g, err := c.do("GET", fmt.Sprintf("/repos/%s/issues/%d", repo, number), nil, "")
    if err != nil { return is, err }
    err = json.Unmarshal(g.Body, &is)
    return is, err
}

// MailboxComments reads one page from one issue, not a repository-wide feed.
// Pagination is handled by the caller so a busy repository cannot hide #24.
func (c *Client) MailboxComments(repo string, number int, since time.Time, page int) ([]Comment, error) {
    path := fmt.Sprintf("/repos/%s/issues/%d/comments?per_page=100&page=%d", repo, number, page)
    if !since.IsZero() { path += "&since=" + url.QueryEscape(since.UTC().Format(time.RFC3339)) }
    g, err := c.do("GET", path, nil, "")
    if err != nil { return nil, err }
    var comments []Comment
    return comments, json.Unmarshal(g.Body, &comments)
}

// PostComment returns the actual ID assigned by GitHub. A response without an
// ID is NOT a verified receipt; the caller must not pretend it was delivered.
func (c *Client) PostComment(repo string, issue int, body string) (int64, error) {
    g, err := c.do("POST", fmt.Sprintf("/repos/%s/issues/%d/comments", repo, issue),
        map[string]string{"body":body}, "")
    if err != nil { return 0, err }
    var receipt struct{ ID int64 `json:"id"` }
    if err := json.Unmarshal(g.Body, &receipt); err != nil { return 0, err }
    if receipt.ID <= 0 { return 0, fmt.Errorf("GitHub reply had no comment ID; delivery unverified") }
    return receipt.ID, nil
}

// TreeEntry is one file in a branch's tree.
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // blob | tree
	SHA  string `json:"sha"`
}

// Tree lists every file on branch (one request; with a matching etag it returns NotModified).
func (c *Client) Tree(repo, branch, etag string) ([]TreeEntry, *Got, error) {
	g, err := c.do("GET", fmt.Sprintf("/repos/%s/git/trees/%s?recursive=1", repo, url.PathEscape(branch)), nil, etag)
	if err != nil || g.NotModified {
		return nil, g, err
	}
	var t struct {
		Tree      []TreeEntry `json:"tree"`
		Truncated bool        `json:"truncated"`
	}
	if err := json.Unmarshal(g.Body, &t); err != nil {
		return nil, nil, err
	}
	return t.Tree, g, nil
}
