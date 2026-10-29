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
	"github.com/charmbracelet/x/ansi"
)

const (
	jiraBase = "https://cleverfarm.atlassian.net"
	labels   = "triage, triaged, triage-failed, ai-fix, ai-mr, ai-fix-failed, ai-verify-retry, ai-verified, ai-verify-failed, ai-done"
	sideW    = 44
	top      = 2
	jiraRow  = 4
	mrRow    = 5
	detailH  = 7
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
		return "on nightly", true
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
	s := "new"
	switch {
	case has["triage-failed"]:
		s = "analysis-failed"
	case has["ai-fix-failed"]:
		s = "bailed"
	case has["ai-done"]:
		s = "done"
	case has["ai-mr"]:
		s = "mr"
	case has["ai-fix"]:
		s = "implementing"
	case has["triaged"]:
		s = "awaiting-ai-fix"
	}
	if has["ai-verified"] {
		s += "+verified"
	}
	if has["ai-verify-failed"] {
		s += "+verify-failed"
	}
	return s
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

func fetchComments(key string) ([]string, error) {
	var out struct {
		Comments []struct {
			Author  struct{ DisplayName string }
			Created string
			Body    string
		}
	}
	if err := jiraReq("GET", "/rest/api/2/issue/"+key+"/comment?orderBy=created&maxResults=100", nil, &out); err != nil {
		return nil, err
	}
	var lines []string
	for _, c := range out.Comments {
		lines = append(lines, "", "\033[1m── "+c.Author.DisplayName+" · "+strings.Replace(c.Created[:min(16, len(c.Created))], "T", " ", 1)+"\033[0m")
		lines = append(lines, strings.Split(strings.ReplaceAll(c.Body, "\r", ""), "\n")...)
	}
	return lines, nil
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
		if out[a].running != out[b].running {
			return out[a].running
		}
		if out[a].parked != out[b].parked {
			return out[b].parked
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
	key   string
	lines []string
	err   error
}
type doneMsg struct {
	status string
	err    error
}
type editedMsg struct{ key, path string }

var tabs = []string{"log", "plan", "comments"}

type model struct {
	issues      []issue
	rows        []row
	body        []string
	comments    map[string][]string
	cur, off    int
	tab, scroll int
	w, h        int
	busy        bool
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
	m.rows = collect(m.issues, m.busy)
	for i, r := range m.rows {
		if r.key == key {
			m.cur = i
		}
	}
	m.move(0)
}

func (m *model) loadBody() {
	key := m.sel().key
	switch tabs[m.tab] {
	case "log":
		b, _ := os.ReadFile(filepath.Join(stateDir, key+".log"))
		m.body = strings.Split(strings.TrimRight(ansi.Strip(string(b)), "\n"), "\n")
	case "plan":
		m.body = []string{"(no PLAN.md in a worktree for " + key + ")"}
		if p := planPath(key); p != "" {
			b, _ := os.ReadFile(p)
			m.body = append([]string{"\033[2m" + p + "\033[0m", ""}, strings.Split(string(b), "\n")...)
		}
	case "comments":
		m.body = []string{"loading…"}
		if c, ok := m.comments[key]; ok {
			m.body = c
		}
	}
}

func (m *model) listH() int {
	return max(1, m.h-top-1)
}

func (m *model) move(d int) {
	prev := m.sel().key
	m.cur = min(max(m.cur+d, 0), max(len(m.rows)-1, 0))
	if m.cur < m.off {
		m.off = m.cur
	}
	if m.cur >= m.off+m.listH() {
		m.off = m.cur - m.listH() + 1
	}
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
	m.ask, m.pending = q+" y/n", cmd
}

func (m model) sel() row {
	if m.cur < len(m.rows) {
		return m.rows[m.cur]
	}
	return row{}
}

func (m model) paneW() int {
	return max(m.w-sideW-1, 10)
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
	return max(1, m.h-top-1-detailH)
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
	m.comments[key] = []string{"loading…"}
	return loadComments(key)
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadJira, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.comments == nil {
		m.comments = map[string][]string{}
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
		m.comments[msg.key] = msg.lines
		if msg.err != nil {
			m.comments[msg.key] = []string{"\033[31m" + msg.err.Error() + "\033[0m"}
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
			if msg.Action != tea.MouseActionPress {
				break
			}
			switch {
			case side && msg.Y >= top && m.off+msg.Y-top < len(m.rows):
				m.move(m.off + msg.Y - top - m.cur)
			case !side && msg.Y == jiraRow:
				m.open(jiraURL(m.sel()), "jira ticket")
			case !side && msg.Y == mrRow:
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
	r := m.sel()
	isTicket := jiraKey.MatchString(r.key) && r.labels != nil
	switch k {
	case "q", "ctrl+c":
		return m, tea.Quit
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
			m.status = "not waiting at plan gate (needs triaged, no ai-fix/ai-mr)"
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

func (m model) sidebar() []string {
	out := []string{fit("\033[2m  KEY        STAGE              AGE  MR\033[0m", sideW)}
	for i := m.off; i < m.off+m.listH(); i++ {
		if i >= len(m.rows) {
			out = append(out, fit("", sideW))
			continue
		}
		r := m.rows[i]
		mark := "  "
		if r.running {
			mark = "\033[32m●\033[0m "
		}
		mr := ""
		if mm := mrIIDRe.FindStringSubmatch(r.mr); mm != nil {
			mr = "!" + mm[1]
		}
		line := fit(fmt.Sprintf("%-10s %-18s %4s %s", r.key, ansi.Truncate(r.stage, 18, "…"), ago(r.seen), mr), sideW-2)
		switch {
		case i == m.cur:
			line = "\033[7m" + line + "\033[0m"
		case r.parked:
			line = "\033[2m" + line + "\033[0m"
		}
		out = append(out, mark+line)
	}
	return out
}

func (m model) pane() []string {
	r, w := m.sel(), m.paneW()
	tabBar := ""
	for i, t := range tabs {
		if i == m.tab {
			tabBar += "\033[7m " + t + " \033[0m"
		} else {
			tabBar += "\033[2m " + t + " \033[0m"
		}
	}
	out := []string{
		fit("\033[1m"+r.key+"\033[0m  "+or(r.summary, "(no jira ticket)"), w),
		fit(fmt.Sprintf("%s · $%.2f · verify %s · %s ago", r.stage, r.cost, or(r.verdict, "-"), ago(r.seen)), w),
		fit("labels   "+or(strings.Join(r.labels, ", "), "-"), w),
		fit("jira     \033[4m"+or(jiraURL(r), "-")+"\033[0m", w),
		fit("mr       \033[4m"+or(r.mr, "-")+"\033[0m", w),
		fit("session  "+or(r.resume, "-"), w),
		fit(tabBar+"\033[2m"+strings.Repeat("─", w)+"\033[0m", w),
	}
	body, h := m.wrapped(), m.bodyH()
	from, to := m.scroll, min(m.scroll+h, len(body))
	if m.bottomAnchored() {
		to = len(body) - m.scroll
		from = max(to-h, 0)
	}
	for i := from; i < to; i++ {
		out = append(out, fit(body[i], w))
	}
	return out
}

func (m model) View() string {
	if m.w == 0 {
		return ""
	}
	var b strings.Builder
	worker := "idle"
	if m.busy {
		worker = "\033[32mBUSY\033[0m"
	}
	jira := "jira " + ago(m.jiraAt) + " ago"
	if m.jiraErr != nil {
		jira = "\033[31mjira lookup failed: " + m.jiraErr.Error() + "\033[0m"
	}
	total := 0.0
	for _, r := range m.rows {
		total += r.cost
	}
	scrolled := ""
	if m.scroll > 0 {
		scrolled = fmt.Sprintf("  \033[33mscrolled %d\033[0m", m.scroll)
	}
	b.WriteString(fit(fmt.Sprintf("\033[1mcf-triage\033[0m  %s  worker %s  service %s  %s  %d tickets $%.2f%s", time.Now().Format("15:04:05"), worker, or(m.svc, "?"), jira, len(m.rows), total, scrolled), m.w) + "\n")
	side, pane := m.sidebar(), m.pane()
	for i := 0; i < m.h-top; i++ {
		s, p := fit("", sideW), ""
		if i < len(side) {
			s = side[i]
		}
		if i < len(pane) {
			p = pane[i]
		}
		b.WriteString(s + "\033[2m│\033[0m" + p + "\n")
	}
	foot := "\033[2mj/k move  tab log|plan|comments  a approve  R reply  t retry  w run worker  c session  enter/J/m open  l less  ^u/^d scroll  r refresh  q quit\033[0m  " + m.status
	if m.ask != "" {
		foot = "\033[1;33m" + m.ask + "\033[0m"
	}
	b.WriteString(fit(foot, m.w))
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(model{comments: map[string][]string{}}, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
