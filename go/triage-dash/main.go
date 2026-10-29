package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

const (
	jiraBase = "https://cleverfarm.atlassian.net"
	labels   = "triage, triaged, triage-failed, ai-fix, ai-mr, ai-fix-failed, ai-verify-retry, ai-verified, ai-verify-failed, ai-done"
	sideW    = 34
	top      = 2
	detailH  = 5
)

var (
	stateDir = filepath.Join(os.Getenv("HOME"), ".local/state/cf-triage")
	costRe   = regexp.MustCompile(`cost=\$([0-9.]+)`)
	verdRe   = regexp.MustCompile(`^\s*VERDICT: (\S+)`)
	mrIIDRe  = regexp.MustCompile(`merge_requests/(\d+)`)
	jiraKey  = regexp.MustCompile(`^[A-Z]+-\d+$`)
)

type issue struct {
	Key    string
	Fields struct {
		Summary string
		Labels  []string
		Updated string
		Status  struct {
			Name           string
			StatusCategory struct{ Key string }
		}
	}
}

type row struct {
	key, stage, summary, verdict, mr, last, resume string
	cost                                           float64
	seen                                           time.Time
	running, parked                                bool
	labels                                         []string
}

func parkedStage(status, category string) (string, bool) {
	switch {
	case status == "On nightly":
		return "nightly", true
	case status == "Release candidate":
		return "RC", true
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
	case has["ai-fix-failed"]:
		return "bailed"
	case has["ai-verify-failed"]:
		return "verify failed"
	case has["ai-done"]:
		return "done"
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
	"plan gate":       {"📋", "33"},
	"verified":        {"✅", "32"},
	"verify failed":   {"❗", "31"},
	"analysis failed": {"❌", "31"},
	"bailed":          {"❌", "31"},
	"implementing":    {"🔨", "36"},
	"mr":              {"🔍", "34"},
	"new":             {"🆕", "36"},
	"done":            {"🏁", "2"},
	"RC":              {"📦", "2"},
	"nightly":         {"🌙", "2"},
	"-":               {"📄", "2"},
}

var groups = []struct{ name, color string }{
	{"RUNNING", "1;32"},
	{"NEEDS YOU", "1;33"},
	{"IN FLIGHT", "1;34"},
	{"PARKED", "2"},
	{"OTHER LOGS", "2"},
}

func group(r row) int {
	switch {
	case r.running:
		return 0
	case r.labels == nil:
		return 4
	case r.parked:
		return 3
	case slices.Contains([]string{"plan gate", "verified", "verify failed", "analysis failed", "bailed"}, r.stage):
		return 1
	}
	return 2
}

func fetchJira() ([]issue, error) {
	jql := "project = DEV AND labels in (" + labels + ") AND (statusCategory != Done OR updated >= -3d) ORDER BY updated DESC"
	var out struct{ Issues []issue }
	return out.Issues, jiraReq("GET", "/rest/api/3/search/jql?fields=labels,summary,updated,status&maxResults=100&jql="+url.QueryEscape(jql), nil, &out)
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
		return fmt.Errorf("jira %s %s: %s", method, path, resp.Status)
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

func fetchComments(key string) (string, error) {
	var out struct {
		Comments []struct {
			Author  struct{ DisplayName string }
			Created string
			Body    string
		}
	}
	if err := jiraReq("GET", "/rest/api/2/issue/"+key+"/comment?orderBy=created&maxResults=100", nil, &out); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range out.Comments {
		fmt.Fprintf(&b, "\n---\n\n**%s · %s**\n\n%s\n", c.Author.DisplayName, strings.Replace(c.Created[:min(16, len(c.Created))], "T", " ", 1), wikiToMD(strings.ReplaceAll(c.Body, "\r", "")))
	}
	return b.String(), nil
}

var wikiRules = []struct {
	re  *regexp.Regexp
	rep string
}{
	{regexp.MustCompile(`\{\{(.+?)\}\}`), "`$1`"},
	{regexp.MustCompile(`\{(code|noformat)(:[^}]*)?\}`), "\n```\n"},
	{regexp.MustCompile(`\[([^|\]\n]+)\|([^\]\n]+)\]`), "[$1]($2)"},
	{regexp.MustCompile(`(?m)^# `), "1. "},
	{regexp.MustCompile(`(?m)^\*\* `), "  - "},
	{regexp.MustCompile(`(^|[\s(])\*([^*\s][^*\n]*?)\*`), "$1**$2**"},
	{regexp.MustCompile(`(?m)^h([1-6])\. *`), "\x00$1"},
}

var headRe = regexp.MustCompile(`(?m)^\x00([1-6])`)

func wikiToMD(s string) string {
	for _, r := range wikiRules {
		s = r.re.ReplaceAllString(s, r.rep)
	}
	return headRe.ReplaceAllStringFunc(s, func(h string) string {
		return strings.Repeat("#", int(h[1]-'0')) + " "
	})
}

func renderMD(src string, w int) []string {
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle("dark"), glamour.WithWordWrap(max(w-4, 20)))
	if err != nil {
		return strings.Split(src, "\n")
	}
	out, err := r.Render(src)
	if err != nil {
		return strings.Split(src, "\n")
	}
	return strings.Split(strings.Trim(out, "\n"), "\n")
}

func isMarker(l string) bool {
	for _, p := range []string{"🔧", "✅", "🆔", "💬", "==="} {
		if strings.HasPrefix(l, p) {
			return true
		}
	}
	return false
}

func renderLog(lines []string, w int) []string {
	var out, md []string
	flush := func() {
		if len(md) > 0 {
			out = append(out, renderMD(strings.Join(md, "\n"), w)...)
		}
		md = nil
	}
	for _, l := range lines {
		if isMarker(l) {
			flush()
		}
		switch {
		case strings.HasPrefix(l, "💬"), md != nil && !isMarker(l):
			md = append(md, l)
		case strings.HasPrefix(l, "🔧"):
			out = append(out, "\033[2m"+l+"\033[0m")
		case strings.HasPrefix(l, "✅"):
			out = append(out, l)
			md = []string{}
		default:
			out = append(out, l)
		}
	}
	flush()
	return out
}

func planPath(key string) string {
	m, _ := filepath.Glob(filepath.Join(os.Getenv("HOME"), "work/cleverfarm/repos/*/.worktrees/"+key+"-*/PLAN.md"))
	if len(m) == 0 {
		return ""
	}
	return m[0]
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
		}
	}
	return "", ""
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
		t := sc.Text()
		if m := costRe.FindStringSubmatch(t); m != nil {
			c, _ := strconv.ParseFloat(m[1], 64)
			r.cost += c
		}
		if m := verdRe.FindStringSubmatch(t); m != nil {
			r.verdict = m[1]
		}
		if strings.HasPrefix(t, "🔧") || strings.HasPrefix(t, "💬") || strings.HasPrefix(t, "✅") {
			r.last = t
		}
		if _, cmd, ok := strings.Cut(t, "reopen: "); ok && strings.HasPrefix(t, "🆔") {
			r.resume = cmd
		}
	}
}

func workerBusy() bool {
	procs, _ := filepath.Glob("/proc/[0-9]*/cmdline")
	for _, p := range procs {
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), "triage-worker") || strings.Contains(string(b), "sentry-triage") {
			return true
		}
	}
	return false
}

func collect(issues []issue, busy bool) []row {
	rows := map[string]*row{}
	for _, i := range issues {
		r := &row{key: i.Key, stage: stage(i.Fields.Labels), summary: i.Fields.Summary, labels: i.Fields.Labels}
		r.seen, _ = time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
		if s, ok := parkedStage(i.Fields.Status.Name, i.Fields.Status.StatusCategory.Key); ok {
			r.stage, r.parked = s, true
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
		r.running = busy && time.Since(st.ModTime()) < 90*time.Second
		parseLog(r, p)
		if b, err := os.ReadFile(filepath.Join(stateDir, key+".mr")); err == nil {
			r.mr = strings.TrimSpace(string(b))
		}
	}
	out := make([]row, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sort.Slice(out, func(a, b int) bool {
		if ga, gb := group(out[a]), group(out[b]); ga != gb {
			return ga < gb
		}
		return out[a].seen.After(out[b].seen)
	})
	return out
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

func fit(s string, w int) string {
	s = ansi.Truncate(s, max(w, 0), "…")
	return s + strings.Repeat(" ", max(w-ansi.StringWidth(s), 0))
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

type tickMsg struct{}
type jiraMsg struct {
	issues []issue
	err    error
}
type commentsMsg struct {
	key string
	md  string
	err error
}
type doneMsg struct {
	status string
	err    error
}
type editedMsg struct{ key, path string }

type sideLine struct {
	text string
	row  int
}

var tabs = []string{"log", "plan", "comments"}

type helpSection struct {
	title string
	keys  [][2]string
}

var helpCols = [][]helpSection{
	{
		{"Navigate", [][2]string{{"j k ↑ ↓", "move"}, {"g G", "top / bottom"}, {"tab", "log / plan / comments"}, {"^u ^d", "scroll pane"}, {"z", "show other logs"}}},
		{"Factory  (asks y/n)", [][2]string{{"a", "approve plan → ai-fix"}, {"R", "reply in Jira"}, {"t", "retry failed stage"}, {"w", "run worker now"}}},
	},
	{
		{"Open", [][2]string{{"enter o", "MR, else Jira"}, {"J", "Jira"}, {"m", "MR"}, {"l", "log in less"}, {"c", "resume session"}}},
		{"Other", [][2]string{{"r", "refresh Jira"}, {"?", "this help"}, {"q", "quit"}}},
	},
}

var legend = []string{"plan gate", "verified", "verify failed", "bailed", "implementing", "mr", "new", "done", "RC", "nightly", "-"}

func helpLines() []string {
	const keyW, colW = 9, 34
	var cols [][]string
	for _, col := range helpCols {
		var ls []string
		for i, s := range col {
			if i > 0 {
				ls = append(ls, "")
			}
			ls = append(ls, "\033[1;33m"+s.title+"\033[0m")
			for _, k := range s.keys {
				ls = append(ls, "  \033[1;36m"+fit(k[0], keyW)+"\033[0m"+k[1])
			}
		}
		cols = append(cols, ls)
	}
	var out []string
	for i := 0; i < max(len(cols[0]), len(cols[1])); i++ {
		l, r := "", ""
		if i < len(cols[0]) {
			l = cols[0][i]
		}
		if i < len(cols[1]) {
			r = cols[1][i]
		}
		out = append(out, fit(l, colW)+r)
	}
	out = append(out, "", "\033[1;33mLegend\033[0m")
	row := "  "
	for i, s := range legend {
		name := s
		if s == "-" {
			name = "no ticket"
		}
		if s == "bailed" {
			name = "failed / bailed"
		}
		row += fit(stageStyle[s].icon+" \033["+stageStyle[s].color+"m"+name+"\033[0m", 22)
		if i%3 == 2 {
			out = append(out, row)
			row = "  "
		}
	}
	out = append(out, row+fit("\033[32m●\033[0m  running", 22), "", "\033[2many key closes\033[0m")
	return out
}

type model struct {
	issues      []issue
	rows        []row
	otherN      int
	body        []string
	bodySrc     string
	comments    map[string]string
	cur, off    int
	tab, scroll int
	w, h        int
	busy, help  bool
	showOther   bool
	svc, status string
	ask         string
	pending     tea.Cmd
	jiraErr     error
	jiraAt      time.Time
}

func tick() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func loadJira() tea.Msg {
	is, err := fetchJira()
	return jiraMsg{is, err}
}

func loadComments(key string) tea.Cmd {
	return func() tea.Msg {
		ls, err := fetchComments(key)
		return commentsMsg{key, ls, err}
	}
}

func do(status string, f func() error) tea.Cmd {
	return func() tea.Msg { return doneMsg{status, f()} }
}

func (m *model) reload() {
	m.busy = workerBusy()
	out, _ := exec.Command("systemctl", "--user", "is-active", "cf-triage.service").Output()
	m.svc = strings.TrimSpace(string(out))
	key := m.sel().key
	m.rows, m.otherN = nil, 0
	for _, r := range collect(m.issues, m.busy) {
		if group(r) == 4 {
			m.otherN++
			if !m.showOther {
				continue
			}
		}
		m.rows = append(m.rows, r)
	}
	for i, r := range m.rows {
		if r.key == key {
			m.cur = i
		}
	}
	m.move(0)
}

func (m *model) loadBody() {
	key, w := m.sel().key, m.paneW()
	var src string
	switch tabs[m.tab] {
	case "log":
		b, _ := os.ReadFile(filepath.Join(stateDir, key+".log"))
		src = ansi.Strip(string(b))
	case "plan":
		src = "_(no PLAN.md in a worktree for " + key + ")_"
		if p := planPath(key); p != "" {
			b, _ := os.ReadFile(p)
			src = "`" + p + "`\n\n" + string(b)
		}
	case "comments":
		src = or(m.comments[key], "_loading…_")
	}
	cache := fmt.Sprint(m.tab, w, src)
	if cache == m.bodySrc {
		return
	}
	m.bodySrc = cache
	if tabs[m.tab] == "log" {
		m.body = renderLog(strings.Split(strings.TrimRight(src, "\n"), "\n"), w)
	} else {
		m.body = renderMD(src, w)
	}
}

func (m model) listH() int {
	return max(1, m.h-top-1)
}

func (m model) sideLines() []sideLine {
	var out []sideLine
	g := -1
	for i, r := range m.rows {
		if gr := group(r); gr != g {
			if g != -1 {
				out = append(out, sideLine{"", -1})
			}
			g = gr
			name := groups[g].name
			if g == 4 {
				name = fmt.Sprintf("%s (%d)", name, m.otherN)
			}
			out = append(out, sideLine{" \033[" + groups[g].color + "m" + name + "\033[0m", -1})
		}
		out = append(out, sideLine{"", i})
	}
	if !m.showOther && m.otherN > 0 {
		out = append(out, sideLine{"", -1}, sideLine{fmt.Sprintf(" \033[2m%s (%d)  z shows\033[0m", groups[4].name, m.otherN), -1})
	}
	return out
}

func (m *model) move(d int) {
	prev := m.sel().key
	m.cur = min(max(m.cur+d, 0), max(len(m.rows)-1, 0))
	lines := m.sideLines()
	at := 0
	for i, l := range lines {
		if l.row == m.cur {
			at = i
		}
	}
	first := at
	if at > 0 && lines[at-1].row == -1 {
		first = at - 1
	}
	if first < m.off {
		m.off = first
	}
	if at >= m.off+m.listH() {
		m.off = at - m.listH() + 1
	}
	m.off = max(min(m.off, len(lines)-m.listH()), 0)
	if m.sel().key != prev {
		m.scroll = 0
	}
	m.loadBody()
}

func (m *model) open(u, what string) {
	if u == "" {
		m.status = "no " + what
		return
	}
	exec.Command("xdg-open", u).Start()
	m.status = "opened " + u
}

func (m *model) confirm(q string, cmd tea.Cmd) {
	m.ask, m.pending = q+"  y/n", cmd
}

func (m model) sel() row {
	if m.cur < len(m.rows) {
		return m.rows[m.cur]
	}
	return row{}
}

func (m model) paneW() int {
	return max(m.w-sideW-4, 10)
}

func (m model) bottomAnchored() bool {
	return tabs[m.tab] != "plan"
}

func (m model) wrapped() []string {
	var out []string
	for _, l := range m.body {
		out = append(out, strings.Split(ansi.Hardwrap(l, m.paneW(), true), "\n")...)
	}
	return out
}

func (m model) bodyH() int {
	return max(1, m.listH()-detailH)
}

func (m *model) scrollBody(up int) {
	if !m.bottomAnchored() {
		up = -up
	}
	m.scroll = min(max(m.scroll+up, 0), max(len(m.wrapped())-m.bodyH(), 0))
}

func (m model) needComments() tea.Cmd {
	key := m.sel().key
	if tabs[m.tab] != "comments" || !jiraKey.MatchString(key) {
		return nil
	}
	if _, ok := m.comments[key]; ok {
		return nil
	}
	m.comments[key] = "_loading…_"
	return loadComments(key)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadJira, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.comments == nil {
		m.comments = map[string]string{}
	}
	m, cmd := m.update(msg)
	return m, tea.Batch(cmd, m.needComments())
}

func (m model) update(msg tea.Msg) (model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.move(0)
	case tickMsg:
		m.reload()
		if time.Since(m.jiraAt) >= time.Minute {
			m.jiraAt = time.Now()
			return m, tea.Batch(loadJira, tick())
		}
		return m, tick()
	case jiraMsg:
		m.jiraAt, m.jiraErr = time.Now(), msg.err
		if msg.err == nil {
			m.issues = msg.issues
		}
		m.reload()
	case commentsMsg:
		m.comments[msg.key] = msg.md
		if msg.err != nil {
			m.comments[msg.key] = "**error:** " + msg.err.Error()
		}
		m.loadBody()
	case doneMsg:
		m.status = "\033[32m" + msg.status + "\033[0m"
		if msg.err != nil {
			m.status = "\033[31m" + msg.err.Error() + "\033[0m"
		}
		delete(m.comments, m.sel().key)
		m.jiraAt = time.Now()
		return m, loadJira
	case editedMsg:
		b, _ := os.ReadFile(msg.path)
		os.Remove(msg.path)
		text := strings.TrimSpace(string(b))
		if text == "" {
			m.status = "reply empty, not sent"
			break
		}
		m.confirm(fmt.Sprintf("post %d-char comment to %s?", len(text), msg.key), do("comment posted to "+msg.key, func() error {
			return jiraReq("POST", "/rest/api/2/issue/"+msg.key+"/comment", map[string]string{"body": text}, nil)
		}))
	case tea.MouseMsg:
		side := msg.X < sideW
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if side {
				m.move(-1)
			} else {
				m.scrollBody(3)
			}
		case tea.MouseButtonWheelDown:
			if side {
				m.move(1)
			} else {
				m.scrollBody(-3)
			}
		case tea.MouseButtonLeft:
			if msg.Action != tea.MouseActionPress || m.help {
				break
			}
			lines, i := m.sideLines(), m.off+msg.Y-top
			switch {
			case side && msg.Y >= top && i < len(lines) && lines[i].row >= 0:
				m.move(lines[i].row - m.cur)
			case !side && msg.Y == top:
				m.open(jiraURL(m.sel()), "jira ticket")
			case !side && msg.Y == top+1:
				m.open(m.sel().mr, "MR")
			}
		}
	case tea.KeyMsg:
		if m.pending != nil {
			cmd := m.pending
			m.ask, m.pending = "", nil
			if msg.String() == "y" {
				m.status = "working…"
				return m, cmd
			}
			m.status = "cancelled"
			return m, nil
		}
		if msg.Type != tea.KeyRunes || len(msg.Runes) < 2 {
			return m.key(msg.String())
		}
		var cmds []tea.Cmd
		for _, r := range msg.Runes {
			var c tea.Cmd
			m, c = m.key(string(r))
			cmds = append(cmds, c)
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m model) key(k string) (model, tea.Cmd) {
	if m.help {
		m.help = false
		return m, nil
	}
	r := m.sel()
	isTicket := jiraKey.MatchString(r.key) && r.labels != nil
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "?":
		m.help = true
	case "z":
		m.showOther = !m.showOther
		m.reload()
	case "j", "down":
		m.move(1)
	case "k", "up":
		m.move(-1)
	case "g", "home":
		m.move(-len(m.rows))
	case "G", "end":
		m.move(len(m.rows))
	case "tab", "shift+tab":
		d := 1
		if k == "shift+tab" {
			d = len(tabs) - 1
		}
		m.tab, m.scroll = (m.tab+d)%len(tabs), 0
		m.loadBody()
	case "ctrl+u", "pgup":
		m.scrollBody(m.bodyH() / 2)
	case "ctrl+d", "pgdown":
		m.scrollBody(-m.bodyH() / 2)
	case "enter", "o":
		if r.mr != "" {
			m.open(r.mr, "MR")
		} else {
			m.open(jiraURL(r), "jira ticket")
		}
	case "J":
		m.open(jiraURL(r), "jira ticket")
	case "m":
		m.open(r.mr, "MR")
	case "r":
		m.status = "refreshing jira"
		delete(m.comments, r.key)
		m.jiraAt = time.Now()
		return m, loadJira
	case "l":
		return m, tea.ExecProcess(exec.Command("less", "+G", filepath.Join(stateDir, r.key+".log")), func(error) tea.Msg { return tickMsg{} })
	case "c":
		if r.resume == "" {
			m.status = "no session in log"
			break
		}
		if os.Getenv("TMUX") != "" {
			exec.Command("tmux", "new-window", "-n", r.key, r.resume).Run()
			break
		}
		return m, tea.ExecProcess(exec.Command("zsh", "-ic", r.resume), func(error) tea.Msg { return tickMsg{} })
	case "a":
		if !isTicket || !slices.Contains(r.labels, "triaged") || slices.Contains(r.labels, "ai-fix") || slices.Contains(r.labels, "ai-mr") {
			m.status = "not waiting at plan gate"
			break
		}
		m.confirm("approve plan + scenarios: add ai-fix to "+r.key+"?", do("ai-fix added to "+r.key, func() error { return relabel(r.key, "", "ai-fix") }))
	case "t":
		rm, add := retryAction(r.labels)
		if !isTicket || rm == "" {
			m.status = "nothing failed to retry"
			break
		}
		m.confirm(fmt.Sprintf("retry %s: remove %s, add %s?", r.key, rm, or(add, "nothing (re-verify)")), do("relabelled "+r.key, func() error { return relabel(r.key, rm, add) }))
	case "R":
		if !isTicket {
			m.status = "no jira ticket"
			break
		}
		f, err := os.CreateTemp("", "triage-reply-"+r.key+"-*.txt")
		if err != nil {
			m.status = err.Error()
			break
		}
		f.Close()
		return m, tea.ExecProcess(exec.Command("zsh", "-ic", `eval "${EDITOR:-vim} ${(q)1}"`, "zsh", f.Name()), func(error) tea.Msg { return editedMsg{r.key, f.Name()} })
	case "w":
		if !isTicket {
			m.status = "no jira ticket"
			break
		}
		if os.Getenv("TMUX") == "" {
			m.status = "w needs tmux"
			break
		}
		m.confirm("run triage-worker "+r.key+" now (new tmux window)?", do("worker started for "+r.key, func() error {
			return exec.Command("tmux", "new-window", "-d", "-n", "w:"+r.key, "~/bin/triage-worker "+r.key+"; echo; read '?done, enter closes'").Run()
		}))
	}
	return m, nil
}

func (m model) rowLine(i int) string {
	r := m.rows[i]
	st := stageStyle[r.stage]
	icon := or(st.icon, "  ")
	if r.running {
		icon = "\033[32m●\033[0m "
	}
	key, stg, age := fmt.Sprintf("%-9s", r.key), fmt.Sprintf("%-15s", ansi.Truncate(r.stage, 15, "…")), fmt.Sprintf("%4s", ago(r.seen))
	if i == m.cur {
		return " " + icon + " \033[7m " + key + " " + stg + age + "\033[0m"
	}
	return fit(" "+icon+"  "+key+" \033["+or(st.color, "0")+"m"+stg+"\033[0m\033[2m"+age+"\033[0m", sideW)
}

func (m model) sidebar() []string {
	lines := m.sideLines()
	var out []string
	for i := m.off; i < m.off+m.listH(); i++ {
		switch {
		case i >= len(lines):
			out = append(out, "")
		case lines[i].row >= 0:
			out = append(out, m.rowLine(lines[i].row))
		default:
			out = append(out, lines[i].text)
		}
	}
	return out
}

func (m model) pane() []string {
	r := m.sel()
	st := stageStyle[r.stage]
	meta := []string{or(st.icon, "") + " \033[" + or(st.color, "0") + "m" + r.stage + "\033[0m", fmt.Sprintf("$%.2f", r.cost)}
	if r.verdict != "" {
		meta = append(meta, "verify "+r.verdict)
	}
	meta = append(meta, ago(r.seen)+" ago")
	if mm := mrIIDRe.FindStringSubmatch(r.mr); mm != nil {
		meta = append(meta, "\033[4m!"+mm[1]+"\033[0m")
	}
	tabBar := ""
	for i, t := range tabs {
		if i == m.tab {
			tabBar += "\033[1;4;36m" + t + "\033[0m   "
		} else {
			tabBar += "\033[2m" + t + "\033[0m   "
		}
	}
	if m.scroll > 0 {
		tabBar += fmt.Sprintf("\033[33m↑ %d\033[0m", m.scroll)
	}
	out := []string{
		"\033[1m" + r.key + "\033[0m  " + or(r.summary, "\033[2m(no jira ticket)\033[0m"),
		strings.Join(meta, " \033[2m·\033[0m "),
		"",
		tabBar,
		"\033[2m" + strings.Repeat("─", m.paneW()) + "\033[0m",
	}
	body, h := m.wrapped(), m.bodyH()
	from, to := m.scroll, min(m.scroll+h, len(body))
	if m.bottomAnchored() {
		to = len(body) - m.scroll
		from = max(to-h, 0)
	}
	return append(out, body[from:to]...)
}

func (m model) overlay(base []string) []string {
	lines := append(append([]string{""}, helpLines()...), "")
	w := 0
	for _, l := range lines {
		w = max(w, ansi.StringWidth(l))
	}
	w += 6
	box := []string{"╭" + strings.Repeat("─", w) + "╮"}
	for _, l := range lines {
		box = append(box, "│   "+fit(l, w-3)+"│")
	}
	box = append(box, "╰"+strings.Repeat("─", w)+"╯")
	x, y := max((m.w-w-2)/2, 0), max((len(base)-len(box))/2, 0)
	for i, b := range box {
		if y+i >= len(base) {
			break
		}
		l := fit(base[y+i], m.w)
		base[y+i] = ansi.Truncate(l, x, "") + "\033[0m" + b + ansi.TruncateLeft(l, x+w+2, "")
	}
	return base
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}
	total := 0.0
	for _, r := range m.rows {
		total += r.cost
	}
	worker := "\033[2m○ idle\033[0m"
	if m.busy {
		worker = "\033[32m● busy\033[0m"
	}
	head := fmt.Sprintf(" \033[1mcf-triage\033[0m   %s   \033[2m%d tickets   $%.2f\033[0m", worker, len(m.rows)+m.otherN, total)
	if m.svc == "failed" {
		head += "   \033[31mservice failed\033[0m"
	}
	switch {
	case m.jiraErr != nil:
		head += "   \033[31mjira: " + m.jiraErr.Error() + "\033[0m"
	case !m.jiraAt.IsZero() && time.Since(m.jiraAt) > 3*time.Minute:
		head += "   \033[33mjira " + ago(m.jiraAt) + " old\033[0m"
	}
	view := []string{fit(head, m.w), ""}
	side, pane := m.sidebar(), m.pane()
	for i := 0; i < m.listH(); i++ {
		s, p := "", ""
		if i < len(side) {
			s = side[i]
		}
		if i < len(pane) {
			p = pane[i]
		}
		view = append(view, fit(s, sideW)+"\033[2m │\033[0m  "+fit(p, m.paneW()))
	}
	foot := fit(" \033[2m? help   q quit\033[0m", 20) + m.status
	if m.ask != "" {
		foot = " \033[1;33m" + m.ask + "\033[0m"
	}
	view = append(view, fit(foot, m.w))
	if m.help {
		view = m.overlay(view)
	}
	return strings.Join(view, "\n")
}

func main() {
	if _, err := tea.NewProgram(model{comments: map[string]string{}}, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
