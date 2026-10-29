package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStage(t *testing.T) {
	cases := map[string][]string{
		"new":             {"triage"},
		"plan gate":       {"triaged", "repo-x"},
		"implementing":    {"triaged", "ai-fix"},
		"mr":              {"triaged", "ai-mr"},
		"verified":        {"triaged", "ai-mr", "ai-verified"},
		"bailed":          {"ai-fix-failed", "ai-fix"},
		"verify failed":   {"ai-done", "ai-mr", "ai-verify-failed"},
		"analysis failed": {"triage-failed"},
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
		{row{stage: "bailed", jira: true}, gNeeds},
		{row{stage: "mr", jira: true}, gFlight},
		{row{stage: "RC", parked: true, jira: true}, gParked},
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
	os.WriteFile(p, []byte("🔧 Bash {}\n✅ exit error=false cost=$0.5\n  VERDICT: fail\n💬 hi\n✅ exit error=false cost=$0.25\n  VERDICT: pass\n"), 0o644)
	var r row
	parseLog(&r, p)
	if r.cost != 0.75 || r.verdict != "pass" || r.last != "✅ exit error=false cost=$0.25" {
		t.Errorf("got %+v", r)
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
	pipe := func(s string) *struct{ Status string } { return &struct{ Status string }{s} }
	cases := map[string]mrRaw{
		"merged":        {State: "merged", BlockingDiscussionsResolved: true},
		"conflict":      {State: "opened", HasConflicts: true, BlockingDiscussionsResolved: true},
		"auto-rebase":   {State: "opened", Draft: true, HasConflicts: true, BlockingDiscussionsResolved: true},
		"rebase queued": {State: "opened", HasConflicts: true, Labels: []string{"ai-rebase"}, BlockingDiscussionsResolved: true},
		"ci failed":     {State: "opened", HeadPipeline: pipe("failed"), BlockingDiscussionsResolved: true},
		"open threads":  {State: "opened"},
		"draft":         {State: "opened", Draft: true, BlockingDiscussionsResolved: true, HeadPipeline: pipe("success")},
		"ci running":    {State: "opened", BlockingDiscussionsResolved: true, HeadPipeline: pipe("running")},
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
	m := model{mrs: map[string]mrInfo{"u": {label: "conflict", icon: "💥", color: "31"}}}
	cases := map[string]string{"waiting": "conflict: b", "failed": "rebase failed", "pending": "rebase pending", "": "rebase pending"}
	for state, want := range cases {
		if _, _, got := m.look(row{stage: "mr", mr: "u", conflict: state}); got != want {
			t.Errorf("conflict %q: got %q, want %q", state, got, want)
		}
	}
}
