package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iliutaadrian/wc26-cli/internal/api"
)

// The Standings tab's Brackets view renders the knockout stage as a left-to-right single-
// elimination bracket: each round is a column, parents are vertically
// centred between the two matches that feed them, and connector lines tie
// the tree together. Winners are highlighted and the champion (once the
// final is decided) gets a banner. It is derived entirely from the matches
// feed — no extra API call.

// --- stage taxonomy ---

// knockoutRank orders the bracket columns left → right. 0 means "not a
// bracket round" (group stage, third-place play-off, or unknown).
func knockoutRank(stage string) int {
	switch stage {
	case "LAST_32", "ROUND_OF_32":
		return 1
	case "LAST_16", "ROUND_OF_16":
		return 2
	case "QUARTER_FINALS", "QUARTER_FINAL":
		return 3
	case "SEMI_FINALS", "SEMI_FINAL":
		return 4
	case "FINAL":
		return 6
	}
	return 0
}

// isThirdPlace reports whether the stage is the third-place play-off, which
// sits outside the main elimination tree.
func isThirdPlace(stage string) bool {
	switch stage {
	case "THIRD_PLACE", "3RD_PLACE_FINAL", "PLAY_OFF_FOR_THIRD_PLACE":
		return true
	}
	return false
}

// stageLabel returns a human-friendly column heading for a knockout stage.
func stageLabel(stage string) string {
	switch stage {
	case "LAST_32", "ROUND_OF_32":
		return "Round of 32"
	case "LAST_16", "ROUND_OF_16":
		return "Round of 16"
	case "QUARTER_FINALS", "QUARTER_FINAL":
		return "Quarter-finals"
	case "SEMI_FINALS", "SEMI_FINAL":
		return "Semi-finals"
	case "FINAL":
		return "Final"
	}
	if isThirdPlace(stage) {
		return "Third place"
	}
	return strings.Title(strings.ToLower(strings.ReplaceAll(stage, "_", " ")))
}

// bracketRound is one column of the bracket.
type bracketRound struct {
	label   string
	matches []api.Match
}

// knockoutRounds groups the loaded matches into ordered bracket columns,
// excluding the third-place play-off.
func (m Model) knockoutRounds() []bracketRound {
	byRank := map[int][]api.Match{}
	for _, mt := range m.matches {
		if isThirdPlace(mt.Stage) {
			continue
		}
		r := knockoutRank(mt.Stage)
		if r == 0 {
			continue
		}
		byRank[r] = append(byRank[r], mt)
	}

	ranks := make([]int, 0, len(byRank))
	for r := range byRank {
		ranks = append(ranks, r)
	}
	sort.Ints(ranks)

	out := make([]bracketRound, 0, len(ranks))
	for _, r := range ranks {
		ms := byRank[r]
		sort.Slice(ms, func(i, j int) bool {
			if !ms[i].UTCDate.Equal(ms[j].UTCDate) {
				return ms[i].UTCDate.Before(ms[j].UTCDate)
			}
			return ms[i].ID < ms[j].ID
		})
		out = append(out, bracketRound{label: stageLabel(ms[0].Stage), matches: ms})
	}
	return out
}

// thirdPlaceMatch returns the third-place play-off, if present.
func (m Model) thirdPlaceMatch() *api.Match {
	for i := range m.matches {
		if isThirdPlace(m.matches[i].Stage) {
			return &m.matches[i]
		}
	}
	return nil
}

// champion returns the winning team's name once the final has finished.
func (m Model) champion() string {
	for _, mt := range m.matches {
		if mt.Stage != "FINAL" || !mt.IsFinished() {
			continue
		}
		switch mt.Score.Winner {
		case "HOME_TEAM":
			return fullName(mt.HomeTeam)
		case "AWAY_TEAM":
			return fullName(mt.AwayTeam)
		}
	}
	return ""
}

func fullName(t api.Team) string {
	if t.Name != "" {
		return t.Name
	}
	return t.DisplayName()
}

// --- bracket geometry ---

const (
	cellH    = 3 // text lines per match cell
	cellGap  = 1 // blank lines between sibling cells in the first column
	colW     = 15
	connW    = 3 // width of the connector gutter between columns
	colUnit  = colW + connW
	slotRows = cellH + cellGap

	// standingsLeftW is the width of the Standings tab's left menu pane; the
	// bracket renders in the remaining space to its right.
	standingsLeftW = 16
)

// bracketAvailWidth is the horizontal space the bracket has inside the
// Standings tab's right pane.
func (m Model) bracketAvailWidth() int {
	w := m.width - standingsLeftW - 4
	if w < 20 {
		w = 20
	}
	return w
}

// bracketLayout decides which rounds fit in the current width (always
// keeping the latest rounds — the climax), computes the top row of every
// cell so parents sit centred between their feeders, and reports the total
// grid height.
func (m Model) bracketLayout() (shown []bracketRound, tops [][]int, totalRows int) {
	rounds := m.knockoutRounds()
	if len(rounds) == 0 {
		return nil, nil, 0
	}

	avail := m.bracketAvailWidth()
	fit := (avail + connW) / colUnit
	if fit < 1 {
		fit = 1
	}
	shown = rounds
	if len(rounds) > fit {
		shown = rounds[len(rounds)-fit:]
	}

	tops = make([][]int, len(shown))
	tops[0] = make([]int, len(shown[0].matches))
	for j := range tops[0] {
		tops[0][j] = j * slotRows
	}
	for r := 1; r < len(shown); r++ {
		prev := tops[r-1]
		n := len(shown[r].matches)
		tops[r] = make([]int, n)
		for j := 0; j < n; j++ {
			hi, lo := 2*j, 2*j+1
			switch {
			case lo < len(prev):
				tops[r][j] = (prev[hi] + prev[lo]) / 2
			case hi < len(prev):
				tops[r][j] = prev[hi]
			default:
				tops[r][j] = j * slotRows
			}
		}
	}

	for r := range shown {
		for j := range shown[r].matches {
			if end := tops[r][j] + cellH; end > totalRows {
				totalRows = end
			}
		}
	}
	return shown, tops, totalRows
}

// bracketViewport is the number of grid rows visible at once.
func (m Model) bracketViewport() int {
	// The bracket sits inside the Standings right pane: subtract the app
	// header/footer, the pane border, and the bracket's own title + column
	// header + footnote lines.
	v := (m.height - 2) - 7
	if v < 1 {
		v = 1
	}
	return v
}

func (m Model) maxBracketScroll() int {
	_, _, total := m.bracketLayout()
	if max := total - m.bracketViewport(); max > 0 {
		return max
	}
	return 0
}

// --- cell rendering ---

// bracketCell renders one match as exactly cellH lines, each colW wide.
func bracketCell(mt api.Match, w int) []string {
	played := (mt.IsFinished() || mt.IsLive()) &&
		mt.Score.FullTime.Home != nil && mt.Score.FullTime.Away != nil
	var hs, as string
	if played {
		hs = fmt.Sprintf("%d", *mt.Score.FullTime.Home)
		as = fmt.Sprintf("%d", *mt.Score.FullTime.Away)
	}

	var meta string
	metaStyle := dimStyle
	switch {
	case mt.IsLive():
		meta, metaStyle = "● LIVE", liveDot
	case mt.IsFinished():
		meta = "FT"
		if mt.Score.Duration == "PENALTY_SHOOTOUT" {
			meta += " (p)"
		}
	case !mt.UTCDate.IsZero():
		meta = mt.UTCDate.Local().Format("Jan 02")
	}

	homeWin := mt.Score.Winner == "HOME_TEAM"
	awayWin := mt.Score.Winner == "AWAY_TEAM"

	return []string{
		metaStyle.Render(padRight(meta, w)),
		teamLine(mt.HomeTeam.DisplayName(), hs, homeWin, played, w),
		teamLine(mt.AwayTeam.DisplayName(), as, awayWin, played, w),
	}
}

func teamLine(name, score string, win, played bool, w int) string {
	raw := " " + fmt.Sprintf("%-4.4s", name)
	gap := w - lipgloss.Width(raw) - lipgloss.Width(score) - 1
	if gap < 1 {
		gap = 1
	}
	raw = padRight(raw+strings.Repeat(" ", gap)+score+" ", w)

	switch {
	case win:
		return winStyle.Render(raw)
	case played:
		return loseStyle.Render(raw)
	default:
		return textStyle.Render(raw)
	}
}

// padRight pads (or truncates) s to exactly w display columns.
func padRight(s string, w int) string {
	if lipgloss.Width(s) > w {
		return truncate(s, w)
	}
	for lipgloss.Width(s) < w {
		s += " "
	}
	return s
}

// centerText centres a plain string within width w.
func centerText(s string, w int) string {
	s = truncate(s, w)
	total := w - lipgloss.Width(s)
	if total <= 0 {
		return s
	}
	left := total / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", total-left)
}

// --- view ---

// bracketBody renders the knockout bracket as content for the Standings
// tab's right pane (the pane supplies the surrounding box).
func (m Model) bracketBody(height int) string {
	shown, tops, totalRows := m.bracketLayout()
	if len(shown) == 0 {
		msg := "The knockout bracket appears once the group stage ends.\n\n" +
			"  48 teams · 12 groups → Round of 32 → Round of 16\n" +
			"  → Quarter-finals → Semi-finals → Final."
		return paneTitle.Render("Bracket") + "\n\n" + dimStyle.Render(msg)
	}

	rounds := m.knockoutRounds()
	hidden := len(rounds) - len(shown)

	// Pre-render every cell and the connector gutters.
	cells := make([][][]string, len(shown))
	for r := range shown {
		cells[r] = make([][]string, len(shown[r].matches))
		for j, mt := range shown[r].matches {
			cells[r][j] = bracketCell(mt, colW)
		}
	}
	conns := buildConnectors(shown, tops, totalRows)

	// Compose the full grid row by row.
	grid := make([]string, totalRows)
	for row := 0; row < totalRows; row++ {
		var b strings.Builder
		for r := range shown {
			seg := strings.Repeat(" ", colW)
			for j := range shown[r].matches {
				if t := tops[r][j]; row >= t && row < t+cellH {
					seg = cells[r][j][row-t]
					break
				}
			}
			b.WriteString(seg)
			if r < len(shown)-1 {
				b.WriteString(bracketConn.Render(string(conns[r][row])))
			}
		}
		grid[row] = b.String()
	}

	// Vertical window.
	viewport := m.bracketViewport()
	scroll := clamp(m.bracketScroll, 0, max0(totalRows-viewport))
	end := scroll + viewport
	if end > totalRows {
		end = totalRows
	}
	visible := grid[scroll:end]

	// Column headers (fixed, above the scroll window).
	var hdr strings.Builder
	for r := range shown {
		hdr.WriteString(paneTitle.Render(centerText(shown[r].label, colW)))
		if r < len(shown)-1 {
			hdr.WriteString(strings.Repeat(" ", connW))
		}
	}

	// Title line, with champion and any hidden-rounds note.
	title := paneTitle.Render("Bracket")
	if champ := m.champion(); champ != "" {
		title += "   " + champStyle.Render("🏆 "+champ)
	}
	if hidden > 0 {
		title += "   " + dimStyle.Render(fmt.Sprintf("← %d earlier round(s) — widen terminal", hidden))
	}

	parts := []string{title, hdr.String(), strings.Join(visible, "\n")}

	var footnotes []string
	if scroll > 0 {
		footnotes = append(footnotes, "↑ more")
	}
	if end < totalRows {
		footnotes = append(footnotes, "↓ more (j/k)")
	}
	if tp := m.thirdPlaceMatch(); tp != nil {
		footnotes = append(footnotes, "3rd place: "+thirdPlaceLine(*tp))
	}
	if len(footnotes) > 0 {
		parts = append(parts, dimStyle.Render("  "+strings.Join(footnotes, "   ·   ")))
	}

	return strings.Join(parts, "\n")
}

// thirdPlaceLine summarises the third-place play-off on a single line.
func thirdPlaceLine(mt api.Match) string {
	home, away := mt.HomeTeam.DisplayName(), mt.AwayTeam.DisplayName()
	if (mt.IsFinished() || mt.IsLive()) && mt.Score.FullTime.Home != nil && mt.Score.FullTime.Away != nil {
		return fmt.Sprintf("%s %d-%d %s", home, *mt.Score.FullTime.Home, *mt.Score.FullTime.Away, away)
	}
	return fmt.Sprintf("%s vs %s", home, away)
}

// buildConnectors draws the bracket lines in each gutter between columns.
func buildConnectors(shown []bracketRound, tops [][]int, totalRows int) [][][]rune {
	conns := make([][][]rune, max0(len(shown)-1))
	for r := 0; r < len(shown)-1; r++ {
		g := make([][]rune, totalRows)
		for i := range g {
			g[i] = []rune(strings.Repeat(" ", connW))
		}
		set := func(row, col int, ch rune) {
			if row >= 0 && row < totalRows && col >= 0 && col < connW {
				g[row][col] = ch
			}
		}
		for j := range shown[r+1].matches {
			hi, lo := 2*j, 2*j+1
			if hi >= len(shown[r].matches) {
				continue
			}
			pCenter := tops[r+1][j] + cellH/2
			hCenter := tops[r][hi] + cellH/2
			if lo < len(shown[r].matches) {
				lCenter := tops[r][lo] + cellH/2
				for row := hCenter; row <= lCenter; row++ {
					set(row, 1, '│')
				}
				set(hCenter, 0, '─')
				set(hCenter, 1, '┐')
				set(lCenter, 0, '─')
				set(lCenter, 1, '┘')
				set(pCenter, 1, '├')
				set(pCenter, 2, '─')
			} else {
				// Lone feeder (incomplete data): straight line through.
				set(hCenter, 0, '─')
				set(hCenter, 1, '─')
				set(hCenter, 2, '─')
			}
		}
		conns[r] = g
	}
	return conns
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
