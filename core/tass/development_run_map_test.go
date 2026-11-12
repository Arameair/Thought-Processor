package tass

// Stage 3A UX repair: the Observe map's connections are decided from recorded correlations, never from
// two blocks merely both being RECORDED.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func mapLink(t *testing.T, r DevelopmentRun, from, to string) ProvenanceLink {
	t.Helper()
	for _, l := range r.Links {
		if l.From == from && l.To == to {
			return l
		}
	}
	t.Fatalf("no link %s→%s in %+v", from, to, r.Links)
	return ProvenanceLink{}
}

func mapBlock(t *testing.T, r DevelopmentRun, id string) ProvenanceBlock {
	t.Helper()
	for _, b := range append(append([]ProvenanceBlock{}, r.Blocks...), r.Extra...) {
		if b.ID == id {
			return b
		}
	}
	t.Fatalf("no block %s", id)
	return ProvenanceBlock{}
}

var mapOrder = []string{"thought", "attention", "authorization", "model_request", "model_reply", "saved_note", "verification"}

func checkMapShape(t *testing.T, r DevelopmentRun) {
	t.Helper()
	if len(r.Blocks) != 7 || len(r.Links) != 6 || len(r.Extra) != 3 {
		t.Fatalf("map shape: %d blocks, %d links, %d extra", len(r.Blocks), len(r.Links), len(r.Extra))
	}
	st := map[string]ProvenanceStage{}
	for _, s := range r.Stages {
		st[s.Stage] = s
	}
	for i, b := range r.Blocks {
		if b.ID != mapOrder[i] || len(b.Stages) == 0 || b.Summary == "" {
			t.Fatalf("block %d: %+v", i, b)
		}
		for _, sid := range b.Stages {
			if _, ok := st[sid]; !ok {
				t.Fatalf("block %s names unknown stage %s", b.ID, sid)
			}
			// A block is never stronger than any stage it groups.
			if b.Status == "RECORDED" && st[sid].Status != "RECORDED" && st[sid].Status != "CODE-DEFINED" && b.ID != "saved_note" {
				t.Errorf("block %s RECORDED but its stage %s is %s", b.ID, sid, st[sid].Status)
			}
		}
		if i < 6 && (r.Links[i].From != b.ID || r.Links[i].To != mapOrder[i+1]) {
			t.Fatalf("link %d: %+v", i, r.Links[i])
		}
	}
	for _, l := range r.Links {
		switch l.Status {
		case "CONNECTED":
			if l.Missing != "" {
				t.Errorf("%s→%s CONNECTED but says something is missing: %q", l.From, l.To, l.Missing)
			}
			// A connection never joins a block with nothing recorded.
			if mapBlock(t, r, l.From).Status == "NOT RECORDED" || mapBlock(t, r, l.To).Status == "NOT RECORDED" {
				t.Errorf("%s→%s CONNECTED to a NOT RECORDED block", l.From, l.To)
			}
		case "PARTIAL", "NOT ESTABLISHED":
			if l.Missing == "" {
				t.Errorf("%s→%s %s must say what is missing", l.From, l.To, l.Status)
			}
		default:
			t.Errorf("%s→%s unknown status %q", l.From, l.To, l.Status)
		}
		if l.Basis == "" {
			t.Errorf("%s→%s has no basis", l.From, l.To)
		}
	}
	// Every stage stays reachable: the map groups stages, it never drops one.
	seen := map[string]bool{}
	for _, b := range append(append([]ProvenanceBlock{}, r.Blocks...), r.Extra...) {
		for _, s := range b.Stages {
			seen[s] = true
		}
	}
	if len(seen) != len(r.Stages) {
		t.Errorf("the map covers %d of %d stages", len(seen), len(r.Stages))
	}
}

func TestDevelopmentRunMapConnectsOnlyOnRecordedEvidence(t *testing.T) {
	c, _, re, nid, _ := processedRun(t)
	reconciled(re)
	r, err := c.DevelopmentRun(nid)
	if err != nil {
		t.Fatal(err)
	}
	checkMapShape(t, r)
	for _, l := range r.Links {
		if l.Status != "CONNECTED" {
			t.Errorf("complete, verified run: %s→%s is %s (%s)", l.From, l.To, l.Status, l.Missing)
		}
	}
	if r.Note == "" || !strings.Contains(r.Stages[7].Text, r.Note) {
		t.Errorf("complete run: outcome Note %q not the saved Note", r.Note)
	}
	for _, b := range r.Blocks {
		if b.Status == "NOT RECORDED" {
			t.Errorf("complete run: block %s NOT RECORDED", b.ID)
		}
	}
}

func TestDevelopmentRunMapShowsGapsNotInventedConnections(t *testing.T) {
	val := func(rec map[string]any, k string) map[string]any {
		return rec[k].(map[string]any)["value"].(map[string]any)
	}
	type want map[string]string // "from>to" → status, or "block:id" → status
	cases := []struct {
		name string
		hook func(rec map[string]any) (int, any)
		set  func(re *runEngine, nid string)
		want want
	}{
		{"HTTP attempt names another request: authorization and model request both RECORDED, not joined", func(rec map[string]any) (int, any) {
			val(rec, "http")["request_id"] = "mr_other999"
			return 200, rec
		}, nil, want{"authorization>model_request": "NOT ESTABLISHED", "block:authorization": "RECORDED", "block:model_request": "RECORDED"}},
		{"reply is digest-bound but does not contain the saved Note: reply and Note both RECORDED, not joined", func(rec map[string]any) (int, any) {
			text := "NOTE: an entirely different idea\nSTATE: keep"
			h := sha256.Sum256([]byte(text))
			rec["model_request"].(map[string]any)["text"] = text
			val(rec, "response")["reply_sha256"] = hex.EncodeToString(h[:])
			return 200, rec
		}, nil, want{"model_reply>saved_note": "PARTIAL", "block:model_reply": "RECORDED", "block:saved_note": "RECORDED", "model_request>model_reply": "CONNECTED"}},
		{"reply does not match the run's digest", func(rec map[string]any) (int, any) {
			rec["model_request"].(map[string]any)["text"] = "NOTE: forged\nSTATE: keep"
			return 200, rec
		}, nil, want{"model_request>model_reply": "PARTIAL", "model_reply>saved_note": "PARTIAL"}},
		{"no reply digest", func(rec map[string]any) (int, any) {
			delete(val(rec, "response"), "reply_sha256")
			return 200, rec
		}, nil, want{"model_request>model_reply": "NOT ESTABLISHED", "block:model_reply": "NOT RECORDED"}},
		{"decision redeemed differs", func(rec map[string]any) (int, any) {
			val(rec, "claim")["identity_decision"] = "dec_other"
			return 200, rec
		}, nil, want{"attention>authorization": "PARTIAL", "authorization>model_request": "CONNECTED"}},
		{"attention digest differs", func(rec map[string]any) (int, any) {
			val(rec, "claim")["attention_sha256"] = strings.Repeat("0", 64)
			return 200, rec
		}, nil, want{"attention>authorization": "PARTIAL", "block:attention": "PARTIAL"}},
		{"no HTTP attempt", func(rec map[string]any) (int, any) {
			rec["http"] = map[string]any{"present": false, "readable": false}
			rec["response"] = map[string]any{"present": false, "readable": false}
			return 200, rec
		}, nil, want{"authorization>model_request": "NOT ESTABLISHED", "model_request>model_reply": "NOT ESTABLISHED"}},
		{"the Engine never claimed it", func(rec map[string]any) (int, any) {
			return 404, map[string]string{"error": "not found: never claimed"}
		}, nil, want{"thought>attention": "PARTIAL", "attention>authorization": "PARTIAL", "authorization>model_request": "NOT ESTABLISHED",
			"model_request>model_reply": "NOT ESTABLISHED", "model_reply>saved_note": "NOT ESTABLISHED", "block:model_request": "NOT RECORDED"}},
		{"duplicate current Note", nil, func(re *runEngine, nid string) {
			re.extraNotes = []map[string]any{{"at": "2026-11-08T09:05:00Z", "text": "SECOND-CURRENT-NOTE", "by": "Nemotron (owner-nudge)",
				"worker_request_id": "mr_" + nid[6:14], "source_nudge_id": nid}}
		}, want{"model_reply>saved_note": "NOT ESTABLISHED", "saved_note>verification": "PARTIAL", "block:saved_note": "PARTIAL"}},
		{"Note no longer stored", nil, func(re *runEngine, nid string) { re.hideNotes = true },
			want{"model_reply>saved_note": "NOT ESTABLISHED", "saved_note>verification": "PARTIAL", "block:saved_note": "NOT RECORDED"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, _, re, nid, _ := processedRun(t)
			reconciled(re)
			re.mu.Lock()
			if tc.hook != nil {
				re.recordHook = func(_ string, rec map[string]any) (int, any) { return tc.hook(rec) }
			}
			if tc.set != nil {
				tc.set(re, nid)
			}
			re.mu.Unlock()
			r, err := c.DevelopmentRun(nid)
			if err != nil {
				t.Fatal(err)
			}
			checkMapShape(t, r)
			for k, w := range tc.want {
				var got string
				if strings.HasPrefix(k, "block:") {
					got = mapBlock(t, r, strings.TrimPrefix(k, "block:")).Status
				} else {
					ft := strings.SplitN(k, ">", 2)
					got = mapLink(t, r, ft[0], ft[1]).Status
				}
				if got != w {
					b, _ := json.Marshal(r.Links)
					t.Errorf("%s = %s, want %s\n%s", k, got, w, b)
				}
			}
			if mapBlock(t, r, "saved_note").Status != "RECORDED" && r.Note != "" {
				t.Errorf("outcome Note %q shown without exactly one current Note", r.Note)
			}
			raw, _ := json.Marshal(r)
			if !r.Verified && (strings.Contains(string(raw), "exactly one") || strings.Contains(string(raw), "verified by Core")) {
				t.Errorf("over-claim in %s", raw)
			}
		})
	}
}
