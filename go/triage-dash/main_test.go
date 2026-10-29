package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStage(t *testing.T) {
	cases := map[string][]string{
		"new":                {"triage"},
		"awaiting-ai-fix":    {"triaged", "repo-x"},
		"mr+verified":        {"triaged", "ai-mr", "ai-verified"},
		"bailed":             {"ai-fix-failed", "ai-fix"},
		"done+verify-failed": {"ai-done", "ai-mr", "ai-verify-failed"},
		"analysis-failed":    {"triage-failed"},
	}
	for want, ls := range cases {
		if got := stage(ls); got != want {
			t.Errorf("stage(%v) = %q, want %q", ls, got, want)
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
