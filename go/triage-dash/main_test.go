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
		{row{running: true, labels: []string{}}, 0},
		{row{stage: "plan gate", labels: []string{}}, 1},
		{row{stage: "bailed", labels: []string{}}, 1},
		{row{stage: "mr", labels: []string{}}, 2},
		{row{stage: "RC", parked: true, labels: []string{}}, 3},
		{row{stage: "-"}, 4},
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
