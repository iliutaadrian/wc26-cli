package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iliutaadrian/wc26-cli/internal/api"
)

func ip(v int) *int { return &v }

func mockModel(t *testing.T) Model {
	t.Helper()
	m := New(nil)
	// Give it a window so View() renders the full layout.
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = mm.(Model)

	now := time.Now()
	matches := &api.MatchesResponse{Matches: []api.Match{
		{ID: 1, UTCDate: now.Add(-2 * time.Hour), Status: "FINISHED", HomeTeam: api.Team{TLA: "MEX"}, AwayTeam: api.Team{TLA: "CAN"},
			Score: api.Score{Winner: "HOME_TEAM", FullTime: api.ScoreLine{Home: ip(2), Away: ip(1)}}},
		{ID: 2, UTCDate: now, Status: "IN_PLAY", HomeTeam: api.Team{TLA: "USA"}, AwayTeam: api.Team{TLA: "GER"},
			Score: api.Score{FullTime: api.ScoreLine{Home: ip(1), Away: ip(1)}}},
		{ID: 3, UTCDate: now.Add(3 * time.Hour), Status: "TIMED", HomeTeam: api.Team{TLA: "BRA"}, AwayTeam: api.Team{TLA: "ARG"}},
	}}
	mm, _ = m.Update(matchesMsg{resp: matches})
	m = mm.(Model)

	st := &api.StandingsResponse{Standings: []api.Standing{
		{Stage: "GROUP_STAGE", Type: "TOTAL", Group: "GROUP_A", Table: []api.TableRow{
			{Position: 1, Team: api.Team{TLA: "MEX"}, PlayedGames: 1, Won: 1, Points: 3, GoalsFor: 2, GoalsAgainst: 1, GoalDifference: 1},
			{Position: 2, Team: api.Team{TLA: "CAN"}, PlayedGames: 1, Lost: 1, GoalsFor: 1, GoalsAgainst: 2, GoalDifference: -1},
		}},
	}}
	mm, _ = m.Update(standingsMsg{resp: st})
	m = mm.(Model)

	sc := &api.ScorersResponse{Scorers: []api.Scorer{
		{Goals: 3, Assists: ip(1), Team: api.Team{TLA: "MEX"}},
	}}
	sc.Scorers[0].Player.Name = "Santiago Giménez"
	mm, _ = m.Update(scorersMsg{resp: sc})
	m = mm.(Model)

	return m
}

func TestViewsRender(t *testing.T) {
	m := mockModel(t)

	for _, tc := range []struct {
		tab  tab
		mode standingsMode
		want string
	}{
		{tabMatches, modeBracket, "MEX"},
		{tabStandings, modeGroups, "Group A"},
		{tabScorers, modeBracket, "Giménez"},
	} {
		m.active = tc.tab
		m.stdMode = tc.mode
		out := m.View()
		if strings.TrimSpace(out) == "" {
			t.Fatalf("tab %s rendered empty", tc.tab)
		}
		if !strings.Contains(out, tc.want) {
			t.Errorf("tab %s: expected output to contain %q\n---\n%s", tc.tab, tc.want, out)
		}
	}
}

// bracketModel builds a model with a full knockout stage (QF → SF → Final
// + third place) for exercising the Bracket tab.
func bracketModel(t *testing.T) Model {
	t.Helper()
	m := New(nil)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m = mm.(Model)

	base := time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC)
	fin := func(id int, stage string, h, a string, hs, as int) api.Match {
		win := "HOME_TEAM"
		if as > hs {
			win = "AWAY_TEAM"
		}
		return api.Match{ID: id, Stage: stage, Status: "FINISHED",
			UTCDate:  base.Add(time.Duration(id) * time.Hour),
			HomeTeam: api.Team{TLA: h, Name: h}, AwayTeam: api.Team{TLA: a, Name: a},
			Score: api.Score{Winner: win, FullTime: api.ScoreLine{Home: ip(hs), Away: ip(as)}}}
	}

	matches := &api.MatchesResponse{Matches: []api.Match{
		fin(1, "QUARTER_FINALS", "BRA", "CRO", 2, 1),
		fin(2, "QUARTER_FINALS", "NED", "ARG", 0, 2),
		fin(3, "QUARTER_FINALS", "ENG", "FRA", 1, 2),
		fin(4, "QUARTER_FINALS", "MAR", "POR", 1, 0),
		fin(5, "SEMI_FINALS", "BRA", "ARG", 3, 0),
		fin(6, "SEMI_FINALS", "FRA", "MAR", 2, 0),
		fin(7, "THIRD_PLACE", "ARG", "MAR", 2, 1),
		fin(8, "FINAL", "BRA", "FRA", 3, 2),
	}}
	mm, _ = m.Update(matchesMsg{resp: matches})
	m = mm.(Model)
	// The bracket now lives inside the Standings tab (default sub-view).
	m.active = tabStandings
	m.stdMode = modeBracket
	return m
}

func TestBracketRenders(t *testing.T) {
	m := bracketModel(t)
	out := m.View()
	for _, want := range []string{"Quarter-finals", "Semi-finals", "Final", "🏆", "BRA", "3rd place"} {
		if !strings.Contains(out, want) {
			t.Errorf("bracket view missing %q\n---\n%s", want, out)
		}
	}
	t.Logf("\n%s", out)
}

func TestBracketNarrowAndScroll(t *testing.T) {
	m := bracketModel(t)
	// Narrow terminal: only the latest rounds fit; expect a hidden-rounds note.
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 44, Height: 18})
	m = mm.(Model)
	out := m.View()
	if !strings.Contains(out, "earlier round") {
		t.Errorf("narrow bracket should note hidden rounds\n---\n%s", out)
	}
	// Over-scrolling must not panic and stays clamped.
	m.bracketScroll = 9999
	m.clampCursors()
	if m.bracketScroll > m.maxBracketScroll() {
		t.Errorf("bracketScroll %d exceeds max %d", m.bracketScroll, m.maxBracketScroll())
	}
	if strings.TrimSpace(m.View()) == "" {
		t.Error("over-scrolled bracket rendered empty")
	}
}

// Empty state: no knockout matches yet.
func TestBracketEmpty(t *testing.T) {
	m := mockModel(t)
	m.active = tabStandings
	m.stdMode = modeBracket
	out := m.View()
	if !strings.Contains(out, "knockout bracket appears") {
		t.Errorf("expected empty-state message\n---\n%s", out)
	}
}

func TestFilterLive(t *testing.T) {
	m := mockModel(t)
	m.matchFilter = filterLive
	got := m.filteredMatches()
	if len(got) != 1 || got[0].HomeTeam.TLA != "USA" {
		t.Fatalf("expected only the live USA match, got %d matches", len(got))
	}
}
