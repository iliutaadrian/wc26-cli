package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iliutaadrian/wc26/internal/api"
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
		want string
	}{
		{tabMatches, "MEX"},
		{tabStandings, "Group A"},
		{tabScorers, "Giménez"},
	} {
		m.active = tc.tab
		out := m.View()
		if strings.TrimSpace(out) == "" {
			t.Fatalf("tab %s rendered empty", tc.tab)
		}
		if !strings.Contains(out, tc.want) {
			t.Errorf("tab %s: expected output to contain %q\n---\n%s", tc.tab, tc.want, out)
		}
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
