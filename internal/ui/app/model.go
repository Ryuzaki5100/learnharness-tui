package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Ryuzaki5100/learnharness-tui/internal/data"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/keys"
	"github.com/Ryuzaki5100/learnharness-tui/internal/ui/styles"
)

// Options configure the root model.
type Options struct {
	DataDir     string
	SyncEnabled bool
}

// Tab is the top-level view.
type Tab int

// Tabs in footer order.
const (
	TabSessions Tab = iota
	TabGraph
	TabStudy
)

func (t Tab) String() string {
	switch t {
	case TabGraph:
		return "graph"
	case TabStudy:
		return "study"
	default:
		return "sessions"
	}
}

type focus int

const (
	focusLeft focus = iota
	focusRight
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeSearch
	modeStudyInput
)

type pendingKeys struct {
	count int
	g     bool
}

// Model is the root Elm model. It owns window size, focus, mode and routing;
// panes never call each other.
type Model struct {
	opts Options
	keys keys.KeyMap
	help help.Model

	width, height int
	ready         bool
	tab           Tab
	focus         focus
	mode          inputMode
	helpVisible   bool

	sessions   sessionsPane
	transcript transcriptPane
	graph      graphPane
	study      studyPane

	sync     data.SyncStatus
	syncBusy bool

	search       textinput.Model
	searchActive bool

	pending pendingKeys
	status  string

	layoutLeftWidth  int
	layoutRightWidth int
	layoutBodyHeight int
}

// New builds the root model.
func New(opts Options) Model {
	km := keys.Default()
	helper := help.New()
	search := textinput.New()
	search.Placeholder = "filter"
	search.Prompt = "/ "
	search.CharLimit = 120

	return Model{
		opts:         opts,
		keys:         km,
		help:         helper,
		search:       search,
		sessions:     newSessionsPane(),
		transcript:   newTranscriptPane(),
		graph:        newGraphPane(),
		study:        newStudyPane(),
		status:       "loading…",
		sync:         data.SyncStatus{Available: data.SyncAvailable(), State: "unknown"},
		helpVisible:  false,
		tab:          TabSessions,
		focus:        focusLeft,
		mode:         modeNormal,
		searchActive: false,
	}
}

// Init loads the session index, graph, study history and sync state.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		loadSessionsCmd(m.opts.DataDir),
		loadGraphCmd(m.opts.DataDir),
		loadStudyIndexCmd(m.opts.DataDir),
	}
	if m.opts.SyncEnabled {
		cmds = append(cmds, syncStatusCmd(m.opts.DataDir))
	}
	return tea.Batch(cmds...)
}

// Update is the single Elm update function.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.layout()
		return m, nil

	case sessionsLoadedMsg:
		if msg.err != nil {
			m.sessions.err = msg.err
			m.status = "failed to load sessions"
			return m, nil
		}
		m.sessions.setSessions(msg.sessions, msg.skipped)
		m.status = fmt.Sprintf("%d sessions", len(msg.sessions))
		if summary, ok := m.sessions.current(); ok {
			return m, loadChatCmd(m.opts.DataDir, summary.ID)
		}
		return m, nil

	case chatLoadedMsg:
		if msg.err != nil {
			m.transcript.err = msg.err
			m.status = "failed to read that session"
			return m, nil
		}
		m.transcript.setEntries(msg.title, msg.entries, m.transcript.innerWidth())
		return m, nil

	case graphLoadedMsg:
		if msg.err != nil {
			m.graph.err = msg.err
			return m, nil
		}
		m.graph.setGraph(msg.graph)
		if len(msg.graph.Warnings) > 0 {
			m.status = fmt.Sprintf("graph loaded with %d warning(s)", len(msg.graph.Warnings))
		}
		return m, nil

	case studyIndexMsg:
		if msg.err != nil {
			m.study.err = msg.err
			return m, nil
		}
		m.study.setDocs(msg.docs)
		return m, nil

	case studyDocMsg:
		if msg.err != nil {
			m.study.err = msg.err
			return m, nil
		}
		m.study.showDoc(msg.doc)
		return m, nil

	case syncStatusMsg:
		m.sync = msg.status
		return m, nil

	case syncDoneMsg:
		m.syncBusy = false
		if msg.err != nil {
			m.status = msg.err.Error()
		} else if strings.TrimSpace(msg.out) == "" {
			m.status = "sync: nothing to do"
		} else {
			m.status = "sync: " + firstLine(msg.out)
		}
		if m.opts.SyncEnabled {
			return m, tea.Batch(syncStatusCmd(m.opts.DataDir), loadSessionsCmd(m.opts.DataDir), loadGraphCmd(m.opts.DataDir))
		}
		return m, nil

	case studyStartedMsg:
		if msg.err != nil {
			m.study.running = false
			m.study.err = msg.err
			m.status = msg.err.Error()
			return m, nil
		}
		m.study.begin(msg.stream, msg.topic, msg.started)
		return m, waitStudyLineCmd(msg.stream)

	case studyLineMsg:
		m.study.appendLine(msg.line)
		if m.study.stream == nil {
			return m, nil
		}
		return m, waitStudyLineCmd(m.study.stream)

	case studyEndedMsg:
		stream := m.study.stream
		m.study.finish(msg.ended)
		if stream != nil {
			stream.Cancel()
		}
		doc := m.study.document()
		if strings.TrimSpace(doc.Body) == "" {
			m.status = "study session produced no output"
			return m, nil
		}
		return m, saveStudyCmd(m.opts.DataDir, doc)

	case studySavedMsg:
		if msg.err != nil {
			m.study.err = msg.err
			m.status = msg.err.Error()
			return m, nil
		}
		m.study.saved = msg.rel
		m.status = "saved " + msg.rel
		cmds := []tea.Cmd{loadStudyIndexCmd(m.opts.DataDir)}
		if m.opts.SyncEnabled && !m.syncBusy {
			m.syncBusy = true
			cmds = append(cmds, syncCmd(m.opts.DataDir, "cycle"))
		}
		return m, tea.Batch(cmds...)

	case interactiveDoneMsg:
		if msg.err != nil {
			m.status = "opencode exited with an error"
		} else {
			m.status = "returned from opencode"
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ctrl+c is always quit; a running study session is cancelled first.
	if msg.String() == "ctrl+c" {
		if m.study.stream != nil {
			m.study.stream.Cancel()
		}
		return m, tea.Quit
	}

	// Help overlay swallows keys except its own toggle and esc.
	if m.helpVisible {
		switch msg.String() {
		case "?", "esc", "q":
			m.helpVisible = false
			m.help.ShowAll = false
			m.layout()
		}
		return m, nil
	}

	// Search input owns keys while active.
	if m.mode == modeSearch {
		switch msg.String() {
		case "esc":
			m.mode = modeNormal
			m.searchActive = false
			m.sessions.setFilter("")
			m.graph.setFilter("")
			m.search.Blur()
			return m, nil
		case "enter":
			m.mode = modeNormal
			m.searchActive = false
			m.search.Blur()
			return m, nil
		default:
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			m.sessions.setFilter(m.search.Value())
			m.graph.setFilter(m.search.Value())
			return m, cmd
		}
	}

	// Study topic input owns keys while active.
	if m.mode == modeStudyInput {
		switch msg.String() {
		case "esc":
			m.mode = modeNormal
			m.study.input.Blur()
			return m, nil
		case "enter":
			topic := strings.TrimSpace(m.study.input.Value())
			if topic == "" {
				m.status = "type a topic first"
				return m, nil
			}
			if m.study.running {
				m.status = "a study session is already running"
				return m, nil
			}
			m.mode = modeNormal
			m.study.input.Blur()
			m.study.reset(topic)
			m.status = "starting opencode…"
			return m, startStudyCmd(topic)
		default:
			var cmd tea.Cmd
			m.study.input, cmd = m.study.input.Update(msg)
			return m, cmd
		}
	}

	// Global keys.
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.study.stream != nil {
			m.status = "cancel the study session first (esc)"
			return m, nil
		}
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.helpVisible = true
		m.help.ShowAll = true
		m.layout()
		return m, nil
	case key.Matches(msg, m.keys.Sessions):
		m.setTab(TabSessions)
		return m, nil
	case key.Matches(msg, m.keys.Graph):
		m.setTab(TabGraph)
		return m, nil
	case key.Matches(msg, m.keys.Study):
		m.setTab(TabStudy)
		return m, nil
	case key.Matches(msg, m.keys.TabRight):
		m.setTab(Tab((int(m.tab) + 1) % 3))
		return m, nil
	case key.Matches(msg, m.keys.TabLeft):
		m.setTab(Tab((int(m.tab) + 2) % 3))
		return m, nil
	case key.Matches(msg, m.keys.Sync):
		return m.startSync()
	case key.Matches(msg, m.keys.Refresh):
		return m, m.refresh()
	case key.Matches(msg, m.keys.Search):
		if m.tab == TabStudy {
			m.status = "search is available in sessions and graph"
			return m, nil
		}
		m.mode = modeSearch
		m.searchActive = true
		m.search.SetValue("")
		m.search.Focus()
		return m, nil
	case key.Matches(msg, m.keys.NewStudy):
		if m.tab != TabStudy {
			return m, nil
		}
		m.mode = modeStudyInput
		m.study.input.Focus()
		return m, nil
	case key.Matches(msg, m.keys.OpenStudy):
		if m.tab != TabStudy {
			return m, nil
		}
		return m, interactiveOpenCodeCmd()
	case key.Matches(msg, m.keys.Left):
		m.focus = focusLeft
		return m, nil
	case key.Matches(msg, m.keys.Right):
		m.focus = focusRight
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		return m.activate()
	case key.Matches(msg, m.keys.Back):
		if m.tab == TabStudy && m.study.running && m.study.stream != nil {
			m.study.stream.Cancel()
			m.status = "cancelling study session…"
			return m, nil
		}
		m.focus = focusLeft
		return m, nil
	}

	// gg sequence and numeric counts.
	if st := msg.String(); st == "g" {
		if m.pending.g {
			m.pending = pendingKeys{}
			return m.moveFocused(-1 << 20)
		}
		m.pending.g = true
		return m, nil
	}
	if n, ok := countDigit(msg.String()); ok {
		if m.pending.count == 0 && n >= 4 || m.pending.count > 0 {
			m.pending.count = m.pending.count*10 + n
			if m.pending.count > 999 {
				m.pending.count = 999
			}
			return m, nil
		}
		return m, nil
	}

	count := m.pending.count
	m.pending.count = 0
	m.pending.g = false

	switch {
	case key.Matches(msg, m.keys.Down):
		return m.moveFocused(positive(count))
	case key.Matches(msg, m.keys.Up):
		return m.moveFocused(-positive(count))
	case key.Matches(msg, m.keys.Bottom):
		return m.moveToEnd()
	case key.Matches(msg, m.keys.HalfDown), key.Matches(msg, m.keys.PageDown):
		return m.scrollFocused(1)
	case key.Matches(msg, m.keys.HalfUp), key.Matches(msg, m.keys.PageUp):
		return m.scrollFocused(-1)
	}
	return m, nil
}

func positive(count int) int {
	if count < 1 {
		return 1
	}
	return count
}

func countDigit(s string) (int, bool) {
	if len(s) != 1 || s[0] < '0' || s[0] > '9' {
		return 0, false
	}
	return int(s[0] - '0'), true
}

func (m *Model) setTab(tab Tab) {
	m.tab = tab
	m.focus = focusLeft
	m.pending = pendingKeys{}
	switch tab {
	case TabGraph:
		if m.graph.graph == nil {
			m.status = "loading graph…"
		}
	case TabStudy:
		if len(m.study.docs) == 0 {
			m.status = "press i to start a study session"
		}
	}
}

func (m Model) refresh() tea.Cmd {
	switch m.tab {
	case TabGraph:
		return loadGraphCmd(m.opts.DataDir)
	case TabStudy:
		return loadStudyIndexCmd(m.opts.DataDir)
	default:
		return tea.Batch(loadSessionsCmd(m.opts.DataDir), syncStatusCmd(m.opts.DataDir))
	}
}

func (m Model) startSync() (tea.Model, tea.Cmd) {
	if !m.opts.SyncEnabled {
		m.status = "sync is disabled (--no-sync)"
		return m, nil
	}
	if m.syncBusy {
		m.status = "a sync is already running"
		return m, nil
	}
	if !data.SyncAvailable() {
		m.status = "lh-sync is not installed"
		return m, nil
	}
	m.syncBusy = true
	m.status = "syncing…"
	return m, syncCmd(m.opts.DataDir, "cycle")
}

func (m Model) activate() (tea.Model, tea.Cmd) {
	switch m.tab {
	case TabSessions:
		summary, ok := m.sessions.current()
		if !ok {
			return m, nil
		}
		return m, loadChatCmd(m.opts.DataDir, summary.ID)
	case TabStudy:
		if m.focus == focusLeft {
			doc, ok := m.study.current()
			if ok {
				return m, loadStudyDocCmd(doc.Path)
			}
		}
	}
	return m, nil
}

func (m Model) moveFocused(delta int) (tea.Model, tea.Cmd) {
	switch m.tab {
	case TabGraph:
		if m.focus == focusLeft {
			m.graph.move(delta)
			m.graph.syncDetail()
			return m, nil
		}
		m.graph.scrollDetail(delta)
		return m, nil
	case TabStudy:
		if m.focus == focusLeft {
			m.study.move(delta)
			return m, nil
		}
		m.study.scrollOutput(delta * 3)
		return m, nil
	default:
		if m.focus == focusLeft {
			before, _ := m.sessions.current()
			m.sessions.move(delta)
			after, ok := m.sessions.current()
			if ok && (before.ID != after.ID) {
				return m, loadChatCmd(m.opts.DataDir, after.ID)
			}
			return m, nil
		}
		if delta > 0 {
			m.transcript.vp.LineDown(delta)
		} else {
			m.transcript.vp.LineUp(-delta)
		}
		return m, nil
	}
}

func (m Model) moveToEnd() (tea.Model, tea.Cmd) {
	switch m.tab {
	case TabGraph:
		if m.focus == focusLeft {
			m.graph.move(1 << 20)
			m.graph.syncDetail()
		} else {
			m.graph.detail.GotoBottom()
		}
	case TabStudy:
		if m.focus == focusLeft {
			m.study.move(1 << 20)
		} else {
			m.study.output.GotoBottom()
		}
	default:
		if m.focus == focusLeft {
			before, _ := m.sessions.current()
			m.sessions.move(1 << 20)
			after, ok := m.sessions.current()
			if ok && before.ID != after.ID {
				return m, loadChatCmd(m.opts.DataDir, after.ID)
			}
		} else {
			m.transcript.vp.GotoBottom()
		}
	}
	return m, nil
}

func (m Model) scrollFocused(direction int) (tea.Model, tea.Cmd) {
	direction *= max(1, m.height/2-2)
	switch m.tab {
	case TabGraph:
		if m.focus == focusLeft {
			m.graph.move(direction)
			m.graph.syncDetail()
		} else if direction > 0 {
			m.graph.detail.LineDown(direction)
		} else {
			m.graph.detail.LineUp(-direction)
		}
	case TabStudy:
		if m.focus == focusLeft {
			m.study.move(direction)
		} else {
			m.study.scrollOutput(direction)
		}
	default:
		if m.focus == focusLeft {
			before, _ := m.sessions.current()
			m.sessions.move(direction)
			after, ok := m.sessions.current()
			if ok && before.ID != after.ID {
				return m, loadChatCmd(m.opts.DataDir, after.ID)
			}
		} else if direction > 0 {
			m.transcript.vp.LineDown(direction)
		} else {
			m.transcript.vp.LineUp(-direction)
		}
	}
	return m, nil
}

func (m *Model) layout() {
	header := 1
	status := 1
	m.help.Width = m.width - 2
	helpHeight := lipgloss.Height(m.help.View(m.keys))
	if helpHeight < 1 {
		helpHeight = 1
	}
	// One row each for the pane titles above the boxes.
	bodyOuter := m.height - header - status - helpHeight - 1
	if bodyOuter < 8 {
		bodyOuter = 8
	}
	leftOuter := int(float64(m.width) * 0.34)
	if leftOuter < 24 {
		leftOuter = 24
	}
	if leftOuter > 48 {
		leftOuter = 48
	}
	if leftOuter > m.width-33 {
		leftOuter = m.width/2 - 1
	}
	if leftOuter < 12 {
		leftOuter = 12
	}
	rightOuter := m.width - leftOuter - 1
	if rightOuter < 20 {
		rightOuter = 20
	}

	innerLeftW := leftOuter - 2
	innerRightW := rightOuter - 2
	innerH := bodyOuter - 2

	m.sessions.setSize(innerLeftW, innerH)
	m.transcript.setSize(innerRightW, innerH)
	m.graph.setSize(innerLeftW, innerH, innerRightW, innerH)
	m.study.setSize(innerLeftW, innerH, innerRightW, innerH)
	m.search.Width = max(12, m.width/2)
	m.layoutLeftWidth = innerLeftW
	m.layoutRightWidth = innerRightW
	m.layoutBodyHeight = innerH
}

// View renders the whole application. It is pure.
func (m Model) View() string {
	if !m.ready {
		return "loading learnharness…"
	}
	if m.width < 60 || m.height < 18 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("terminal too small (need 60x18, have %dx%d)", m.width, m.height))
	}

	header := m.headerView()
	body := m.bodyView()
	status := m.statusView()
	helpView := m.help.View(m.keys)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, status, helpView)
}

func (m Model) headerView() string {
	tabs := []string{
		m.tabLabel(TabSessions, fmt.Sprintf("sessions %d", len(m.sessions.filtered))),
		m.tabLabel(TabGraph, fmt.Sprintf("graph %d", m.graph.count())),
		m.tabLabel(TabStudy, fmt.Sprintf("study %d", len(m.study.docs))),
	}
	left := styles.TabActive().Render("learnharness")
	for _, tab := range tabs {
		left += tab
	}
	if m.mode == modeSearch {
		left = "search: " + m.search.View()
	} else if m.mode == modeStudyInput {
		left = "topic: " + m.study.input.View()
	}
	dirLabel := m.opts.DataDir
	if m.width < 130 {
		dirLabel = filepath.Base(dirLabel)
	}
	right := styles.FaintStyle().Render(dirLabel)
	leftAvail := m.width - lipgloss.Width(right) - 2
	if leftAvail < 36 {
		right = ""
		leftAvail = m.width - 1
	}
	left = styles.Truncate(left, max(10, leftAvail))
	gap := max(1, m.width-lipgloss.Width(left)-lipgloss.Width(right))
	return styles.Header(m.width).Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) tabLabel(tab Tab, label string) string {
	if m.tab == tab {
		return styles.TabActive().Render(label)
	}
	return styles.TabIdle().Render(label)
}

func (m Model) bodyView() string {
	leftTitle := "sessions"
	rightTitle := "transcript"
	var leftContent, rightContent string

	switch m.tab {
	case TabGraph:
		leftTitle, rightTitle = "graph", "proposition"
		leftContent = m.graph.listView()
		rightContent = m.graph.detailView()
	case TabStudy:
		leftTitle, rightTitle = "study history", "study"
		leftContent = m.study.historyView()
		rightContent = m.study.outputView()
	default:
		leftContent = m.sessions.view()
		rightContent = m.transcript.view()
		if m.transcript.title != "" {
			rightTitle = styles.Truncate(m.transcript.title, max(8, m.layoutRightWidth-6))
		}
	}

	leftFocused := m.focus == focusLeft
	rightFocused := m.focus == focusRight
	leftBox, leftTitleStyle := styles.Box(m.layoutLeftWidth, m.layoutBodyHeight, leftFocused, leftTitle)
	rightBox, rightTitleStyle := styles.Box(m.layoutRightWidth, m.layoutBodyHeight, rightFocused, rightTitle)

	leftPane := lipgloss.JoinVertical(lipgloss.Left,
		leftTitleStyle.Render(" "+leftTitle),
		leftBox.Render(leftContent),
	)
	rightPane := lipgloss.JoinVertical(lipgloss.Left,
		rightTitleStyle.Render(" "+rightTitle),
		rightBox.Render(rightContent),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
}

func (m Model) statusView() string {
	mode := "NORMAL"
	switch m.mode {
	case modeSearch:
		mode = "SEARCH"
	case modeStudyInput:
		mode = "INSERT"
	}
	left := fmt.Sprintf("%s  %s", mode, m.sync.String())
	if m.syncBusy {
		left += " · busy"
	}
	if m.status != "" {
		left += "  ·  " + m.status
	}
	return styles.Status(m.width).Render(styles.Truncate(" "+left, m.width))
}

func firstLine(text string) string {
	if index := strings.IndexByte(text, '\n'); index >= 0 {
		return text[:index]
	}
	return text
}
