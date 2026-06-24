package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iliutaadrian/wc26/internal/api"
)

// --- data helpers ---

// filteredMatches returns matches matching the active filter, sorted by date.
func (m Model) filteredMatches() []api.Match {
	out := make([]api.Match, 0, len(m.matches))
	now := time.Now()
	for _, mt := range m.matches {
		switch m.matchFilter {
		case filterLive:
			if !mt.IsLive() {
				continue
			}
		case filterToday:
			if !sameDay(mt.UTCDate.Local(), now) {
				continue
			}
		case filterUpcoming:
			if mt.IsFinished() || mt.IsLive() {
				continue
			}
		case filterFinished:
			if !mt.IsFinished() {
				continue
			}
		}
		out = append(out, mt)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UTCDate.Before(out[j].UTCDate) })
	return out
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// filterCount counts how many matches match a given filter (for menu badges).
func (m Model) countFor(f matchFilter) int {
	saved := m.matchFilter
	m.matchFilter = f
	n := len(m.filteredMatches())
	m.matchFilter = saved
	return n
}

// totalStandings keeps only the cumulative group tables.
func totalStandings(in []api.Standing) []api.Standing {
	out := make([]api.Standing, 0, len(in))
	for _, s := range in {
		if s.Type == "" || s.Type == "TOTAL" {
			out = append(out, s)
		}
	}
	return out
}

// --- view ---

func (m Model) View() string {
	if !m.ready {
		return "\n  Starting World Cup 2026…\n"
	}
	if m.showHelp {
		return m.helpView()
	}

	header := m.headerView()
	footer := m.footerView()
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(footer)
	if bodyHeight < 3 {
		bodyHeight = 3
	}

	var body string
	switch m.active {
	case tabMatches:
		body = m.matchesView(bodyHeight)
	case tabStandings:
		body = m.standingsView(bodyHeight)
	case tabScorers:
		body = m.scorersView(bodyHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) headerView() string {
	title := titleStyle.Render("⚽ World Cup 2026")

	tabs := make([]string, 0, int(tabCount))
	for t := tabMatches; t < tabCount; t++ {
		label := fmt.Sprintf("%d %s", int(t)+1, t.String())
		if t == m.active {
			tabs = append(tabs, tabActive.Render(label))
		} else {
			tabs = append(tabs, tabInactive.Render(label))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Status on the right.
	status := m.status
	if m.loading {
		status = m.spinner.View() + " " + status
	}
	st := statusOK
	if m.statusIsErr {
		st = statusErr
	}
	statusR := st.Render(status)

	left := lipgloss.JoinHorizontal(lipgloss.Top, title, "  ", tabBar)
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(statusR)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + statusR
}

func (m Model) footerView() string {
	keys := []string{
		hk("tab") + " switch",
		hk("j/k") + " move",
		hk("r") + " refresh",
	}
	if m.active == tabMatches {
		keys = append(keys, hk("f")+" filter:"+m.matchFilter.String())
	}
	keys = append(keys, hk("?")+" help", hk("q")+" quit")
	return footerStyle.Render(strings.Join(keys, "   "))
}

func hk(s string) string { return headerKey.Render(s) }

// --- Matches tab ---

func (m Model) matchesView(height int) string {
	matches := m.filteredMatches()

	// Left pane: filter menu with per-filter counts.
	leftW := 18
	var fl strings.Builder
	fl.WriteString(paneTitle.Render("Filter") + "\n\n")
	for f := filterAll; f < filterCount; f++ {
		marker := "  "
		style := textStyle
		if f == m.matchFilter {
			marker = "▸ "
			style = winStyle
		}
		label := fmt.Sprintf("%-9s%s", f.String(), dimStyle.Render(fmt.Sprintf("%d", m.countFor(f))))
		fl.WriteString(style.Render(marker) + style.Render(label) + "\n")
	}
	fl.WriteString("\n" + dimStyle.Render("f next · F prev"))
	left := paneBorder.Width(leftW).Height(height - 2).Render(fl.String())

	// Right pane: match rows.
	rightW := m.width - lipgloss.Width(left) - 2
	if rightW < 20 {
		rightW = 20
	}
	rows := make([]string, 0, len(matches))
	if len(matches) == 0 {
		rows = append(rows, m.emptyMatches())
	}
	for i, mt := range matches {
		rows = append(rows, m.matchRow(mt, i == m.matchCursor, rightW-2))
	}
	list := renderWindow(rows, m.matchCursor, height-4)
	title := fmt.Sprintf("Matches — %s (%d)", m.matchFilter.String(), len(matches))
	right := paneBorderActive.Width(rightW).Height(height - 2).Render(
		paneTitle.Render(title) + "\n" + list)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

// emptyMatches returns a helpful message when the active filter has no matches.
func (m Model) emptyMatches() string {
	var msg string
	switch m.matchFilter {
	case filterLive:
		msg = fmt.Sprintf("No matches are live right now.\n  %d already played · %d still to come.\n  Press f for Today / Upcoming / All.",
			m.countFor(filterFinished), m.countFor(filterUpcoming))
	case filterToday:
		msg = "No matches scheduled today.\n  Press f for Upcoming or All."
	case filterUpcoming:
		msg = "No upcoming matches — the tournament may be over."
	case filterFinished:
		msg = "No matches have finished yet."
	default:
		if len(m.matches) == 0 {
			msg = "No matches loaded yet. Press r to refresh."
		} else {
			msg = "No matches for this filter."
		}
	}
	return "\n  " + dimStyle.Render(msg)
}

func (m Model) matchRow(mt api.Match, selected bool, width int) string {
	local := mt.UTCDate.Local()
	var when string
	if sameDay(local, time.Now()) {
		when = "Today " + local.Format("15:04")
	} else {
		when = local.Format("Jan 02 15:04")
	}
	when = fmt.Sprintf("%-12s", when)

	home := fmt.Sprintf("%-4s", mt.HomeTeam.DisplayName())
	away := fmt.Sprintf("%-4s", mt.AwayTeam.DisplayName())

	var score string
	if (mt.IsFinished() || mt.IsLive()) && mt.Score.FullTime.Home != nil && mt.Score.FullTime.Away != nil {
		score = fmt.Sprintf("%d - %d", *mt.Score.FullTime.Home, *mt.Score.FullTime.Away)
	} else {
		score = " vs  "
	}

	homeS, awayS := home, away
	switch mt.Score.Winner {
	case "HOME_TEAM":
		homeS = winStyle.Render(home)
	case "AWAY_TEAM":
		awayS = winStyle.Render(away)
	}

	var badge string
	switch {
	case mt.IsLive():
		badge = liveStyle.Render("LIVE")
	case mt.IsFinished():
		badge = dimStyle.Render("FT")
	}

	line := fmt.Sprintf("%s %s  %s  %s  %s", dimStyle.Render(when), homeS, score, awayS, badge)

	cursor := "  "
	if selected {
		cursor = winStyle.Render("▸ ")
	}
	row := cursor + line
	if selected {
		return selectedRow.Width(width).Render(stripToWidth(row, width))
	}
	return row
}

// --- Standings tab ---

func (m Model) standingsView(height int) string {
	leftW := 16
	var gl strings.Builder
	gl.WriteString(paneTitle.Render("Groups") + "\n\n")
	if len(m.standings) == 0 {
		gl.WriteString(dimStyle.Render("  none yet"))
	}
	for i, s := range m.standings {
		marker := "  "
		style := textStyle
		if i == m.groupCursor {
			marker = "▸ "
			style = winStyle
		}
		gl.WriteString(style.Render(marker+groupLabel(s)) + "\n")
	}
	left := paneBorder.Width(leftW).Height(height - 2).Render(gl.String())

	rightW := m.width - lipgloss.Width(left) - 2
	if rightW < 30 {
		rightW = 30
	}

	var table string
	if m.groupCursor >= 0 && m.groupCursor < len(m.standings) {
		table = m.standingTable(m.standings[m.groupCursor])
	} else {
		table = dimStyle.Render("Standings appear once the group stage begins.")
	}
	right := paneBorderActive.Width(rightW).Height(height - 2).Render(table)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func groupLabel(s api.Standing) string {
	if s.Group != "" {
		return strings.Title(strings.ToLower(strings.ReplaceAll(s.Group, "_", " ")))
	}
	if s.Stage != "" {
		return strings.Title(strings.ToLower(strings.ReplaceAll(s.Stage, "_", " ")))
	}
	return "Table"
}

func (m Model) standingTable(s api.Standing) string {
	var b strings.Builder
	b.WriteString(paneTitle.Render(groupLabel(s)) + "\n\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%-3s %-5s %2s %2s %2s %2s %3s %3s %3s %3s",
		"#", "Team", "P", "W", "D", "L", "GF", "GA", "GD", "Pts")) + "\n")
	for _, r := range s.Table {
		line := fmt.Sprintf("%-3d %-5s %2d %2d %2d %2d %3d %3d %+3d %3s",
			r.Position, r.Team.DisplayName(), r.PlayedGames, r.Won, r.Draw, r.Lost,
			r.GoalsFor, r.GoalsAgainst, r.GoalDifference, winStyle.Render(fmt.Sprintf("%d", r.Points)))
		// Highlight the top two (typical qualification spots).
		if r.Position <= 2 {
			line = textStyle.Render(line)
		} else {
			line = dimStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// --- Scorers tab ---

func (m Model) scorersView(height int) string {
	w := m.width - 2
	var b strings.Builder
	b.WriteString(paneTitle.Render("Top Scorers") + "\n\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%-3s %-22s %-5s %3s %3s", "#", "Player", "Team", "G", "A")) + "\n")

	rows := make([]string, 0, len(m.scorers))
	for i, s := range m.scorers {
		assists := 0
		if s.Assists != nil {
			assists = *s.Assists
		}
		name := truncate(s.Player.Name, 22)
		line := fmt.Sprintf("%-3d %-22s %-5s %3d %3d", i+1, name, s.Team.DisplayName(), s.Goals, assists)
		if i == m.scorerCursor {
			line = selectedRow.Width(w - 2).Render(line)
		} else {
			line = textStyle.Render(line)
		}
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = append(rows, dimStyle.Render("No goals scored yet."))
	}
	// Reserve lines for: title, blank, header, and the "↓ more" indicator.
	b.WriteString(renderWindow(rows, m.scorerCursor, height-6))
	return paneBorderActive.Width(w).Height(height - 2).Render(b.String())
}

// --- helpers ---

// truncate shortens s to at most n runes (not bytes), appending an ellipsis.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// renderWindow returns the visible slice of lines keeping cursor in view.
func renderWindow(lines []string, cursor, height int) string {
	if height < 1 {
		height = 1
	}
	if len(lines) <= height {
		return strings.Join(lines, "\n")
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	if start+height > len(lines) {
		start = len(lines) - height
	}
	visible := lines[start : start+height]
	out := strings.Join(visible, "\n")
	if start+height < len(lines) {
		out += "\n" + dimStyle.Render("  ↓ more")
	}
	return out
}

func stripToWidth(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return s
}

func (m Model) helpView() string {
	lines := []string{
		titleStyle.Render("⚽ World Cup 2026 — Help"),
		"",
		hk("1 / 2 / 3") + "    jump to Matches / Standings / Scorers",
		hk("tab / h l") + "    switch tabs",
		hk("j / k") + "        move selection (also ↑ ↓)",
		hk("g / G") + "        jump to top / bottom",
		hk("f") + "            cycle match filter (All/Live/Today/Upcoming/Finished)",
		hk("r") + "            refresh current tab from the API",
		hk("?") + "            toggle this help",
		hk("q") + "            quit",
		"",
		dimStyle.Render("Data: football-data.org (FIFA World Cup). Live tabs auto-refresh every 30s."),
		dimStyle.Render("Press any key to return."),
	}
	box := lipgloss.NewStyle().Padding(1, 2).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
