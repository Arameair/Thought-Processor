package tass

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Arameair/Thought-Processor/core/engine"
)

// A teaching draft is local data, not a GitHub issue or a ChatGPT request.
// Identity must authorize a separately implemented, scoped publisher before delivery.
type teachingDraft struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Topic     string    `json:"topic"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
}

var teachingChatRequest = regexp.MustCompile(`(?i)^\s*(?:tass[,!]?\s+)?ask chatgpt to\s+(.+?)\s*$`)
var teachingMention = regexp.MustCompile(`(?i)^\s*@ask-chatgpt\s*[:,-]?\s+(.+?)\s*$`)

func requestedTeachingTopic(message string) string {
	for _, re := range []*regexp.Regexp{teachingChatRequest, teachingMention} {
		if match := re.FindStringSubmatch(message); len(match) == 2 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

// createTeachingDraft records one bounded, owner-visible request without sending it.
// A successful draft must NEVER be presented as though GitHub or ChatGPT received it.
func (c *Core) createTeachingDraft(topic string) (string, error) {
	topic = strings.Join(strings.Fields(topic), " ")
	if len(topic) < 8 || len(topic) > 500 {
		return "", errors.New("teaching question must be 8 to 500 characters")
	}
	id := "teach_" + engine.NewRequestID()
	item := teachingDraft{
		ID: id, CreatedAt: c.now().UTC(), From: "owner-chat-request",
		To: "ChatGPT", Topic: topic, Status: "DRAFT_NOT_SENT",
		Note: "A reviewed Identity-authorized GitHub writer and response channel are not yet connected.",
	}
	data, err := json.Marshal(item)
	if err != nil {
		return "", err
	}
	c.fileMu.Lock()
	defer c.fileMu.Unlock()
	file, err := os.OpenFile(c.file("teaching-requests.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("save teaching draft: %w", err)
	}
	_, writeErr := file.Write(append(data, '\n'))
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return "", fmt.Errorf("save teaching draft: %w", writeErr)
	}
	if closeErr != nil {
		return "", fmt.Errorf("close teaching draft: %w", closeErr)
	}
	return id, nil
}
