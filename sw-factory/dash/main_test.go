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
		{row{stage: "to test", jira: true}, gNeeds},
		{row{stage: "to test", parked: true, jira: true}, gParked},
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

func TestParkedStage(t *testing.T) {
	cases := []struct {
		status, cat, want string
		parked            bool
	}{
		{"On nightly", "indeterminate", "nightly", true},
		{"Release candidate", "indeterminate", "RC", true},
		{"Rejected", "indeterminate", "done", true},
		{"X", "done", "done", true},
		{"To test", "indeterminate", "", false},
	}
	for _, c := range cases {
		if got, ok := parkedStage(c.status, c.cat); got != c.want || ok != c.parked {
			t.Errorf("parkedStage(%q,%q) = %q,%v", c.status, c.cat, got, ok)
		}
	}
}

func TestCachedLog(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DEV-1.log")
	os.WriteFile(p, []byte("✅ exit error=false cost=$0.5\n"), 0o644)
	run := func() float64 {
		st, _ := os.Stat(p)
		var r row
		cachedLog(&r, p, st)
		return r.cost
	}
	if a, b := run(), run(); a != 0.5 || b != 0.5 {
		t.Fatalf("got %v %v", a, b)
	}
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("✅ exit error=false cost=$0.25\n")
	f.Close()
	if c := run(); c != 0.75 {
		t.Errorf("after append got %v", c)
	}
}

func TestOrder(t *testing.T) {
	t0 := time.Now().Add(-time.Hour)
	keys := func(rs []row) string {
		var ks []string
		for _, r := range rs {
			ks = append(ks, r.key)
		}
		return strings.Join(ks, " ")
	}
	rs := []row{
		{key: "T-1", stage: "mr", jira: true, seen: t0},
		{key: "T-2", stage: "mr", jira: true, seen: t0.Add(time.Minute)},
		{key: "T-3", stage: "plan gate", jira: true, seen: t0},
	}
	order(rs, t0.Add(2*time.Minute))
	if got := keys(rs); got != "T-3 T-2 T-1" {
		t.Fatalf("first order %q", got)
	}
	rs[2].seen = t0.Add(30 * time.Minute)
	order(rs, t0.Add(30*time.Minute))
	if got := keys(rs); got != "T-3 T-2 T-1" {
		t.Errorf("log write reordered: %q", got)
	}
	for i := range rs {
		if rs[i].key == "T-1" {
			rs[i].stage = "reviewed"
		}
	}
	order(rs, t0.Add(40*time.Minute))
	if got := keys(rs); got != "T-1 T-3 T-2" {
		t.Errorf("group change: %q", got)
	}
}

func TestParseLog(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DEV-1.log")
	os.WriteFile(p, []byte("🔧 Bash {}\n✅ exit error=false cost=$0.5\n  VERDICT: fail\n13:20:15 💬 hi\n13:20:16 🆔 session s — reopen: cd /x && claude --resume s — transcript ~/.claude/projects/-x/s.jsonl\n13:21:00 ✅ exit error=false cost=$0.25\n  VERDICT: pass\n"), 0o644)
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
		{"open without http url", "POST", "/api/open", "127.0.0.1:7777", "http://127.0.0.1:7777", true, 400},
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
	os.WriteFile(filepath.Join(stateDir, "DEV-1.verify.md"), []byte("VERDICT: fail\nS1 First: fail — broke. Screenshot: /x.jpeg\nVideo: /x/DEV-1.verify-media/DEV-1.short.webm\n"), 0o644)
	os.MkdirAll(filepath.Join(stateDir, "holdout"), 0o755)
	os.WriteFile(filepath.Join(stateDir, "holdout", "DEV-1.md"), []byte("S1: First\nGiven: a\nThen: b\n\nS2: Other\nGiven: c\n"), 0o644)
	m := mediaFiles("DEV-1")
	if len(m) != 2 || !m[0].Current || m[0].Name != "DEV-1.short.webm" || m[1].Current {
		t.Fatalf("current video not first: %+v", m)
	}
	ch := m[0].Chapters
	if len(ch) != 2 || ch[1].Start != 9.2 || ch[1].Title != "S1 · First" || ch[1].Verdict != "fail" || ch[0].Verdict != "" || ch[1].Text != "Given: a\nThen: b" || ch[0].Text != "" || ch[1].Result != "broke." {
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

func TestParseRunsHeaders(t *testing.T) {
	p := filepath.Join(t.TempDir(), "DEV-9.log")
	os.WriteFile(p, []byte("=== DEV-1 (repo) 2026-11-12T13:49:31+01:00\n=== review (incremental) DEV-2 (x:1) 2026-11-12T13:49:31+01:00\n=== holdout amend DEV-3 2026-11-12T13:49:31+01:00\n=== conflict https://gitlab.com/a/-/merge_requests/1 onto develop 2026-11-12T13:49:31+01:00\n=== deps-triage 2026-11-12T13:49:31+01:00: DEV-4 from repo\n=== verify-ui DEV-5 (x:1, env e) 2026-11-12T13:49:31+01:00\n"), 0o644)
	want := []string{"analyze DEV-1", "review (incremental) DEV-2", "holdout amend DEV-3", "conflict DEV-9", "deps-triage DEV-4", "verify-ui DEV-5"}
	r := parseRuns(p)
	if len(r) != len(want) {
		t.Fatalf("got %d runs", len(r))
	}
	for i, w := range want {
		if got := r[i].stage + " " + r[i].key; got != w {
			t.Errorf("run %d = %q, want %q", i, got, w)
		}
	}
}

func TestParseUnits(t *testing.T) {
	got := parseUnits("sw-factory-DEV-3536-review.service loaded active running /home/zimmi/bin/factory-worker --run review\nsw-factory-DEV-1-rfix.service loaded active running x\nsw-factory-feat-x-y-conflict.service loaded active running x\n")
	want := map[string]string{"DEV-3536": "reviewing", "DEV-1": "fixing review", "feat-x-y": "rebasing"}
	if !maps.Equal(got, want) {
		t.Errorf("parseUnits = %v, want %v", got, want)
	}
}

func TestAcceptProposal(t *testing.T) {
	old := stateDir
	stateDir = t.TempDir()
	defer func() { stateDir = old }()
	for _, d := range []string{"holdout", "holdout-proposed", "notes"} {
		os.MkdirAll(filepath.Join(stateDir, d), 0o755)
	}
	os.WriteFile(filepath.Join(stateDir, "holdout", "DEV-1.md"), []byte("## S1: a\nThen: old\n"), 0o644)
	os.WriteFile(filepath.Join(stateDir, "holdout-proposed", "DEV-1.md"), []byte("## S1: a\nThen: new\n"), 0o644)
	os.WriteFile(filepath.Join(stateDir, "holdout-proposed", "DEV-1.why"), []byte("S1 changed: ticket says new\nS2 kept: fine\n"), 0o644)
	os.WriteFile(filepath.Join(stateDir, "notes", "DEV-1.md"), []byte("## Analysis\nx\n## Holdout changes\n2026-01-01 S4 changed (accepted by Michal)\n"), 0o644)
	if d := proposalDiff("DEV-1"); !strings.Contains(d, "-Then: old") || !strings.Contains(d, "+Then: new") {
		t.Fatalf("diff: %q", d)
	}
	got, err := acceptProposal("DEV-1", time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC))
	if err != nil || len(got) != 1 {
		t.Fatalf("accept: %v %v", got, err)
	}
	b, _ := os.ReadFile(filepath.Join(stateDir, "holdout", "DEV-1.md"))
	n, _ := os.ReadFile(filepath.Join(stateDir, "notes", "DEV-1.md"))
	if string(b) != "## S1: a\nThen: new\n" || len(proposedFiles("DEV-1")) != 0 || fileExists(filepath.Join(stateDir, "holdout-proposed", "DEV-1.why")) {
		t.Fatalf("holdout %q, proposal left", b)
	}
	if string(n) != "## Analysis\nx\n## Holdout changes\n2026-11-18 S1 changed (accepted by Michal)\n2026-01-01 S4 changed (accepted by Michal)\n" {
		t.Fatalf("notes %q", n)
	}
	if _, err := acceptProposal("DEV-1", time.Now()); err == nil {
		t.Fatal("second accept should fail")
	}
}

func TestUnverifiedIDs(t *testing.T) {
	old := stateDir
	stateDir = t.TempDir()
	defer func() { stateDir = old }()
	if got := unverifiedIDs("DEV-1"); got != "" {
		t.Fatalf("no file: got %q", got)
	}
	os.MkdirAll(filepath.Join(stateDir, "unverified"), 0o755)
	os.WriteFile(filepath.Join(stateDir, "unverified", "DEV-1.md"), []byte("S7 Group: blocked — DATA: none\nS8 Machine: blocked — DATA: none\n"), 0o644)
	if got := unverifiedIDs("DEV-1"); got != "S7,S8" {
		t.Fatalf("got %q", got)
	}
}
