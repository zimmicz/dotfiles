package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	}
}

type row struct {
	key, stage, summary, verdict, mr, last, resume string
	cost                                           float64
	seen                                           time.Time
	running                                        bool
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
	req, _ := http.NewRequest("GET", jiraBase+"/rest/api/3/search/jql?fields=labels,summary,updated&maxResults=100&jql="+url.QueryEscape(jql), nil)
	req.SetBasicAuth(os.Getenv("JIRA_EMAIL"), os.Getenv("JIRA_TOKEN"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jira %s", resp.Status)
	}
	var out struct{ Issues []issue }
	return out.Issues, json.NewDecoder(resp.Body).Decode(&out)
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
		r := &row{key: i.Key, stage: stage(i.Fields.Labels), summary: i.Fields.Summary}
		r.seen, _ = time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
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

type model struct {
	issues      []issue
	rows        []row
	log         []string
	cur, off    int
	logBack     int
	w, h        int
	busy        bool
	svc, status string
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

func (m *model) loadLog() {
	b, _ := os.ReadFile(filepath.Join(stateDir, m.sel().key+".log"))
	m.log = strings.Split(strings.TrimRight(ansi.Strip(string(b)), "\n"), "\n")
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
		m.logBack = 0
	}
	m.loadLog()
}

func (m *model) open(u, what string) {
	if u == "" {
		m.status = "no " + what
		return
	}
	exec.Command("xdg-open", u).Start()
	m.status = "opened " + u
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

func (m model) wrappedLog() []string {
	var out []string
	for _, l := range m.log {
		out = append(out, strings.Split(ansi.Hardwrap(l, m.paneW(), true), "\n")...)
	}
	return out
}

func (m model) logH() int {
	return max(1, m.h-top-1-detailH)
}

func (m *model) scrollLog(d int) {
	m.logBack = min(max(m.logBack+d, 0), max(len(m.wrappedLog())-m.logH(), 0))
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadJira, tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case tea.MouseMsg:
		side := msg.X < sideW
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if side {
				m.move(-1)
			} else {
				m.scrollLog(3)
			}
		case tea.MouseButtonWheelDown:
			if side {
				m.move(1)
			} else {
				m.scrollLog(-3)
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
		r := m.sel()
		switch msg.String() {
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
		case "ctrl+u", "pgup":
			m.scrollLog(m.logH() / 2)
		case "ctrl+d", "pgdown":
			m.scrollLog(-m.logH() / 2)
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
		}
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
		if i == m.cur {
			line = "\033[7m" + line + "\033[0m"
		}
		out = append(out, mark+line)
	}
	return out
}

func (m model) pane() []string {
	r, w := m.sel(), m.paneW()
	verdict := or(r.verdict, "-")
	out := []string{
		fit("\033[1m"+r.key+"\033[0m  "+or(r.summary, "(no jira ticket)"), w),
		fit(fmt.Sprintf("%s · $%.2f · verify %s · %s ago", r.stage, r.cost, verdict, ago(r.seen)), w),
		fit("", w),
		fit("jira     \033[4m"+or(jiraURL(r), "-")+"\033[0m", w),
		fit("mr       \033[4m"+or(r.mr, "-")+"\033[0m", w),
		fit("session  "+or(r.resume, "-"), w),
		fit("\033[2m"+strings.Repeat("─", w)+"\033[0m", w),
	}
	log := m.wrappedLog()
	end := len(log) - m.logBack
	for i := max(end-m.logH(), 0); i < end; i++ {
		out = append(out, fit(log[i], w))
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
	follow := ""
	if m.logBack > 0 {
		follow = fmt.Sprintf("  \033[33mlog scrolled -%d\033[0m", m.logBack)
	}
	b.WriteString(fit(fmt.Sprintf("\033[1mcf-triage\033[0m  %s  worker %s  service %s  %s  %d tickets $%.2f%s", time.Now().Format("15:04:05"), worker, or(m.svc, "?"), jira, len(m.rows), total, follow), m.w) + "\n")
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
	b.WriteString(fit("\033[2mj/k move  enter open MR|jira  J jira  m MR  c resume session  l less  ^u/^d log  r refresh  q quit\033[0m  "+m.status, m.w))
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(model{}, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
