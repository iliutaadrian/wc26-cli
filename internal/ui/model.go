package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/iliutaadrian/wc26-cli/internal/api"
)

type tab int

const (
	tabMatches tab = iota
	tabStandings
	tabScorers
	tabCount
)

func (t tab) String() string {
	switch t {
	case tabMatches:
		return "Matches"
	case tabStandings:
		return "Standings"
	case tabScorers:
		return "Scorers"
	}
	return ""
}

// standingsMode selects what the Standings tab shows: the knockout bracket
// or the group-stage tables. Toggled in the left panel (default: bracket).
type standingsMode int

const (
	modeBracket standingsMode = iota
	modeGroups
	modeCount
)

func (s standingsMode) String() string {
	switch s {
	case modeBracket:
		return "Brackets"
	case modeGroups:
		return "Group Stage"
	}
	return ""
}

type matchFilter int

const (
	filterAll matchFilter = iota
	filterLive
	filterToday
	filterUpcoming
	filterFinished
	filterCount
)

func (f matchFilter) String() string {
	switch f {
	case filterAll:
		return "All"
	case filterLive:
		return "Live"
	case filterToday:
		return "Today"
	case filterUpcoming:
		return "Upcoming"
	case filterFinished:
		return "Finished"
	}
	return ""
}

// --- messages ---

type matchesMsg struct {
	resp *api.MatchesResponse
	err  error
}
type standingsMsg struct {
	resp *api.StandingsResponse
	err  error
}
type scorersMsg struct {
	resp *api.ScorersResponse
	err  error
}
type tickMsg time.Time

const liveRefresh = 30 * time.Second

func tickCmd() tea.Cmd {
	return tea.Tick(liveRefresh, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// --- model ---

type Model struct {
	client        *api.Client
	width, height int
	ready         bool

	active   tab
	spinner  spinner.Model
	loading  bool
	status   string
	statusIsErr bool
	showHelp bool

	matches         []api.Match
	matchFilter     matchFilter
	matchCursor     int
	matchesLoaded   bool
	filterPicked    bool

	standings        []api.Standing
	groupCursor      int
	standingsLoaded  bool
	stdMode          standingsMode

	scorers        []api.Scorer
	scorerCursor   int
	scorersLoaded  bool

	bracketScroll int

	lastUpdated time.Time
}

func New(client *api.Client) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		client:  client,
		spinner: sp,
		active:  tabMatches,
		loading: true,
		status:  "loading…",
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadMatches(false), tickCmd())
}

func (m Model) loadMatches(fresh bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Matches(fresh)
		return matchesMsg{resp, err}
	}
}
func (m Model) loadStandings(fresh bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Standings(fresh)
		return standingsMsg{resp, err}
	}
}
func (m Model) loadScorers(fresh bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := m.client.Scorers(fresh)
		return scorersMsg{resp, err}
	}
}

// loadForTab lazily loads the active tab's data if not yet loaded.
func (m *Model) loadForTab() tea.Cmd {
	switch m.active {
	case tabStandings:
		// The Standings tab shows either the bracket (derived from the
		// matches feed) or the group tables, so both feeds may be needed.
		var cmds []tea.Cmd
		if !m.standingsLoaded {
			cmds = append(cmds, m.loadStandings(false))
		}
		if !m.matchesLoaded {
			cmds = append(cmds, m.loadMatches(false))
		}
		if len(cmds) > 0 {
			m.loading = true
			return tea.Batch(cmds...)
		}
	case tabScorers:
		if !m.scorersLoaded {
			m.loading = true
			return m.loadScorers(false)
		}
	case tabMatches:
		if !m.matchesLoaded {
			m.loading = true
			return m.loadMatches(false)
		}
	}
	return nil
}

func (m Model) hasLive() bool {
	for _, mt := range m.matches {
		if mt.IsLive() {
			return true
		}
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tickMsg:
		// Auto-refresh while matches are live.
		var cmd tea.Cmd
		if m.hasLive() {
			cmd = m.loadMatches(true)
		}
		return m, tea.Batch(cmd, tickCmd())

	case matchesMsg:
		m.loading = false
		m.matchesLoaded = true
		if msg.err != nil {
			m.setErr(msg.err.Error())
			return m, nil
		}
		m.matches = msg.resp.Matches
		m.lastUpdated = time.Now()
		// On first load, default to today's matches (which include any live
		// game); fall back to All on rest days so the list is never empty.
		if !m.filterPicked {
			m.filterPicked = true
			if m.countFor(filterToday) > 0 {
				m.matchFilter = filterToday
			} else {
				m.matchFilter = filterAll
			}
		}
		m.clampCursors()
		m.setOK()
		return m, nil

	case standingsMsg:
		m.loading = false
		m.standingsLoaded = true
		if msg.err != nil {
			m.setErr(msg.err.Error())
			return m, nil
		}
		m.standings = totalStandings(msg.resp.Standings)
		m.lastUpdated = time.Now()
		m.clampCursors()
		m.setOK()
		return m, nil

	case scorersMsg:
		m.loading = false
		m.scorersLoaded = true
		if msg.err != nil {
			m.setErr(msg.err.Error())
			return m, nil
		}
		m.scorers = msg.resp.Scorers
		m.lastUpdated = time.Now()
		m.clampCursors()
		m.setOK()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) setErr(s string) {
	m.status = s
	m.statusIsErr = true
}
func (m *Model) setOK() {
	m.status = "updated " + m.lastUpdated.Format("15:04:05")
	m.statusIsErr = false
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "esc":
		m.showHelp = false
		return m, nil

	case "tab", "l", "right":
		m.active = (m.active + 1) % tabCount
		return m, m.loadForTab()

	case "shift+tab", "h", "left":
		m.active = (m.active - 1 + tabCount) % tabCount
		return m, m.loadForTab()

	case "1":
		m.active = tabMatches
		return m, m.loadForTab()
	case "2":
		m.active = tabStandings
		return m, m.loadForTab()
	case "3":
		m.active = tabScorers
		return m, m.loadForTab()

	case "r":
		m.loading = true
		m.status = "refreshing…"
		m.statusIsErr = false
		switch m.active {
		case tabMatches:
			return m, tea.Batch(m.spinner.Tick, m.loadMatches(true))
		case tabStandings:
			if m.stdMode == modeBracket {
				return m, tea.Batch(m.spinner.Tick, m.loadMatches(true))
			}
			return m, tea.Batch(m.spinner.Tick, m.loadStandings(true))
		case tabScorers:
			return m, tea.Batch(m.spinner.Tick, m.loadScorers(true))
		}

	case "f":
		switch m.active {
		case tabMatches:
			m.matchFilter = (m.matchFilter + 1) % filterCount
			m.matchCursor = 0
		case tabStandings:
			m.stdMode = (m.stdMode + 1) % modeCount
		}
		return m, nil

	case "F", "shift+f":
		switch m.active {
		case tabMatches:
			m.matchFilter = (m.matchFilter - 1 + filterCount) % filterCount
			m.matchCursor = 0
		case tabStandings:
			m.stdMode = (m.stdMode - 1 + modeCount) % modeCount
		}
		return m, nil

	case "j", "down":
		m.moveCursor(1)
		return m, nil
	case "k", "up":
		m.moveCursor(-1)
		return m, nil

	case "g", "home":
		m.setCursor(0)
		return m, nil
	case "G", "end":
		m.setCursor(1 << 30)
		return m, nil
	}
	return m, nil
}

func (m *Model) moveCursor(delta int) {
	switch m.active {
	case tabMatches:
		m.matchCursor += delta
	case tabStandings:
		if m.stdMode == modeBracket {
			m.bracketScroll += delta
		} else {
			m.groupCursor += delta
		}
	case tabScorers:
		m.scorerCursor += delta
	}
	m.clampCursors()
}

func (m *Model) setCursor(v int) {
	switch m.active {
	case tabMatches:
		m.matchCursor = v
	case tabStandings:
		if m.stdMode == modeBracket {
			m.bracketScroll = v
		} else {
			m.groupCursor = v
		}
	case tabScorers:
		m.scorerCursor = v
	}
	m.clampCursors()
}

func (m *Model) clampCursors() {
	m.matchCursor = clamp(m.matchCursor, 0, len(m.filteredMatches())-1)
	m.groupCursor = clamp(m.groupCursor, 0, len(m.standings)-1)
	m.scorerCursor = clamp(m.scorerCursor, 0, len(m.scorers)-1)
	m.bracketScroll = clamp(m.bracketScroll, 0, m.maxBracketScroll())
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
