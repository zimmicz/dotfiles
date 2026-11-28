package main

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	jiraBase = "https://cleverfarm.atlassian.net"
	labels   = "triage, needs-answer, triaged, triage-failed, ai-fix, ai-mr, ai-fix-failed, ai-review-retry, ai-reviewed, ai-review-findings, ai-verify-retry, ai-verified, ai-verify-failed, ai-quizzed, ai-done"
)

var (
	stateDir = filepath.Join(os.Getenv("HOME"), ".local/state/sw-factory")
	costRe   = regexp.MustCompile(`cost=\$([0-9.]+)`)
	verdRe   = regexp.MustCompile(`^\s*VERDICT: (pass|fail|blocked)\b`)
	tsRe     = regexp.MustCompile(`^\d\d:\d\d:\d\d `)
	mrIIDRe  = regexp.MustCompile(`merge_requests/(\d+)`)
	jiraKey  = regexp.MustCompile(`^[A-Z]+-\d+$`)
)

type issue struct {
	Key    string
	Fields struct {
		Summary  string
		Labels   []string
		Updated  string
		Sprint   []struct{ Name, State string } `json:"customfield_10007"`
		Assignee *struct {
			AccountID string `json:"accountId"`
		}
		Status struct {
			Name           string
			StatusCategory struct{ Key string }
		}
	}
}

type row struct {
	key, stage, summary, verdict, mr, last, resume string
	phase                                          string
	cost                                           float64
	seen                                           time.Time
	running, parked, jira, todo, mrNeeds           bool
	conflict                                       string
	labels                                         []string
}

func parkedStage(status, category string) (string, bool) {
	switch {
	case status == "On nightly":
		return "nightly", true
	case status == "Release candidate":
		return "RC", true
	case status == "Rejected":
		return "done", true
	case category == "done":
		return "done", true
	}
	return "", false
}

func stage(ls []string) string {
	has := map[string]bool{}
	for _, l := range ls {
		has[l] = true
	}
	switch {
	case has["triage-failed"]:
		return "analysis failed"
	case has["needs-answer"]:
		return "questions"
	case has["ai-fix-failed"]:
		return "bailed"
	case has["ai-verify-failed"]:
		return "verify failed"
	case has["ai-done"]:
		return "done"
	case has["ai-review-findings"]:
		return "review findings"
	case has["ai-review-retry"]:
		return "review fix"
	case has["ai-reviewed"] && has["ai-verified"] && has["ai-quizzed"]:
		return "reviewed"
	case has["ai-reviewed"] && has["ai-verified"]:
		return "quiz"
	case has["ai-reviewed"]:
		return "re-verify"
	case has["ai-verified"]:
		return "verified"
	case has["ai-mr"]:
		return "mr"
	case has["ai-fix"]:
		return "implementing"
	case has["triaged"]:
		return "plan gate"
	}
	return "new"
}

var stageStyle = map[string]struct{ icon, color string }{
	"plan gate":        {"📋", "33"},
	"to test":          {"👀", "33"},
	"questions":        {"❓", "33"},
	"verified":         {"🧪", "34"},
	"verify failed":    {"❗", "31"},
	"holdout proposal": {"✏️", "33"},
	"analysis failed":  {"❌", "31"},
	"bailed":           {"❌", "31"},
	"implementing":     {"🔨", "36"},
	"mr":               {"🔍", "34"},
	"reviewed":         {"✅", "32"},
	"quiz":             {"🎓", "33"},
	"re-verify":        {"🔁", "36"},
	"review fix":       {"🔧", "36"},
	"review findings":  {"🔶", "33"},
	"new":              {"🆕", "36"},
	"done":             {"🏁", "2"},
	"RC":               {"📦", "2"},
	"nightly":          {"🌙", "2"},
	"-":                {"📄", "2"},
	"todo":             {"📝", "35"},
}

const (
	gRunning = iota
	gNeeds
	gTodo
	gFlight
	gParked
	gOther
)

func group(r row) int {
	switch {
	case r.running:
		return gRunning
	case r.mrNeeds:
		return gNeeds
	case !r.jira:
		return gOther
	case r.todo:
		return gTodo
	case r.parked:
		return gParked
	case slices.Contains([]string{"plan gate", "questions", "quiz", "reviewed", "verify failed", "holdout proposal", "review findings", "analysis failed", "bailed", "to test"}, r.stage):
		return gNeeds
	}
	return gFlight
}

func search(jql string) ([]issue, error) {
	var out struct{ Issues []issue }
	return out.Issues, jiraReq("GET", "/rest/api/3/search/jql?fields=labels,summary,updated,status,assignee,customfield_10007&maxResults=100&jql="+url.QueryEscape(jql), nil, &out)
}

func fetchJira() ([]issue, []issue, error) {
	issues, err := search("project = DEV AND (assignee = currentUser() OR assignee is EMPTY) AND labels in (" + labels + ") AND (statusCategory != Done OR updated >= -3d) ORDER BY updated DESC")
	if err != nil {
		return nil, nil, err
	}
	todos, err := search("assignee = currentUser() AND sprint in openSprints() AND statusCategory = \"To Do\" AND (labels is EMPTY OR labels not in (" + labels + ")) ORDER BY rank")
	return issues, todos, err
}

func repos() []string {
	dirs, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), "work/cleverfarm/repos/*/.git"))
	var out []string
	for _, d := range dirs {
		out = append(out, filepath.Base(filepath.Dir(d)))
	}
	return out
}

func feed(key, repo string) error {
	ops := []map[string]string{{"add": "triage"}, {"add": "repo-" + repo}}
	return jiraReq("PUT", "/rest/api/2/issue/"+key, map[string]any{"update": map[string]any{"labels": ops}}, nil)
}

var newTypes = []string{"Technical", "Task", "Bug"}

func product(repo string) string {
	switch {
	case strings.HasPrefix(repo, "can-"):
		return "11707"
	case repo == "cf-agroevidence":
		return "11711"
	case repo == "cf-iot" || repo == "irrigation":
		return "11712"
	case repo == "cf-satellite":
		return "11713"
	case strings.HasPrefix(repo, "telematics-"):
		return "11714"
	}
	return "11710"
}

func createTicket(title, desc, typ, repo, epic string) (string, error) {
	me := myAccount()
	if me == "" {
		return "", fmt.Errorf("jira myself lookup failed")
	}
	var sp struct{ Values []struct{ ID int } }
	if err := jiraReq("GET", "/rest/agile/1.0/board/198/sprint?state=active", nil, &sp); err != nil {
		return "", err
	}
	f := map[string]any{
		"project": map[string]string{"key": "DEV"}, "issuetype": map[string]string{"name": typ}, "summary": title, "description": desc,
		"assignee": map[string]string{"accountId": me}, "customfield_13543": []map[string]string{{"id": product(repo)}}, "labels": []string{"repo-" + repo},
	}
	if len(sp.Values) > 0 {
		f["customfield_10007"] = sp.Values[0].ID
	}
	if epic != "" {
		f["parent"] = map[string]string{"key": epic}
	}
	var out struct{ Key string }
	if err := jiraReq("POST", "/rest/api/2/issue", map[string]any{"fields": f}, &out); err != nil {
		return "", err
	}
	if out.Key == "" {
		return "", fmt.Errorf("jira create returned no key")
	}
	return out.Key, nil
}

type mrRaw struct {
	State                       string
	Draft                       bool
	DetailedMergeStatus         string `json:"detailed_merge_status"`
	HasConflicts                bool   `json:"has_conflicts"`
	BlockingDiscussionsResolved bool   `json:"blocking_discussions_resolved"`
	Reviewers                   []struct{ Username string }
	HeadPipeline                *pipeRaw `json:"head_pipeline"`
	Gate                        string   `json:"-"`
	Approved                    bool
	ApprovalsLeft               int                                        `json:"approvals_left"`
	ApprovedBy                  []struct{ User struct{ Username string } } `json:"approved_by"`
	Labels                      []string
}

type pipeRaw struct {
	ID          int64
	Status, Sha string
}

type mrInfo struct {
	label, icon, color, detail string
	final                      bool
}

var mrURLRe = regexp.MustCompile(`gitlab\.com/(.+)/-/merge_requests/(\d+)`)

func mrStatus(r mrRaw) mrInfo {
	pipe := ""
	if r.HeadPipeline != nil {
		pipe = r.HeadPipeline.Status
	}
	var reviewers []string
	for _, u := range r.Reviewers {
		reviewers = append(reviewers, u.Username)
	}
	detail := fmt.Sprintf("ci %s · approvals %d, %d left", or(pipe, "none"), len(r.ApprovedBy), r.ApprovalsLeft)
	if len(reviewers) > 0 {
		detail += " · reviewers " + strings.Join(reviewers, ", ")
	}
	if r.Gate != "" {
		detail += " · gate " + r.Gate
	}
	in := mrInfo{detail: detail}
	switch {
	case r.State == "merged":
		in.label, in.icon, in.color, in.final = "merged", "🟣", "35", true
	case r.State == "closed":
		in.label, in.icon, in.color, in.final = "closed", "⛔", "2", true
	case r.HasConflicts || r.DetailedMergeStatus == "conflict" || r.DetailedMergeStatus == "need_rebase":
		in.label, in.icon, in.color = "conflict", "💥", "31"
		if slices.Contains(r.Labels, "ai-rebase") {
			in.label, in.icon, in.color = "rebase queued", "🔧", "36"
		} else if r.Draft && len(r.ApprovedBy) == 0 {
			in.label, in.icon, in.color = "auto-rebase", "🔧", "36"
		}
	case r.Gate == "failed":
		in.label, in.icon, in.color = "gate failed", "🚫", "31"
	case pipe == "failed":
		in.label, in.icon, in.color = "ci failed", "🔴", "31"
	case r.DetailedMergeStatus == "requested_changes":
		in.label, in.icon, in.color = "changes asked", "🔁", "31"
	case !r.BlockingDiscussionsResolved:
		in.label, in.icon, in.color = "open threads", "💬", "33"
	case r.Draft:
		in.label, in.icon, in.color = "draft", "🚧", "33"
	case r.Gate == "pending":
		in.label, in.icon, in.color = "gate pending", "🔒", "36"
	case pipe == "running" || pipe == "pending" || pipe == "created":
		in.label, in.icon, in.color = "ci running", "⏳", "36"
	case r.Approved && r.ApprovalsLeft == 0:
		in.label, in.icon, in.color = "approved", "👍", "32"
	case len(reviewers) > 0:
		in.label, in.icon, in.color = "in review", "👀", "36"
	default:
		in.label, in.icon, in.color = "no reviewer", "📭", "33"
	}
	return in
}

func addMRLabel(u, label string) error {
	m := mrURLRe.FindStringSubmatch(u)
	if m == nil {
		return fmt.Errorf("not a gitlab MR url: %s", u)
	}
	out, err := exec.Command("glab", "api", "-X", "PUT", "projects/"+url.PathEscape(m[1])+"/merge_requests/"+m[2], "-f", "add_labels="+label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("glab: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func fetchMR(u string) (mrInfo, error) {
	m := mrURLRe.FindStringSubmatch(u)
	if m == nil {
		return mrInfo{}, fmt.Errorf("not a gitlab MR url: %s", u)
	}
	base := "projects/" + url.PathEscape(m[1]) + "/merge_requests/" + m[2]
	var r mrRaw
	for _, p := range []string{base, base + "/approvals"} {
		out, err := exec.Command("glab", "api", p).Output()
		if err != nil {
			return mrInfo{}, err
		}
		if err := json.Unmarshal(out, &r); err != nil {
			return mrInfo{}, err
		}
	}
	if r.HeadPipeline != nil && r.HeadPipeline.Sha != "" {
		r.Gate = fetchGate("projects/" + url.PathEscape(m[1]) + "/repository/commits/" + r.HeadPipeline.Sha + "/statuses?name=cf-factory")
	}
	return mrStatus(r), nil
}

func fetchGate(path string) string {
	var st []pipeRaw
	out, err := exec.Command("glab", "api", path).Output()
	if err != nil || json.Unmarshal(out, &st) != nil || len(st) == 0 {
		return ""
	}
	return slices.MaxFunc(st, func(a, b pipeRaw) int { return cmp.Compare(a.ID, b.ID) }).Status
}

func jiraReq(method, path string, body, out any) error {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, jiraBase+path, rd)
	req.SetBasicAuth(os.Getenv("JIRA_EMAIL"), os.Getenv("JIRA_TOKEN"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
		return fmt.Errorf("jira %s %s: %s %s", method, path, resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func relabel(key, remove, add string) error {
	var ops []map[string]string
	if remove != "" {
		ops = append(ops, map[string]string{"remove": remove})
	}
	if add != "" {
		ops = append(ops, map[string]string{"add": add})
	}
	return jiraReq("PUT", "/rest/api/2/issue/"+key, map[string]any{"update": map[string]any{"labels": ops}}, nil)
}

func fetchIssue(key string) (string, string, error) {
	var out struct {
		Fields struct {
			Description string
			Comment     struct {
				Comments []struct {
					Author  struct{ DisplayName string }
					Created string
					Body    string
				}
			}
		}
	}
	if err := jiraReq("GET", "/rest/api/2/issue/"+key+"?fields=description,comment", nil, &out); err != nil {
		return "", "", err
	}
	var b strings.Builder
	for _, c := range out.Fields.Comment.Comments {
		fmt.Fprintf(&b, "\n---\n\n**%s · %s**\n\n%s\n", c.Author.DisplayName, strings.Replace(c.Created[:min(16, len(c.Created))], "T", " ", 1), wikiToMD(mentions(strings.ReplaceAll(c.Body, "\r", ""))))
	}
	return wikiToMD(mentions(strings.ReplaceAll(out.Fields.Description, "\r", ""))), b.String(), nil
}

var (
	mentionRe = regexp.MustCompile(`\[~accountid:([^\]]+)\]`)
	usersMu   sync.Mutex
	userNames map[string]string
)

func userName(id string) string {
	usersMu.Lock()
	defer usersMu.Unlock()
	p := filepath.Join(stateDir, "users.json")
	if userNames == nil {
		userNames = map[string]string{}
		if b, err := os.ReadFile(p); err == nil {
			json.Unmarshal(b, &userNames)
		}
	}
	if n, ok := userNames[id]; ok {
		return n
	}
	var u struct{ DisplayName string }
	if jiraReq("GET", "/rest/api/2/user?accountId="+url.QueryEscape(id), nil, &u) != nil || u.DisplayName == "" {
		return ""
	}
	userNames[id] = u.DisplayName
	if b, err := json.MarshalIndent(userNames, "", "  "); err == nil {
		os.WriteFile(p, b, 0o644)
	}
	return u.DisplayName
}

func mentions(s string) string {
	return mentionRe.ReplaceAllStringFunc(s, func(m string) string {
		if n := userName(mentionRe.FindStringSubmatch(m)[1]); n != "" {
			return "**@" + n + "**"
		}
		return m
	})
}

var wikiRules = []struct {
	re  *regexp.Regexp
	rep string
}{
	{regexp.MustCompile(`\{\{(.+?)\}\}`), "`$1`"},
	{regexp.MustCompile(`\{(code|noformat)(:[^}]*)?\}`), "\n```\n"},
	{regexp.MustCompile(`\[([^|\]\n]+)\|([^\]\n]+)\]`), "[$1]($2)"},
	{regexp.MustCompile(`(?m)^\*\* `), "  - "},
	{regexp.MustCompile(`(^|[\s(])\*([^*\s][^*\n]*?)\*`), "$1**$2**"},
	{regexp.MustCompile(`(?m)^h([1-6])\. *`), "\x00$1"},
}

var headRe = regexp.MustCompile(`(?m)^\x00([1-6])`)

func numberLists(s string) string {
	lines, n := strings.Split(s, "\n"), 0
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "# "):
			n++
			lines[i] = fmt.Sprintf("%d. %s", n, l[2:])
		case strings.TrimSpace(l) != "" && (i == 0 || strings.TrimSpace(lines[i-1]) == "") && !strings.HasPrefix(l, "** ") && !strings.HasPrefix(l, "## ") && !strings.HasPrefix(l, " "):
			n = 0
		}
	}
	return strings.Join(lines, "\n")
}

func wikiToMD(s string) string {
	s = numberLists(s)
	for _, r := range wikiRules {
		s = r.re.ReplaceAllString(s, r.rep)
	}
	return headRe.ReplaceAllStringFunc(s, func(h string) string {
		return strings.Repeat("#", int(h[1]-'0')) + " "
	})
}

func splitTS(l string) (ts, body string) {
	if m := tsRe.FindString(l); m != "" {
		return strings.TrimSpace(m), l[len(m):]
	}
	return "", l
}

func isMarker(l string) bool {
	for _, p := range []string{"🔧", "✅", "🆔", "💬", "==="} {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

func planPath(key string) string {
	m, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), "work/cleverfarm/repos/*/.worktrees/"+key+"-*/PLAN.md"))
	if len(m) == 0 {
		return ""
	}
	return m[0]
}

func proposedFiles(key string) []string {
	var out []string
	for _, f := range []string{key + ".md", key + ".api.md"} {
		if p := filepath.Join(stateDir, "holdout-proposed", f); fileExists(p) {
			out = append(out, p)
		}
	}
	return out
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func proposalDiff(key string) string {
	var b strings.Builder
	for _, p := range proposedFiles(key) {
		cur := filepath.Join(stateDir, "holdout", filepath.Base(p))
		if !fileExists(cur) {
			cur = os.DevNull
		}
		out, _ := exec.Command("diff", "-u", "--label", "current/"+filepath.Base(p), "--label", "proposed/"+filepath.Base(p), cur, p).Output()
		b.Write(out)
	}
	return b.String()
}

var changedRe = regexp.MustCompile(`(?m)^([SA]\d+) changed`)

func proposalWhy(key string) (why string, changed []string) {
	b, _ := os.ReadFile(filepath.Join(stateDir, "holdout-proposed", key+".why"))
	for _, m := range changedRe.FindAllStringSubmatch(string(b), -1) {
		changed = append(changed, m[1])
	}
	return strings.TrimSpace(string(b)), changed
}

func acceptProposal(key string, now time.Time) (accepted []string, err error) {
	_, changed := proposalWhy(key)
	for _, p := range proposedFiles(key) {
		dst := filepath.Join(stateDir, "holdout", filepath.Base(p))
		if err := os.Rename(p, dst); err != nil {
			return accepted, err
		}
		accepted = append(accepted, dst)
	}
	if len(accepted) == 0 {
		return nil, fmt.Errorf("no holdout proposal for %s", key)
	}
	os.Remove(filepath.Join(stateDir, "holdout-proposed", key+".why"))
	notes := filepath.Join(stateDir, "notes", key+".md")
	b, _ := os.ReadFile(notes)
	ids := strings.Join(changed, ",")
	if ids == "" {
		ids = "scenarios"
	}
	line := now.Format("2006-01-02") + " " + ids + " changed (accepted by Michal)\n"
	c, head := string(b), "## Holdout changes\n"
	if i := strings.Index(c, head); i >= 0 {
		c = c[:i+len(head)] + line + c[i+len(head):]
	} else {
		if c != "" && !strings.HasSuffix(c, "\n") {
			c += "\n"
		}
		c += "\n" + head + line
	}
	return accepted, os.WriteFile(notes, []byte(c), 0o644)
}

func rejectProposal(key string) {
	for _, p := range proposedFiles(key) {
		os.Remove(p)
	}
	os.Remove(filepath.Join(stateDir, "holdout-proposed", key+".why"))
}

func retryAction(ls []string) (remove, add string) {
	for _, l := range ls {
		switch l {
		case "triage-failed":
			return l, "triage"
		case "ai-fix-failed":
			return l, "ai-fix"
		case "ai-verify-failed":
			return l, ""
		case "ai-review-findings":
			return l, ""
		}
	}
	return "", ""
}

var phaseNames = map[string]string{"new": "analysing", "reply": "replying", "mr": "MR follow-up", "impl": "implementing", "rfix": "fixing review", "verify": "verifying", "review": "reviewing", "conflict": "rebasing", "done": "closing"}

func parseUnits(out string) map[string]string {
	m := map[string]string{}
	for _, l := range strings.Split(out, "\n") {
		f := strings.Fields(l)
		if len(f) == 0 {
			continue
		}
		u := strings.TrimSuffix(strings.TrimPrefix(f[0], "sw-factory-"), ".service")
		if i := strings.LastIndex(u, "-"); i > 0 {
			m[u[:i]] = or(phaseNames[u[i+1:]], u[i+1:])
		}
	}
	return m
}

func runningUnits() map[string]string {
	out, _ := exec.Command("systemctl", "--user", "list-units", "sw-factory-*", "--plain", "--no-legend", "--state=active").Output()
	return parseUnits(string(out))
}

func parseLog(r *row, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		_, t := splitTS(sc.Text())
		if m := costRe.FindStringSubmatch(t); m != nil {
			c, _ := strconv.ParseFloat(m[1], 64)
			r.cost += c
		}
		if m := verdRe.FindStringSubmatch(t); m != nil {
			r.verdict = m[1]
		}
		switch {
		case strings.HasPrefix(t, "=== conflict "):
			r.conflict = "pending"
		case strings.Contains(t, "waiting for ai-rebase"):
			r.conflict = "waiting"
		case strings.Contains(t, "conflict resolve failed"):
			r.conflict = "failed"
		case strings.Contains(t, "rebased + pushed"):
			r.conflict = ""
		}
		if strings.HasPrefix(t, "🔧") || strings.HasPrefix(t, "💬") || strings.HasPrefix(t, "✅") {
			r.last = t
		}
		if _, cmd, ok := strings.Cut(t, "reopen: "); ok && strings.HasPrefix(t, "🆔") {
			r.resume, _, _ = strings.Cut(cmd, " — transcript ")
		}
	}
}

type logFacts struct {
	mod                             time.Time
	size                            int64
	cost                            float64
	verdict, conflict, last, resume string
}

var (
	logMu    sync.Mutex
	logCache = map[string]logFacts{}
)

func cachedLog(r *row, p string, st os.FileInfo) {
	logMu.Lock()
	c, ok := logCache[p]
	logMu.Unlock()
	if !ok || !c.mod.Equal(st.ModTime()) || c.size != st.Size() {
		var t row
		parseLog(&t, p)
		c = logFacts{st.ModTime(), st.Size(), t.cost, t.verdict, t.conflict, t.last, t.resume}
		logMu.Lock()
		logCache[p] = c
		logMu.Unlock()
	}
	r.cost, r.verdict, r.conflict, r.last, r.resume = c.cost, c.verdict, c.conflict, c.last, c.resume
}

func workerBusy() bool {
	procs, _ := filepath.Glob("/proc/[0-9]*/cmdline")
	for _, p := range procs {
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), "factory-worker") || strings.Contains(string(b), "sentry-triage") {
			return true
		}
	}
	return false
}

func collect(issues, todos []issue, busy bool, units map[string]string) []row {
	rows := map[string]*row{}
	for _, i := range todos {
		r := &row{key: i.Key, stage: "todo", summary: i.Fields.Summary, labels: i.Fields.Labels, jira: true, todo: true}
		r.seen, _ = time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
		rows[i.Key] = r
	}
	for _, i := range issues {
		r := &row{key: i.Key, stage: stage(i.Fields.Labels), summary: i.Fields.Summary, labels: i.Fields.Labels, jira: true}
		r.seen, _ = time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
		if s, ok := parkedStage(i.Fields.Status.Name, i.Fields.Status.StatusCategory.Key); ok {
			r.stage, r.parked = s, true
		} else if i.Fields.Status.Name == "To test" {
			r.stage, r.parked = "to test", i.Fields.Assignee == nil
		} else if len(proposedFiles(i.Key)) > 0 {
			r.stage = "holdout proposal"
		}
		rows[i.Key] = r
	}
	logs, _ := filepath.Glob(filepath.Join(stateDir, "*.log"))
	for _, p := range logs {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		key := strings.TrimSuffix(filepath.Base(p), ".log")
		r, ok := rows[key]
		if !ok {
			if time.Since(st.ModTime()) > 24*time.Hour {
				continue
			}
			r = &row{key: key, stage: "-"}
			rows[key] = r
		}
		r.seen = st.ModTime()
		r.phase = units[key]
		r.running = r.phase != "" || busy && time.Since(st.ModTime()) < 90*time.Second
		cachedLog(r, p, st)
		if b, err := os.ReadFile(filepath.Join(stateDir, key+".mr")); err == nil {
			r.mr = strings.TrimSpace(string(b))
		}
	}
	out := make([]row, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	return out
}

type entry struct {
	g  int
	at time.Time
}

var (
	enteredMu sync.Mutex
	entered   = map[string]entry{}
)

func order(rows []row, now time.Time) {
	enteredMu.Lock()
	defer enteredMu.Unlock()
	for _, r := range rows {
		g := group(r)
		if e, ok := entered[r.key]; !ok {
			entered[r.key] = entry{g, r.seen}
		} else if e.g != g {
			entered[r.key] = entry{g, now}
		}
	}
	sort.SliceStable(rows, func(a, b int) bool {
		ea, eb := entered[rows[a].key], entered[rows[b].key]
		switch {
		case ea.g != eb.g:
			return ea.g < eb.g
		case !ea.at.Equal(eb.at):
			return ea.at.After(eb.at)
		}
		return rows[a].key > rows[b].key
	})
}

func ago(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func or(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func jiraURL(r row) string {
	if !jiraKey.MatchString(r.key) {
		return ""
	}
	return jiraBase + "/browse/" + r.key
}

func imageRe(dir string) *regexp.Regexp {
	return regexp.MustCompile(`!(` + regexp.QuoteMeta(dir) + `/img-\d+\.png)!`)
}

func uploadImages(text, dir, prefix string, attach func(path, name string) error) (string, error) {
	var err error
	out := imageRe(dir).ReplaceAllStringFunc(text, func(tok string) string {
		p := tok[1 : len(tok)-1]
		name := prefix + "-" + filepath.Base(p)
		if err == nil {
			err = attach(p, name)
		}
		return "!" + name + "!"
	})
	return out, err
}

func jiraAttach(key, path, name string) error {
	out, err := exec.Command("jira", "attach", key, path, name).CombinedOutput()
	if err != nil || !bytes.HasPrefix(bytes.TrimSpace(out), []byte("[")) {
		msg := []rune(strings.TrimSpace(string(out)))
		return fmt.Errorf("jira attach %s failed: %s", name, string(msg[:min(len(msg), 120)]))
	}
	return nil
}

func mrStage(s string) bool {
	return s == "mr" || s == "verified" || s == "review fix" || s == "re-verify"
}

var editor = sync.OnceValue(func() []string {
	out, _ := exec.Command("zsh", "-ic", `e=${EDITOR:-vim}; print -r -- ${aliases[$e]:-$e}`).Output()
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if f := strings.Fields(lines[len(lines)-1]); len(f) > 0 {
		return f
	}
	return []string{"vi"}
})

const defaultAddr = "127.0.0.1:7777"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		addr := defaultAddr
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		serve(addr)
		return
	}
	u := "http://" + defaultAddr + "/?t=" + loadToken()
	for _, c := range [][]string{{"vivaldi", "--app=" + u}, {"xdg-open", u}} {
		if exec.Command(c[0], c[1:]...).Start() == nil {
			return
		}
	}
	fmt.Println(u)
}
