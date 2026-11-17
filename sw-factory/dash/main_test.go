package main

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUploadImages(t *testing.T) {
	dir := "/tmp/triage-img-DEV-1-42"
	var got [][2]string
	text := "see !" + dir + "/img-1.png! and !" + dir + "/img-2.png!, keep !other.png!"
	out, err := uploadImages(text, dir, "reply-x", func(p, name string) error {
		got = append(got, [2]string{p, name})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "see !reply-x-img-1.png! and !reply-x-img-2.png!, keep !other.png!"; out != want {
		t.Errorf("got %q, want %q", out, want)
	}
	if len(got) != 2 || got[1] != [2]string{dir + "/img-2.png", "reply-x-img-2.png"} {
		t.Errorf("attach calls %v", got)
	}
}

func TestStage(t *testing.T) {
	cases := map[string][]string{
		"new":             {"triage"},
		"plan gate":       {"triaged", "repo-x"},
		"implementing":    {"triaged", "ai-fix"},
		"mr":              {"triaged", "ai-mr"},
		"verified":        {"triaged", "ai-mr", "ai-verified"},
		"quiz":            {"triaged", "ai-mr", "ai-reviewed", "ai-verified"},
		"reviewed":        {"triaged", "ai-mr", "ai-reviewed", "ai-verified", "ai-quizzed"},
		"re-verify":       {"triaged", "ai-mr", "ai-reviewed"},
		"review fix":      {"triaged", "ai-mr", "ai-review-retry", "ai-verified"},
		"review findings": {"triaged", "ai-mr", "ai-review-findings", "ai-verified"},
		"bailed":          {"ai-fix-failed", "ai-fix"},
		"verify failed":   {"ai-done", "ai-mr", "ai-verify-failed"},
		"analysis failed": {"triage-failed"},
		"questions":       {"needs-answer", "ai-fix"},
	}
	for want, ls := range cases {
		if got := stage(ls); got != want {
			t.Errorf("stage(%v) = %q, want %q", ls, got, want)
		}
	}
}

func TestGroup(t *testing.T) {
	cases := []struct {
		r    row
		want int
	}{
		{row{running: true, jira: true}, gRunning},
		{row{stage: "plan gate", jira: true}, gNeeds},
		{row{stage: "questions", jira: true}, gNeeds},
		{row{stage: "bailed", jira: true}, gNeeds},
		{row{stage: "mr", jira: true}, gFlight},
		{row{stage: "nightly", parked: true, jira: true}, gParked},
		{row{stage: "todo", jira: true, todo: true}, gTodo},
		{row{stage: "-"}, gOther},
	}
	for _, c := range cases {
		if got := group(c.r); got != c.want {
			t.Errorf("group(%+v) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestParseLog(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DEV-1.log")
	os.WriteFile(p, []byte("🔧 Bash {}\n✅ exit error=false cost=$0.5\n  VERDICT: fail\n13:20:15 💬 hi\n13:20:16 🆔 session s — reopen: cd /x && claude --resume s\n13:21:00 ✅ exit error=false cost=$0.25\n  VERDICT: pass\n"), 0o644)
	var r row
	parseLog(&r, p)
	if r.cost != 0.75 || r.verdict != "pass" || r.last != "✅ exit error=false cost=$0.25" || r.resume != "cd /x && claude --resume s" {
		t.Errorf("got %+v", r)
	}
	if ts, body := splitTS("13:20:15 🔧 x"); ts != "13:20:15" || body != "🔧 x" {
		t.Errorf("splitTS = %q %q", ts, body)
	}
}

func TestRetryAction(t *testing.T) {
	cases := []struct {
		labels      []string
		remove, add string
	}{
		{[]string{"repo-x", "triage-failed"}, "triage-failed", "triage"},
		{[]string{"triaged", "ai-fix-failed"}, "ai-fix-failed", "ai-fix"},
		{[]string{"ai-mr", "ai-verify-failed"}, "ai-verify-failed", ""},
		{[]string{"triaged"}, "", ""},
	}
	for _, c := range cases {
		if rm, add := retryAction(c.labels); rm != c.remove || add != c.add {
			t.Errorf("retryAction(%v) = %q,%q", c.labels, rm, add)
		}
	}
}

func TestWikiToMD(t *testing.T) {
	in := "h3. Poznámka\n{{code}} {{a-b}} je *tučně*, [MR|https://x/1]\n# první\n* odrážka\n{code:js}x(){code}"
	want := "### Poznámka\n`code` `a-b` je **tučně**, [MR](https://x/1)\n1. první\n* odrážka\n\n```\nx()\n```\n"
	if got := wikiToMD(in); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestMRStatus(t *testing.T) {
	pipe := func(s string) *pipeRaw { return &pipeRaw{Status: s} }
	cases := map[string]mrRaw{
		"merged":        {State: "merged", BlockingDiscussionsResolved: true},
		"conflict":      {State: "opened", HasConflicts: true, BlockingDiscussionsResolved: true},
		"auto-rebase":   {State: "opened", Draft: true, HasConflicts: true, BlockingDiscussionsResolved: true},
		"rebase queued": {State: "opened", HasConflicts: true, Labels: []string{"ai-rebase"}, BlockingDiscussionsResolved: true},
		"ci failed":     {State: "opened", HeadPipeline: pipe("failed"), BlockingDiscussionsResolved: true},
		"open threads":  {State: "opened"},
		"draft":         {State: "opened", Draft: true, BlockingDiscussionsResolved: true, HeadPipeline: pipe("success")},
		"ci running":    {State: "opened", BlockingDiscussionsResolved: true, HeadPipeline: pipe("running")},
		"gate pending":  {State: "opened", BlockingDiscussionsResolved: true, HeadPipeline: pipe("running"), Gate: "pending"},
		"gate failed":   {State: "opened", BlockingDiscussionsResolved: true, HeadPipeline: pipe("failed"), Gate: "failed"},
		"approved":      {State: "opened", BlockingDiscussionsResolved: true, Approved: true},
		"in review":     {State: "opened", BlockingDiscussionsResolved: true, ApprovalsLeft: 1, Reviewers: []struct{ Username string }{{"x"}}},
		"no reviewer":   {State: "opened", BlockingDiscussionsResolved: true, ApprovalsLeft: 1},
		"changes asked": {State: "opened", BlockingDiscussionsResolved: true, DetailedMergeStatus: "requested_changes"},
	}
	for want, r := range cases {
		if got := mrStatus(r).label; got != want {
			t.Errorf("mrStatus(%+v) = %q, want %q", r, got, want)
		}
	}
}

func TestLookConflict(t *testing.T) {
	mrs := map[string]mrInfo{"u": {label: "conflict", icon: "💥", color: "31"}}
	cases := map[string]string{"waiting": "conflict: b", "failed": "rebase failed", "pending": "rebase pending", "": "rebase pending"}
	for state, want := range cases {
		if _, _, got := look(row{stage: "mr", mr: "u", conflict: state}, mrs); got != want {
			t.Errorf("conflict %q: got %q, want %q", state, got, want)
		}
	}
}

func TestGuard(t *testing.T) {
	s := &server{mrs: map[string]mrInfo{}, token: "secret-token"}
	h := s.handler()
	call := func(method, path, host, origin string, cookie bool) int {
		r := httptest.NewRequest(method, path, strings.NewReader("{}"))
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		if cookie {
			r.AddCookie(&http.Cookie{Name: "fd", Value: "secret-token"})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	cases := []struct {
		name                       string
		method, path, host, origin string
		cookie                     bool
		want                       int
	}{
		{"no cookie", "GET", "/api/status", "127.0.0.1:7777", "", false, 401},
		{"foreign host", "GET", "/api/status", "evil.example", "", true, 403},
		{"foreign origin", "POST", "/api/action", "127.0.0.1:7777", "http://evil.example", true, 403},
		{"missing origin on post", "POST", "/api/action", "127.0.0.1:7777", "", true, 403},
		{"media traversal", "GET", "/media/DEV-1.verify-media/..%2f..%2fdash.token", "127.0.0.1:7777", "", true, 404},
		{"media wrong dir", "GET", "/media/holdout/DEV-1.jpg", "127.0.0.1:7777", "", true, 404},
		{"token login", "GET", "/?t=secret-token", "127.0.0.1:7777", "", false, 302},
	}
	for _, c := range cases {
		if got := call(c.method, c.path, c.host, c.origin, c.cookie); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestChapters(t *testing.T) {
	old := stateDir
	stateDir = t.TempDir()
	defer func() { stateDir = old }()
	os.MkdirAll(filepath.Join(stateDir, "DEV-1.verify-media", "narrate"), 0o755)
	os.WriteFile(filepath.Join(stateDir, "DEV-1.verify-media", "narrate", "chapters.txt"), []byte(";FFMETADATA1\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=0\nEND=9200\ntitle=DEV-1 · Intro\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=9200\nEND=27892\ntitle=S1 · First\n"), 0o644)
	os.WriteFile(filepath.Join(stateDir, "DEV-1.verify-media", "DEV-1.short.webm"), nil, 0o644)
	os.WriteFile(filepath.Join(stateDir, "DEV-1.verify-media", "S1.webm"), nil, 0o644)
	os.WriteFile(filepath.Join(stateDir, "DEV-1.verify.md"), []byte("VERDICT: fail\nS1 First: fail — broke\nVideo: /x/DEV-1.verify-media/DEV-1.short.webm\n"), 0o644)
	m := mediaFiles("DEV-1")
	if len(m) != 2 || !m[0].Current || m[0].Name != "DEV-1.short.webm" || m[1].Current {
		t.Fatalf("current video not first: %+v", m)
	}
	ch := m[0].Chapters
	if len(ch) != 2 || ch[1].Start != 9.2 || ch[1].Title != "S1 · First" || ch[1].Verdict != "fail" || ch[0].Verdict != "" {
		t.Fatalf("chapters: %+v", ch)
	}
}

func TestParseRuns(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DEV-1.log")
	os.WriteFile(p, []byte("=== implement DEV-1 (repo) 2026-11-12T13:49:31+01:00\n13:49:40 🔧 Bash {\"command\":\"ls\"}\n13:50:00 🔧 Bash {}\n13:59:31 ✅ exit error=false cost=$1.5 turns=20 peak=84K\n"), 0o644)
	r := parseRuns(p)
	if len(r) != 1 || r[0].stage != "implement" || r[0].key != "DEV-1" || r[0].cost != 1.5 || r[0].turns != 20 || r[0].peak != 84 || r[0].tools["Bash"] != 2 || r[0].dur != 10*time.Minute || r[0].fail {
		t.Fatalf("got %+v", r)
	}
}

func TestParseUnits(t *testing.T) {
	got := parseUnits("sw-factory-DEV-3536-review.service loaded active running /home/zimmi/bin/factory-worker --run review\nsw-factory-DEV-1-rfix.service loaded active running x\nsw-factory-feat-x-y-conflict.service loaded active running x\n")
	want := map[string]string{"DEV-3536": "reviewing", "DEV-1": "fixing review", "feat-x-y": "rebasing"}
	if !maps.Equal(got, want) {
		t.Errorf("parseUnits = %v, want %v", got, want)
	}
}
