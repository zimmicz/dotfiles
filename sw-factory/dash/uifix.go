package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var uiFixStates = []string{"ui-fix-wip", "ui-fix-deploy", "ui-fix-test", "ui-fix-merge", "ui-fix-done", "ui-fix-rejected", "ui-fix-error"}

type uiFixMeta struct {
	MR    int    `json:"mr"`
	MRURL string `json:"mrUrl"`
	Round int    `json:"round"`
}

type uiFixRow struct {
	Key, Summary, State, Status, StatusCat, Reporter, MRURL, Updated string
	MR, Round                                                        int
	Cost                                                             float64
}

func uiFixState(ls []string) string {
	for _, s := range uiFixStates {
		if slices.Contains(ls, s) {
			return strings.TrimPrefix(s, "ui-fix-")
		}
	}
	return "new"
}

func uiFixStateDir() string {
	if d := os.Getenv("UI_FIX_STATE"); d != "" {
		return d
	}
	return filepath.Join(os.Getenv("HOME"), ".local/state/ui-fix")
}

func uiFixLocal(dir, key string) (m uiFixMeta, cost float64) {
	if b, err := os.ReadFile(filepath.Join(dir, key, "meta.json")); err == nil {
		json.Unmarshal(b, &m)
	}
	files, _ := filepath.Glob(filepath.Join(dir, key, "agent-*.json"))
	for _, f := range files {
		var a struct {
			Cost float64 `json:"total_cost_usd"`
		}
		if b, err := os.ReadFile(f); err == nil && json.Unmarshal(b, &a) == nil {
			cost += a.Cost
		}
	}
	return
}

func fetchUIFix(dir string) ([]uiFixRow, error) {
	var out struct {
		Issues []struct {
			Key    string
			Fields struct {
				Summary  string
				Labels   []string
				Updated  string
				Reporter struct{ DisplayName string }
				Status   struct {
					Name           string
					StatusCategory struct{ Key string }
				}
			}
		}
	}
	jql := "project = DEV AND labels = ui-fix AND (statusCategory != Done OR updated >= -7d) ORDER BY updated DESC"
	if err := jiraReq("GET", "/rest/api/3/search/jql?fields=labels,summary,updated,status,reporter&maxResults=100&jql="+url.QueryEscape(jql), nil, &out); err != nil {
		return nil, err
	}
	rows := make([]uiFixRow, 0, len(out.Issues))
	for _, i := range out.Issues {
		m, cost := uiFixLocal(dir, i.Key)
		rows = append(rows, uiFixRow{
			Key: i.Key, Summary: i.Fields.Summary, State: uiFixState(i.Fields.Labels),
			Status: i.Fields.Status.Name, StatusCat: i.Fields.Status.StatusCategory.Key,
			Reporter: i.Fields.Reporter.DisplayName, Updated: i.Fields.Updated,
			MR: m.MR, MRURL: m.MRURL, Round: m.Round, Cost: cost,
		})
	}
	return rows, nil
}
