package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"maps"
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

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

//go:embed web/index.html
var indexHTML []byte

var (
	md         = goldmark.New(goldmark.WithExtensions(extension.GFM), goldmark.WithRendererOptions(ghtml.WithHardWraps()))
	mediaDirRe = regexp.MustCompile(`^[A-Z]+-\d+\.(verify|layout)-media$`)
	mediaExtRe = regexp.MustCompile(`(?i)\.(jpe?g|png|webm|mp4)$`)
	wikiImgRe  = regexp.MustCompile(`!([^!|\s]+\.(?i:png|jpe?g|gif|webp|webm|mp4))(\|[^!\n]*)?!`)
	pasteRe    = regexp.MustCompile(`^img-\d+\.png$`)
	exitRe     = regexp.MustCompile(`✅ exit error=(\w+)`)
	turnsRe    = regexp.MustCompile(`turns=(\d+)`)
	peakRe     = regexp.MustCompile(`peak=(\d+)K`)
	toolRe     = regexp.MustCompile(`^🔧 (\S+)`)
	boardCols  = []string{"todo", "analysis", "plan gate", "implement", "verify", "review", "quiz", "merge", "nightly", "RC", "done"}
	webTabs    = []string{"log", "plan", "holdout", "description", "comments", "verify", "review"}
)

type server struct {
	mu            sync.Mutex
	issues, todos []issue
	jiraErr       error
	jiraAt        time.Time
	mrs           map[string]mrInfo
	status        string
	token         string
	atts          map[string]map[string]string
	reports       map[string]report
	kick          chan struct{}
	mrAt, started time.Time
	addr, mrErr   string
}

type report struct {
	html string
	at   time.Time
}

type rowOut struct {
	Key, Stage, Summary, Verdict, MR, MRIID, MRDetail, Last, Phase string
	Icon, Color, Label, Ball, Next, Col, JiraURL                   string
	Group                                                          int
	Cost                                                           float64
	Seen                                                           int64
	Running, Todo, Hidden, Resume, Jira                            bool
	Labels                                                         []string
}

func look(r row, mrs map[string]mrInfo) (icon, color, label string) {
	st := stageStyle[r.stage]
	if in, ok := mrs[r.mr]; ok && (mrStage(r.stage) || r.stage == "todo" || r.stage == "-") {
		if in.label == "conflict" {
			switch r.conflict {
			case "waiting":
				return in.icon, in.color, "conflict: b"
			case "failed":
				return "🔴", "31", "rebase failed"
			}
			return "🔧", "36", "rebase pending"
		}
		return in.icon, in.color, in.label
	}
	return st.icon, st.color, r.stage
}

var nextSteps = map[string][2]string{
	"questions":       {"you", "answer the questions in Jira (R)"},
	"plan gate":       {"you", "approve plan (a) or reply (R)"},
	"quiz":            {"you", "take the quiz (Q)"},
	"reviewed":        {"you", "gate green: mark ready, get it merged (m)"},
	"verify failed":   {"you", "read verify tab, then retry (t) or reply (R)"},
	"review findings": {"you", "read review tab, then retry (t)"},
	"analysis failed": {"you", "read log, then retry (t)"},
	"bailed":          {"you", "read log, then retry (t)"},
	"todo":            {"you", "feed to factory (f)"},
	"conflict: b":     {"you", "let the agent resolve it (b)"},
	"rebase failed":   {"you", "resolve the conflict by hand"},
	"gate failed":     {"you", "finish quiz (Q) or skip gate (s)"},
	"gate pending":    {"you", "finish quiz (Q) or skip gate (s)"},
	"ci failed":       {"you", "check the pipeline (m)"},
	"changes asked":   {"you", "address reviewer changes (m)"},
	"open threads":    {"you", "resolve MR threads (m)"},
	"no reviewer":     {"you", "assign a reviewer (m)"},
	"approved":        {"you", "merge (m)"},
	"in review":       {"reviewer", "waiting for review"},
	"ci running":      {"ci", "pipeline running"},
	"draft":           {"factory", "draft MR: verify + review pending"},
	"rebase pending":  {"factory", "ai-rebase will resolve"},
	"rebase queued":   {"factory", "ai-rebase will resolve"},
	"auto-rebase":     {"factory", "ai-rebase will resolve"},
	"new":             {"factory", "analysis queued"},
	"implementing":    {"factory", "implementation queued"},
	"mr":              {"factory", "holdout verify queued"},
	"verified":        {"factory", "independent review queued"},
	"review fix":      {"factory", "fixing review findings"},
	"re-verify":       {"factory", "re-verify after review fix"},
	"nightly":         {"-", "on nightly"},
	"RC":              {"-", "in release candidate"},
	"done":            {"-", "done"},
	"merged":          {"-", "merged"},
	"closed":          {"-", "MR closed"},
}

func column(stage string) string {
	switch stage {
	case "todo", "nightly", "RC", "done", "quiz", "plan gate":
		return stage
	case "new", "questions", "analysis failed":
		return "analysis"
	case "implementing", "bailed":
		return "implement"
	case "mr", "verify failed", "re-verify":
		return "verify"
	case "verified", "review findings", "review fix":
		return "review"
	case "reviewed":
		return "merge"
	}
	return ""
}

func (s *server) rows() []row {
	s.mu.Lock()
	issues, todos, mrs := s.issues, s.todos, maps.Clone(s.mrs)
	s.mu.Unlock()
	rows := collect(issues, todos, workerBusy(), runningUnits())
	for i := range rows {
		_, _, l := look(rows[i], mrs)
		rows[i].mrNeeds = l == "conflict: b" || l == "rebase failed"
	}
	sort.SliceStable(rows, func(a, b int) bool { return group(rows[a]) < group(rows[b]) })
	return rows
}

func (s *server) out(r row) rowOut {
	s.mu.Lock()
	mrs := s.mrs
	icon, color, label := look(r, mrs)
	in, hasMR := mrs[r.mr]
	s.mu.Unlock()
	o := rowOut{Key: r.key, Stage: r.stage, Summary: r.summary, Verdict: r.verdict, MR: r.mr, Last: r.last, Phase: r.phase,
		Icon: icon, Color: color, Label: label, Col: column(r.stage), JiraURL: jiraURL(r), Group: group(r), Cost: r.cost,
		Running: r.running, Todo: r.todo, Hidden: r.stage == "RC" || r.stage == "done", Resume: r.resume != "", Jira: r.jira, Labels: r.labels}
	if !r.seen.IsZero() {
		o.Seen = r.seen.Unix()
	}
	if m := mrIIDRe.FindStringSubmatch(r.mr); m != nil {
		o.MRIID = m[1]
	}
	if hasMR {
		o.MRDetail = in.detail
		if !mrStage(r.stage) {
			o.MRDetail = in.icon + " " + in.label + " · " + in.detail
		}
	}
	if o.Hidden {
		o.Group = gParked
	}
	n := nextSteps[label]
	o.Ball, o.Next = or(n[0], "-"), n[1]
	if r.running {
		o.Ball, o.Next = "factory", "running: "+or(r.phase, "worker")
	}
	return o
}

func (s *server) find(key string) (row, bool) {
	for _, r := range s.rows() {
		if r.key == key {
			return r, true
		}
	}
	return row{}, false
}

func (s *server) refresh() {
	is, td, err := fetchJira()
	s.mu.Lock()
	s.jiraAt, s.jiraErr = time.Now(), err
	if err == nil {
		s.issues, s.todos = is, td
	}
	s.mu.Unlock()
	var wg sync.WaitGroup
	mrErr := ""
	for _, r := range s.rows() {
		s.mu.Lock()
		in, ok := s.mrs[r.mr]
		s.mu.Unlock()
		if r.mr == "" || r.parked || ok && in.final {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			in, err := fetchMR(r.mr)
			s.mu.Lock()
			defer s.mu.Unlock()
			if err != nil {
				mrErr = r.key + ": " + err.Error()
				return
			}
			s.mrs[r.mr] = in
		}()
	}
	wg.Wait()
	s.mu.Lock()
	s.mrAt, s.mrErr = time.Now(), mrErr
	s.mu.Unlock()
}

func (s *server) loop() {
	for {
		s.refresh()
		select {
		case <-time.After(time.Minute):
		case <-s.kick:
		}
	}
}

func (s *server) poke() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *server) setStatus(st string) {
	s.mu.Lock()
	s.status = st
	s.mu.Unlock()
}

func mdHTML(src string) string {
	var b bytes.Buffer
	if md.Convert([]byte(src), &b) != nil {
		return "<pre>" + html.EscapeString(src) + "</pre>"
	}
	return b.String()
}

func logHTML(lines []string) string {
	var b strings.Builder
	var say []string
	inSay := false
	flush := func() {
		if len(say) > 0 {
			b.WriteString(`<div class="say">` + mdHTML(strings.Join(say, "\n")) + `</div>`)
		}
		say, inSay = nil, false
	}
	line := func(class, ts, l string) {
		fmt.Fprintf(&b, `<div class="l %s"><span class="ts">%s</span> %s</div>`, class, ts, html.EscapeString(l))
	}
	for _, raw := range lines {
		ts, l := splitTS(raw)
		if isMarker(l) {
			flush()
		}
		switch {
		case strings.HasPrefix(l, "💬"), inSay && !isMarker(l):
			say, inSay = append(say, strings.TrimSpace(ts+" "+l)), true
		case strings.HasPrefix(l, "🔧"):
			name, args, _ := strings.Cut(strings.TrimPrefix(l, "🔧 "), " ")
			fmt.Fprintf(&b, `<div class="l tool"><span class="ts">%s</span> 🔧 <b>%s</b> %s</div>`, ts, html.EscapeString(name), html.EscapeString(args))
		case strings.HasPrefix(l, "✅"):
			line("ok", ts, l)
			inSay = true
		case strings.HasPrefix(l, "==="):
			line("head", ts, l)
		default:
			line("", ts, l)
		}
	}
	flush()
	return b.String()
}

type mediaOut struct {
	URL, Name, Kind, When string
	Current, Stale        bool
	Chapters              []chapter
	at                    time.Time
}

type chapter struct {
	Start, End     float64
	Title, Verdict string
}

var (
	chapterRe = regexp.MustCompile(`(?m)^START=(\d+)\nEND=(\d+)\ntitle=(.*)$`)
	verdictRe = regexp.MustCompile(`(?m)^(S\d+)\b[^:\n]*: (pass|fail|blocked|not run)`)
)

func chapters(key string) []chapter {
	b, err := os.ReadFile(filepath.Join(stateDir, key+".verify-media", "narrate", "chapters.txt"))
	if err != nil {
		return nil
	}
	verdicts := map[string]string{}
	if r, err := os.ReadFile(filepath.Join(stateDir, key+".verify.md")); err == nil {
		for _, m := range verdictRe.FindAllStringSubmatch(string(r), -1) {
			verdicts[m[1]] = m[2]
		}
	}
	var out []chapter
	for _, m := range chapterRe.FindAllStringSubmatch(string(b), -1) {
		s, _ := strconv.ParseFloat(m[1], 64)
		e, _ := strconv.ParseFloat(m[2], 64)
		id, _, _ := strings.Cut(m[3], " ")
		out = append(out, chapter{s / 1000, e / 1000, m[3], verdicts[id]})
	}
	return out
}

var reportVideoRe = regexp.MustCompile(`(?m)^Video: .*?([^/\s]+)\s*$`)

func mediaFiles(key string) []mediaOut {
	var out []mediaOut
	report := filepath.Join(stateDir, key+".verify.md")
	var reportAt time.Time
	if fi, err := os.Stat(report); err == nil {
		reportAt = fi.ModTime()
	}
	current := ""
	if b, err := os.ReadFile(report); err == nil {
		if m := reportVideoRe.FindAllStringSubmatch(string(b), -1); m != nil {
			current = m[len(m)-1][1]
		}
	}
	newest := -1
	for _, kind := range []string{"verify", "layout"} {
		dir := key + "." + kind + "-media"
		es, _ := os.ReadDir(filepath.Join(stateDir, dir))
		for _, e := range es {
			fi, err := e.Info()
			if err != nil || e.IsDir() || !mediaExtRe.MatchString(e.Name()) {
				continue
			}
			m := mediaOut{URL: "/media/" + dir + "/" + url.PathEscape(e.Name()), Name: e.Name(), Kind: "image", When: fi.ModTime().Format("2006-01-02 15:04"), at: fi.ModTime()}
			m.Stale = kind == "verify" && !reportAt.IsZero() && reportAt.Sub(m.at) > time.Hour
			if kind == "layout" {
				m.Name = "layout · " + m.Name
			}
			if strings.HasSuffix(e.Name(), ".webm") || strings.HasSuffix(e.Name(), ".mp4") {
				m.Kind, m.Current = "video", kind == "verify" && e.Name() == current
				if kind == "verify" && (newest < 0 || m.at.After(out[newest].at)) {
					newest = len(out)
				}
			}
			out = append(out, m)
		}
	}
	if newest >= 0 && !slices.ContainsFunc(out, func(m mediaOut) bool { return m.Current }) {
		out[newest].Current = true
	}
	for i := range out {
		if out[i].Current && strings.Contains(out[i].Name, ".short.") {
			out[i].Chapters = chapters(key)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Current != out[b].Current {
			return out[a].Current
		}
		return out[a].Kind > out[b].Kind
	})
	return out
}

func linkMedia(key, src string) string {
	re := regexp.MustCompile(`\S*(` + regexp.QuoteMeta(key) + `\.(?:verify|layout)-media/[\w.\-]+\.(?i:jpe?g|png|webm|mp4))`)
	return re.ReplaceAllString(src, "[$1](/media/$1)")
}

func (s *server) body(key, tab string, full bool) (htm, src string, media []mediaOut) {
	read := func(p string) string { b, _ := os.ReadFile(p); return string(b) }
	switch tab {
	case "log":
		src = read(filepath.Join(stateDir, key+".log"))
		lines := strings.Split(strings.TrimRight(src, "\n"), "\n")
		if tail := 1500; !full && len(lines) > tail {
			htm = fmt.Sprintf(`<div class="l head">… %d earlier lines hidden, l loads the full log</div>`, len(lines)-tail)
			lines = lines[len(lines)-tail:]
		}
		return htm + logHTML(lines), src, nil
	case "plan":
		src = "_(no PLAN.md in a worktree for " + key + ")_"
		if p := planPath(key); p != "" {
			src = "`" + p + "`\n\n" + read(p)
		}
	case "holdout":
		src = "_(no holdout scenarios for " + key + ")_"
		var parts []string
		for _, f := range []string{key + ".md", key + ".api.md"} {
			p := filepath.Join(stateDir, "holdout", f)
			if b, err := os.ReadFile(p); err == nil {
				parts = append(parts, "`"+p+"`\n\n"+string(b))
			}
		}
		if len(parts) > 0 {
			src = strings.Join(parts, "\n\n---\n\n")
		}
	case "comments":
		h, cm, err := comments(key)
		if err != nil {
			return mdHTML("**error:** " + err.Error()), "", nil
		}
		return or(h, "<p><em>(no comments)</em></p>"), cm, nil
	case "description":
		desc, _, err := fetchIssue(key)
		src = or(desc, "_(no description)_")
		if err != nil {
			src = "**error:** " + err.Error()
		}
		return mdHTML(attImages(key, src)), src, nil
	case "verify":
		src = or(read(filepath.Join(stateDir, key+".verify.md")), "_(no verify report for "+key+")_")
		media = mediaFiles(key)
		return mdHTML(linkMedia(key, src)), src, media
	case "review":
		src = or(read(filepath.Join(stateDir, key+".review.md")), "_(no review report for "+key+")_")
	}
	return mdHTML(src), src, nil
}

func attImages(key, src string) string {
	return wikiImgRe.ReplaceAllStringFunc(src, func(m string) string {
		name := wikiImgRe.FindStringSubmatch(m)[1]
		return "![" + name + "](/att?key=" + key + "&name=" + url.QueryEscape(name) + ")"
	})
}

var myAccount = sync.OnceValue(func() string {
	var u struct {
		AccountID string `json:"accountId"`
	}
	jiraReq("GET", "/rest/api/2/myself", nil, &u)
	return u.AccountID
})

func comments(key string) (htm, src string, err error) {
	var out struct {
		Fields struct {
			Comment struct {
				Comments []struct {
					Author struct {
						DisplayName string
						AccountID   string `json:"accountId"`
					}
					Created, Body string
				}
			}
		}
	}
	if err := jiraReq("GET", "/rest/api/2/issue/"+key+"?fields=comment", nil, &out); err != nil {
		return "", "", err
	}
	var h, s strings.Builder
	for _, c := range out.Fields.Comment.Comments {
		body := wikiToMD(mentions(strings.ReplaceAll(c.Body, "\r", "")))
		ts := strings.Replace(c.Created[:min(16, len(c.Created))], "T", " ", 1)
		who, name := "other", c.Author.DisplayName
		switch {
		case strings.HasPrefix(strings.TrimSpace(c.Body), "🤖"):
			who, name = "ai", "AI"
		case c.Author.AccountID == myAccount():
			who, name = "me", "You"
		}
		fmt.Fprintf(&s, "\n---\n\n**%s · %s**\n\n%s\n", c.Author.DisplayName, ts, body)
		fmt.Fprintf(&h, `<div class="cm %s"><div class="who"><b>%s</b><span>%s</span></div>%s</div>`, who, html.EscapeString(name), ts, mdHTML(attImages(key, body)))
	}
	return h.String(), s.String(), nil
}

func (s *server) attachment(w http.ResponseWriter, key, name string) {
	s.mu.Lock()
	u := s.atts[key][name]
	s.mu.Unlock()
	if u == "" {
		var out struct {
			Fields struct {
				Attachment []struct{ Filename, Content string }
			}
		}
		if err := jiraReq("GET", "/rest/api/2/issue/"+key+"?fields=attachment", nil, &out); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		m := map[string]string{}
		for _, a := range out.Fields.Attachment {
			m[a.Filename] = a.Content
		}
		s.mu.Lock()
		s.atts[key] = m
		s.mu.Unlock()
		u = m[name]
	}
	if u == "" {
		http.NotFound(w, nil)
		return
	}
	req, _ := http.NewRequest("GET", u, nil)
	req.SetBasicAuth(os.Getenv("JIRA_EMAIL"), os.Getenv("JIRA_TOKEN"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func tmuxWindow(detached bool, name, cmd string) error {
	out, err := exec.Command("tmux", "list-clients", "-F", "#{client_activity} #{session_name}").Output()
	sess, best := "", int64(-1)
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		a, n, ok := strings.Cut(l, " ")
		if t, perr := strconv.ParseInt(a, 10, 64); ok && perr == nil && t > best {
			sess, best = n, t
		}
	}
	if err != nil || sess == "" {
		return fmt.Errorf("no attached tmux client to open %s in", name)
	}
	args := []string{"new-window"}
	if detached {
		args = append(args, "-d")
	}
	if b, err := exec.Command("tmux", append(args, "-t", sess+":", "-n", name, cmd)...).CombinedOutput(); err != nil {
		return fmt.Errorf("tmux: %s", strings.TrimSpace(string(b)))
	}
	return nil
}

type actionReq struct {
	Key, Action, Repo, Tab, Text string
	Dry                          bool
	Images                       map[string]string
}

func (s *server) editFiles(r row, tab string) error {
	var files []string
	switch tab {
	case "plan":
		if p := planPath(r.key); p != "" {
			files = []string{p}
		}
	case "holdout":
		for _, f := range []string{r.key + ".md", r.key + ".api.md"} {
			if p := filepath.Join(stateDir, "holdout", f); func() bool { _, err := os.Stat(p); return err == nil }() {
				files = append(files, p)
			}
		}
		if len(files) == 0 {
			files = []string{filepath.Join(stateDir, "holdout", r.key+".md")}
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("nothing to edit on this tab")
	}
	before := map[string]string{}
	var argv []string
	for _, a := range editor() {
		argv = append(argv, shq(a))
	}
	for _, p := range files {
		b, _ := os.ReadFile(p)
		before[p] = string(b)
		argv = append(argv, shq(p))
	}
	ch := "factory-dash-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := tmuxWindow(false, "edit:"+r.key, strings.Join(argv, " ")+"; tmux wait-for -S "+ch); err != nil {
		return err
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Hour)
		defer cancel()
		if exec.CommandContext(ctx, "tmux", "wait-for", ch).Run() != nil {
			return
		}
		if tab != "holdout" || !jiraKey.MatchString(r.key) {
			s.setStatus("edited " + r.key + " " + tab)
			return
		}
		var sent []string
		for _, p := range files {
			b, err := os.ReadFile(p)
			if err != nil || string(b) == before[p] {
				continue
			}
			name := strings.Replace(filepath.Base(p), r.key, "holdout", 1)
			if err := jiraAttach(r.key, p, name); err != nil {
				s.setStatus("error: " + err.Error())
				return
			}
			sent = append(sent, name)
		}
		if len(sent) == 0 {
			s.setStatus("edited, unchanged")
			return
		}
		s.setStatus("edited, attached to Jira: " + strings.Join(sent, ", "))
	}()
	return nil
}

func (s *server) reply(key, text string, images map[string]string) error {
	dir, err := os.MkdirTemp("", "factory-reply-"+key+"-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	for name, data := range images {
		_, b64, ok := strings.Cut(data, ";base64,")
		raw, derr := base64.StdEncoding.DecodeString(b64)
		if !pasteRe.MatchString(name) || !ok || derr != nil {
			return fmt.Errorf("bad pasted image %s", name)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
			return err
		}
		text = strings.ReplaceAll(text, "!"+name+"!", "!"+filepath.Join(dir, name)+"!")
	}
	body, err := uploadImages(text, dir, "reply-"+time.Now().Format("20060102-150405"), func(p, name string) error { return jiraAttach(key, p, name) })
	if err != nil {
		return err
	}
	return jiraReq("POST", "/rest/api/2/issue/"+key+"/comment", map[string]string{"body": body}, nil)
}

func (s *server) action(a actionReq) (ask string, run func() (string, error), kick bool, err error) {
	if a.Action == "refresh" {
		return "", func() (string, error) { return "refreshing jira", nil }, false, nil
	}
	r, ok := s.find(a.Key)
	if !ok {
		return "", nil, false, fmt.Errorf("unknown ticket %s", a.Key)
	}
	isTicket := jiraKey.MatchString(r.key) && r.jira
	s.mu.Lock()
	in, hasMR := s.mrs[r.mr]
	s.mu.Unlock()
	switch a.Action {
	case "approve":
		if !isTicket || !slices.Contains(r.labels, "triaged") || slices.Contains(r.labels, "ai-fix") || slices.Contains(r.labels, "ai-mr") {
			return "", nil, false, fmt.Errorf("not waiting at plan gate")
		}
		return "approve plan + scenarios: add ai-fix to " + r.key + "?", func() (string, error) { return "ai-fix added to " + r.key, relabel(r.key, "", "ai-fix") }, true, nil
	case "retry":
		rm, add := retryAction(r.labels)
		if !isTicket || rm == "" {
			return "", nil, false, fmt.Errorf("nothing failed to retry")
		}
		return fmt.Sprintf("retry %s: remove %s, add %s?", r.key, rm, or(add, "nothing (re-verify)")), func() (string, error) { return "relabelled " + r.key, relabel(r.key, rm, add) }, true, nil
	case "rebase":
		if !hasMR || in.label != "conflict" {
			return "", nil, false, fmt.Errorf("b only works on an MR in conflict")
		}
		iid := mrIIDRe.FindStringSubmatch(r.mr)[1]
		return "let the agent resolve the code conflict in !" + iid + " (adds MR label ai-rebase)?", func() (string, error) { return "ai-rebase added to !" + iid, addMRLabel(r.mr, "ai-rebase") }, true, nil
	case "skipgate":
		if !hasMR || (in.label != "gate pending" && in.label != "gate failed") {
			return "", nil, false, fmt.Errorf("s only works on an MR held by the factory gate")
		}
		iid := mrIIDRe.FindStringSubmatch(r.mr)[1]
		return "skip the factory gate on !" + iid + " (adds MR label ai-gate-skip, next tick sets it green)?", func() (string, error) { return "ai-gate-skip added to !" + iid, addMRLabel(r.mr, "ai-gate-skip") }, true, nil
	case "feed":
		if !r.todo {
			return "", nil, false, fmt.Errorf("f feeds a TODO ticket")
		}
		if !slices.Contains(repos(), a.Repo) {
			return "", nil, false, fmt.Errorf("unknown repo %q", a.Repo)
		}
		return "feed " + r.key + " to factory: add triage + repo-" + a.Repo + "?", func() (string, error) { return r.key + " fed to factory (repo-" + a.Repo + ")", feed(r.key, a.Repo) }, true, nil
	case "reply":
		text := strings.TrimSpace(a.Text)
		if !isTicket {
			return "", nil, false, fmt.Errorf("no jira ticket")
		}
		if text == "" {
			return "", nil, false, fmt.Errorf("reply empty, not sent")
		}
		imgs := ""
		if len(a.Images) > 0 {
			imgs = fmt.Sprintf(" + %d image(s)", len(a.Images))
		}
		return fmt.Sprintf("post %d-char comment%s to %s?", len(text), imgs, r.key), func() (string, error) { return "comment posted to " + r.key, s.reply(r.key, text, a.Images) }, true, nil
	case "worker":
		if !isTicket {
			return "", nil, false, fmt.Errorf("no jira ticket")
		}
		return "run factory-worker " + r.key + " now (new tmux window)?", func() (string, error) {
			return "worker started for " + r.key, tmuxWindow(true, "w:"+r.key, "~/bin/factory-worker "+r.key+"; echo; read '?done, enter closes'")
		}, false, nil
	case "quiz":
		if !isTicket || r.stage != "quiz" {
			return "", nil, false, fmt.Errorf("Q works on a reviewed MR waiting for quiz")
		}
		return "", func() (string, error) {
			return "quiz opened in tmux", tmuxWindow(false, "quiz:"+r.key, "claude --model sonnet '/cf-quiz "+r.key+"'")
		}, false, nil
	case "resume":
		if r.resume == "" {
			return "", nil, false, fmt.Errorf("no session in log")
		}
		return "", func() (string, error) { return "session opened in tmux", tmuxWindow(false, r.key, r.resume) }, false, nil
	case "edit":
		return "", func() (string, error) { return "editor opened in tmux", s.editFiles(r, a.Tab) }, false, nil
	}
	return "", nil, false, fmt.Errorf("unknown action %q", a.Action)
}

type runRec struct {
	key, stage string
	start      time.Time
	dur        time.Duration
	cost       float64
	turns      int
	peak       int
	fail, done bool
	tools      map[string]int
}

func parseRuns(path string) []runRec {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []runRec
	var cur *runRec
	for _, raw := range strings.Split(string(b), "\n") {
		ts, l := splitTS(raw)
		if strings.HasPrefix(l, "=== ") {
			f := strings.Fields(l)
			if len(f) < 3 {
				continue
			}
			r := runRec{tools: map[string]int{}}
			switch {
			case jiraKey.MatchString(f[1]):
				r.stage, r.key = "analyze", f[1]
			case strings.HasPrefix(f[1], "http"):
				r.stage, r.key = "conflict", strings.TrimSuffix(filepath.Base(path), ".log")
			default:
				r.stage, r.key = f[1], f[2]
			}
			r.start, _ = time.Parse(time.RFC3339, f[len(f)-1])
			out = append(out, r)
			cur = &out[len(out)-1]
			continue
		}
		if cur == nil || cur.done {
			continue
		}
		if m := toolRe.FindStringSubmatch(l); m != nil {
			cur.tools[m[1]]++
		}
		if m := exitRe.FindStringSubmatch(l); m != nil {
			cur.done, cur.fail = true, m[1] != "false"
			if c := costRe.FindStringSubmatch(l); c != nil {
				cur.cost, _ = strconv.ParseFloat(c[1], 64)
			}
			if t := turnsRe.FindStringSubmatch(l); t != nil {
				cur.turns, _ = strconv.Atoi(t[1])
			}
			if p := peakRe.FindStringSubmatch(l); p != nil {
				cur.peak, _ = strconv.Atoi(p[1])
			}
			if end, err := time.ParseInLocation("2006-01-02 15:04:05", cur.start.Format("2006-01-02 ")+ts, cur.start.Location()); err == nil && !cur.start.IsZero() {
				for end.Before(cur.start) {
					end = end.Add(24 * time.Hour)
				}
				cur.dur = end.Sub(cur.start)
			}
		}
	}
	return out
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sort.Float64s(v)
	if n := len(v); n%2 == 0 {
		return (v[n/2-1] + v[n/2]) / 2
	}
	return v[len(v)/2]
}

type stageStat struct {
	Stage                          string
	Runs, Fails                    int
	Cost, MedCost, MedMin, MedPeak float64
	MedTurns                       float64
}

type nameN struct {
	Name string
	N    int
	Runs int
	Cost float64
}

type statsOut struct {
	Days, Runs, Tickets, Fails int
	Cost                       float64
	ByDay                      []nameN
	Stages                     []stageStat
	Tools                      []nameN
	TopTickets                 []nameN
}

func stats(days int) statsOut {
	since := time.Now().AddDate(0, 0, -days)
	logs, _ := filepath.Glob(filepath.Join(stateDir, "*.log"))
	o := statsOut{Days: days}
	day, tools, toolRuns, tick, tickRuns := map[string]*nameN{}, map[string]int{}, map[string]int{}, map[string]float64{}, map[string]int{}
	type agg struct {
		n, fails                int
		cost, mins, peak, turns []float64
	}
	st := map[string]*agg{}
	for _, p := range logs {
		if fi, err := os.Stat(p); err != nil || fi.ModTime().Before(since) {
			continue
		}
		for _, r := range parseRuns(p) {
			if r.start.Before(since) || !r.done {
				continue
			}
			o.Runs++
			o.Cost += r.cost
			d := r.start.Format("2006-01-02")
			if day[d] == nil {
				day[d] = &nameN{Name: d}
			}
			day[d].N++
			day[d].Cost += r.cost
			tick[r.key] += r.cost
			tickRuns[r.key]++
			a := st[r.stage]
			if a == nil {
				a = &agg{}
				st[r.stage] = a
			}
			a.n++
			a.cost = append(a.cost, r.cost)
			if r.dur > 0 {
				a.mins = append(a.mins, r.dur.Minutes())
			}
			if r.peak > 0 {
				a.peak = append(a.peak, float64(r.peak))
			}
			if r.turns > 0 {
				a.turns = append(a.turns, float64(r.turns))
			}
			if r.fail {
				a.fails++
				o.Fails++
			}
			for t, n := range r.tools {
				tools[t] += n
				toolRuns[t]++
			}
		}
	}
	o.Tickets = len(tick)
	for d := since.AddDate(0, 0, 1); !d.After(time.Now()); d = d.AddDate(0, 0, 1) {
		k := d.Format("2006-01-02")
		if day[k] == nil {
			day[k] = &nameN{Name: k}
		}
		o.ByDay = append(o.ByDay, *day[k])
	}
	for s, a := range st {
		sum := 0.0
		for _, c := range a.cost {
			sum += c
		}
		o.Stages = append(o.Stages, stageStat{s, a.n, a.fails, sum, median(a.cost), median(a.mins), median(a.peak), median(a.turns)})
	}
	sort.Slice(o.Stages, func(a, b int) bool { return o.Stages[a].Cost > o.Stages[b].Cost })
	for t, n := range tools {
		o.Tools = append(o.Tools, nameN{Name: t, N: n, Runs: toolRuns[t]})
	}
	sort.Slice(o.Tools, func(a, b int) bool { return o.Tools[a].N > o.Tools[b].N })
	for k, c := range tick {
		o.TopTickets = append(o.TopTickets, nameN{Name: k, Cost: c, Runs: tickRuns[k]})
	}
	sort.Slice(o.TopTickets, func(a, b int) bool { return o.TopTickets[a].Cost > o.TopTickets[b].Cost })
	o.TopTickets = o.TopTickets[:min(len(o.TopTickets), 15)]
	return o
}

func (s *server) report(name string) string {
	s.mu.Lock()
	c, ok := s.reports[name]
	s.mu.Unlock()
	if ok && time.Since(c.at) < 10*time.Minute {
		return c.html
	}
	out, err := exec.Command(filepath.Join(os.Getenv("HOME"), "bin", name)).CombinedOutput()
	h := mdHTML(string(out))
	if err != nil {
		h = "<p><b>" + html.EscapeString(name+": "+err.Error()) + "</b></p>" + h
	}
	s.mu.Lock()
	s.reports[name] = report{h, time.Now()}
	s.mu.Unlock()
	return h
}

type statusRow struct{ Name, Value, Level string }

func sysShow(unit string, props ...string) map[string]string {
	out, _ := exec.Command("systemctl", "--user", "show", unit, "-p", strings.Join(props, ",")).Output()
	m := map[string]string{}
	for _, l := range strings.Split(string(out), "\n") {
		if k, v, ok := strings.Cut(l, "="); ok {
			m[k] = v
		}
	}
	return m
}

func (s *server) statusRows() []statusRow {
	s.mu.Lock()
	jiraAt, jiraErr, nIssues, nTodos, nMRs, mrAt, mrErr := s.jiraAt, s.jiraErr, len(s.issues), len(s.todos), len(s.mrs), s.mrAt, s.mrErr
	s.mu.Unlock()
	at := func(t time.Time) string {
		if t.IsZero() {
			return "never"
		}
		return ago(t) + " ago (" + t.Format("15:04:05") + ")"
	}
	jira := statusRow{"Jira", fmt.Sprintf("fetched %s · %d factory tickets, %d todos · refetched every minute, r forces", at(jiraAt), nIssues, nTodos), "ok"}
	switch {
	case jiraErr != nil:
		jira.Value, jira.Level = jiraErr.Error()+" · last try "+at(jiraAt), "bad"
	case time.Since(jiraAt) > 3*time.Minute:
		jira.Level = "warn"
	}
	gl := statusRow{"GitLab MRs", fmt.Sprintf("%d tracked · fetched %s", nMRs, at(mrAt)), "ok"}
	if mrErr != "" {
		gl.Value, gl.Level = gl.Value+" · last error "+mrErr, "bad"
	}
	rows := []statusRow{jira, gl}
	t := sysShow("sw-factory.timer", "ActiveState", "NextElapseUSecRealtime", "LastTriggerUSec")
	timer := statusRow{"Dispatcher timer", fmt.Sprintf("%s · next %s · last %s", or(t["ActiveState"], "unknown"), or(t["NextElapseUSecRealtime"], "-"), or(t["LastTriggerUSec"], "-")), "ok"}
	if t["ActiveState"] != "active" {
		timer.Level = "bad"
	}
	d := sysShow("sw-factory.service", "ActiveState", "Result", "ExecMainExitTimestamp", "ExecMainStatus")
	disp := statusRow{"Dispatcher run", fmt.Sprintf("%s · result %s · exit code %s · finished %s", or(d["ActiveState"], "unknown"), or(d["Result"], "-"), or(d["ExecMainStatus"], "-"), or(d["ExecMainExitTimestamp"], "-")), "ok"}
	if d["ActiveState"] == "failed" || d["Result"] != "success" {
		disp.Level = "bad"
	}
	worker := statusRow{"Worker process", "idle", ""}
	if workerBusy() {
		worker = statusRow{"Worker process", "busy", "ok"}
	}
	rows = append(rows, timer, disp, worker)
	units, _ := exec.Command("systemctl", "--user", "list-units", "sw-factory-*", "--plain", "--no-legend", "--all").Output()
	n := 0
	for _, l := range strings.Split(string(units), "\n") {
		f := strings.Fields(l)
		if len(f) < 4 {
			continue
		}
		n++
		r := statusRow{"Stage unit", strings.TrimSuffix(f[0], ".service") + " · " + f[2] + " (" + f[3] + ")", "ok"}
		if f[2] == "failed" {
			r.Level = "bad"
		}
		rows = append(rows, r)
	}
	if n == 0 {
		rows = append(rows, statusRow{"Stage units", "none running", ""})
	}
	return append(rows, statusRow{"Dashboard", "serving " + s.addr + " · up since " + s.started.Format("2006-01-02 15:04"), "ok"})
}

func loadToken() string {
	p := filepath.Join(stateDir, "dash.token")
	if b, err := os.ReadFile(p); err == nil && len(bytes.TrimSpace(b)) >= 32 {
		return string(bytes.TrimSpace(b))
	}
	b := make([]byte, 24)
	rand.Read(b)
	t := hex.EncodeToString(b)
	os.WriteFile(p, []byte(t+"\n"), 0o600)
	return t
}

func (s *server) guard(next http.Handler) http.Handler {
	eq := func(v string) bool { return subtle.ConstantTimeCompare([]byte(v), []byte(s.token)) == 1 }
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, _ := strings.Cut(r.Host, ":")
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "bad host", http.StatusForbidden)
			return
		}
		if t := r.URL.Query().Get("t"); t != "" && r.URL.Path == "/" && eq(t) {
			http.SetCookie(w, &http.Cookie{Name: "fd", Value: t, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 365 * 86400})
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		if c, err := r.Cookie("fd"); err != nil || !eq(c.Value) {
			http.Error(w, "not signed in: run `factory-dash` to open the dashboard", http.StatusUnauthorized)
			return
		}
		if o := r.Header.Get("Origin"); r.Method != http.MethodGet && o != "http://"+r.Host {
			http.Error(w, "bad origin", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		rows := s.rows()
		out := make([]rowOut, 0, len(rows))
		for _, r := range rows {
			out = append(out, s.out(r))
		}
		svc, _ := exec.Command("systemctl", "--user", "is-active", "sw-factory.service").Output()
		s.mu.Lock()
		defer s.mu.Unlock()
		jerr := ""
		if s.jiraErr != nil {
			jerr = s.jiraErr.Error()
		}
		writeJSON(w, map[string]any{"Rows": out, "Busy": workerBusy(), "Svc": strings.TrimSpace(string(svc)), "JiraErr": jerr,
			"JiraAge": int(time.Since(s.jiraAt).Seconds()), "Status": s.status, "Repos": repos(), "Cols": boardCols, "Tabs": webTabs})
	})
	mux.HandleFunc("GET /api/ticket", func(w http.ResponseWriter, r *http.Request) {
		key, tab := r.URL.Query().Get("key"), r.URL.Query().Get("tab")
		if !regexp.MustCompile(`^[\w.-]+$`).MatchString(key) || strings.Contains(key, "..") || !slices.Contains(webTabs, tab) {
			http.Error(w, "bad key or tab", http.StatusBadRequest)
			return
		}
		h, src, media := s.body(key, tab, r.URL.Query().Get("full") != "")
		writeJSON(w, map[string]any{"HTML": h, "Src": src, "Media": media})
	})
	mux.HandleFunc("GET /api/lookup", func(w http.ResponseWriter, r *http.Request) {
		key := strings.ToUpper(r.URL.Query().Get("key"))
		if !jiraKey.MatchString(key) {
			http.Error(w, "bad key", http.StatusBadRequest)
			return
		}
		is, err := search("key = " + key)
		if err != nil || len(is) == 0 {
			http.Error(w, "not found in Jira: "+key, http.StatusNotFound)
			return
		}
		for _, row := range collect(is, nil, false, runningUnits()) {
			if row.key == key {
				writeJSON(w, s.out(row))
				return
			}
		}
	})
	mux.HandleFunc("GET /att", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if !jiraKey.MatchString(key) {
			http.Error(w, "bad key", http.StatusBadRequest)
			return
		}
		s.attachment(w, key, r.URL.Query().Get("name"))
	})
	mux.HandleFunc("GET /media/{dir}/{file}", func(w http.ResponseWriter, r *http.Request) {
		dir, file := r.PathValue("dir"), r.PathValue("file")
		if !mediaDirRe.MatchString(dir) || !mediaExtRe.MatchString(file) {
			http.NotFound(w, r)
			return
		}
		root, err := os.OpenRoot(filepath.Join(stateDir, dir))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer root.Close()
		if ext := strings.ToLower(filepath.Ext(file)); ext == ".webm" || ext == ".mp4" {
			w.Header().Set("Content-Type", "video/"+ext[1:])
		}
		http.ServeFileFS(w, r, root.FS(), file)
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, s.statusRows())
	})
	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		days, _ := strconv.Atoi(r.URL.Query().Get("days"))
		writeJSON(w, stats(min(max(days, 1), 365)))
	})
	mux.HandleFunc("GET /api/report", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name != "factory-stats" && name != "factory-compare" {
			http.Error(w, "bad report", http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]string{"HTML": s.report(name)})
	})
	mux.HandleFunc("POST /api/action", func(w http.ResponseWriter, r *http.Request) {
		var a actionReq
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 40<<20)).Decode(&a); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ask, run, kick, err := s.action(a)
		switch {
		case err != nil:
			writeJSON(w, map[string]string{"Err": err.Error()})
		case ask != "" && a.Dry:
			writeJSON(w, map[string]string{"Ask": ask})
		default:
			st, err := run()
			if err != nil {
				writeJSON(w, map[string]string{"Err": err.Error()})
				return
			}
			if kick {
				exec.Command("systemctl", "--user", "start", "--no-block", "sw-factory.service").Run()
				st += ", dispatcher kicked (a free slot picks it up)"
			}
			s.setStatus(st)
			s.poke()
			writeJSON(w, map[string]string{"Status": st})
		}
	})
	return s.guard(mux)
}

func serve(addr string) {
	s := &server{mrs: map[string]mrInfo{}, atts: map[string]map[string]string{}, reports: map[string]report{}, kick: make(chan struct{}, 1), token: loadToken(), started: time.Now(), addr: addr}
	go editor()
	go s.loop()
	fmt.Printf("factory-dash serving http://%s (run `factory-dash` to open it signed in)\n", addr)
	if err := http.ListenAndServe(addr, s.handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
