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
)

const (
	jiraBase  = "https://cleverfarm.atlassian.net"
	labels    = "triage, triaged, triage-failed, ai-fix, ai-mr, ai-fix-failed, ai-verify-retry, ai-verified, ai-verify-failed, ai-done"
	headLines = 3
	footLines = 7
	keyW      = 10
	mrX       = 10 + 1 + 26 + 1 + 5 + 1 + 7 + 1 + 8 + 1
	mrW       = 6
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

func cut(s string, n int) string {
	r := []rune(s)
	if n < 1 {
		return ""
	}
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
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
	cur, off    int
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
	var key string
	if m.cur < len(m.rows) {
		key = m.rows[m.cur].key
	}
	m.rows = collect(m.issues, m.busy)
	for i, r := range m.rows {
		if r.key == key {
			m.cur = i
		}
	}
	m.move(0)
}

func (m *model) visible() int {
	return max(1, m.h-headLines-footLines)
}

func (m *model) move(d int) {
	m.cur = min(max(m.cur+d, 0), max(len(m.rows)-1, 0))
	if m.cur < m.off {
		m.off = m.cur
	}
	if m.cur >= m.off+m.visible() {
		m.off = m.cur - m.visible() + 1
	}
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
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.move(-1)
		case tea.MouseButtonWheelDown:
			m.move(1)
		case tea.MouseButtonLeft:
			if msg.Action != tea.MouseActionPress {
				break
			}
			i := m.off + msg.Y - headLines
			if msg.Y < headLines || i >= len(m.rows) || msg.Y-headLines >= m.visible() {
				break
			}
			m.cur = i
			switch {
			case msg.X < keyW:
				m.open(jiraURL(m.sel()), "jira ticket")
			case msg.X >= mrX && msg.X < mrX+mrW:
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

func (m model) View() string {
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
	fmt.Fprintf(&b, "\033[1mcf-triage\033[0m  %s  worker %s  service %s  %s  %d tickets $%.2f\n\n", time.Now().Format("15:04:05"), worker, or(m.svc, "?"), jira, len(m.rows), total)
	fmt.Fprintf(&b, "\033[2m%-10s %-26s %5s %7s %-8s %-6s %s\033[0m\n", "KEY", "STAGE", "AGE", "COST", "VERDICT", "MR", "TITLE")
	end := min(m.off+m.visible(), len(m.rows))
	for i := m.off; i < m.visible()+m.off; i++ {
		if i >= end {
			b.WriteString("\n")
			continue
		}
		r := m.rows[i]
		mr := "-"
		if mm := mrIIDRe.FindStringSubmatch(r.mr); mm != nil {
			mr = "!" + mm[1]
		}
		line := fmt.Sprintf("%-10s %-26s %5s %7s %-8s %-6s %s", r.key, cut(r.stage, 26), ago(r.seen), fmt.Sprintf("$%.2f", r.cost), cut(or(r.verdict, "-"), 8), mr, r.summary)
		line = fmt.Sprintf("%-*s", m.w, cut(line, m.w))
		switch {
		case i == m.cur:
			line = "\033[7m" + line + "\033[0m"
		case r.running:
			line = "\033[1;32m" + line + "\033[0m"
		}
		b.WriteString(line + "\n")
	}
	r := m.sel()
	fmt.Fprintf(&b, "\033[2m%s\033[0m\n", strings.Repeat("─", max(m.w, 1)))
	fmt.Fprintf(&b, "\033[1m%s\033[0m  %s\n", r.key, cut(or(r.summary, "(no jira ticket)"), m.w-len(r.key)-2))
	fmt.Fprintf(&b, "jira  %s\n", or(jiraURL(r), "-"))
	fmt.Fprintf(&b, "mr    %s\n", or(r.mr, "-"))
	fmt.Fprintf(&b, "\033[2m%s\033[0m\n", cut(or(r.last, "-"), m.w))
	fmt.Fprintf(&b, "\033[2menter/o open MR|jira  J jira  m MR  l log  c resume session  r refresh  j/k move  q quit  click KEY/MR to open\033[0m  %s", m.status)
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(model{}, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
