package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUIFixState(t *testing.T) {
	for _, c := range []struct {
		labels []string
		want   string
	}{
		{[]string{"ui-fix"}, "new"},
		{[]string{"ui-fix", "ui-fix-wip"}, "wip"},
		{[]string{"ui-fix-test", "ui-fix", "frontend"}, "test"},
		{[]string{"ui-fix", "ui-fix-rejected"}, "rejected"},
		{[]string{"ui-fix", "ui-fix-error"}, "error"},
	} {
		if got := uiFixState(c.labels); got != c.want {
			t.Errorf("uiFixState(%v) = %q, want %q", c.labels, got, c.want)
		}
	}
}

func TestUIFixLocal(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "DEV-1")
	os.MkdirAll(key, 0o755)
	os.WriteFile(filepath.Join(key, "meta.json"), []byte(`{"mr": 3412, "mrUrl": "https://gitlab.com/x/-/merge_requests/3412", "round": 2}`), 0o644)
	os.WriteFile(filepath.Join(key, "agent-1.json"), []byte(`{"total_cost_usd": 0.07}`), 0o644)
	os.WriteFile(filepath.Join(key, "agent-2.json"), []byte(`{"total_cost_usd": 0.05}`), 0o644)
	os.WriteFile(filepath.Join(key, "agent-3.json"), []byte(`claude failed: not json`), 0o644)
	m, cost := uiFixLocal(dir, "DEV-1")
	if m.MR != 3412 || m.Round != 2 || m.MRURL == "" {
		t.Errorf("meta = %+v", m)
	}
	if cost < 0.119 || cost > 0.121 {
		t.Errorf("cost = %v, want 0.12", cost)
	}
	if m, cost := uiFixLocal(dir, "DEV-2"); m.MR != 0 || cost != 0 {
		t.Errorf("missing dir: %+v %v", m, cost)
	}
}
