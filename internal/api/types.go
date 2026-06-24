package api

import "time"

// --- Shared ---

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ShortName string `json:"shortName"`
	TLA       string `json:"tla"`
	Crest     string `json:"crest"`
}

// DisplayName prefers the 3-letter code, falling back to the full name.
func (t Team) DisplayName() string {
	if t.TLA != "" {
		return t.TLA
	}
	if t.ShortName != "" {
		return t.ShortName
	}
	if t.Name != "" {
		return t.Name
	}
	return "TBD"
}

type Competition struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Emblem string `json:"emblem"`
}

// --- Matches ---

type ScoreLine struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

type Score struct {
	Winner   string    `json:"winner"`
	Duration string    `json:"duration"`
	FullTime ScoreLine `json:"fullTime"`
	HalfTime ScoreLine `json:"halfTime"`
}

type Match struct {
	ID       int       `json:"id"`
	UTCDate  time.Time `json:"utcDate"`
	Status   string    `json:"status"`
	Matchday int       `json:"matchday"`
	Stage    string    `json:"stage"`
	Group    string    `json:"group"`
	HomeTeam Team      `json:"homeTeam"`
	AwayTeam Team      `json:"awayTeam"`
	Score    Score     `json:"score"`
}

// IsLive reports whether the match is currently being played.
func (m Match) IsLive() bool {
	return m.Status == "IN_PLAY" || m.Status == "PAUSED"
}

// IsFinished reports whether the match has ended.
func (m Match) IsFinished() bool {
	return m.Status == "FINISHED"
}

type MatchesResponse struct {
	Competition Competition `json:"competition"`
	ResultSet   struct {
		Count  int `json:"count"`
		Played int `json:"played"`
	} `json:"resultSet"`
	Matches []Match `json:"matches"`
}

// --- Standings ---

type TableRow struct {
	Position       int  `json:"position"`
	Team           Team `json:"team"`
	PlayedGames    int  `json:"playedGames"`
	Won            int  `json:"won"`
	Draw           int  `json:"draw"`
	Lost           int  `json:"lost"`
	Points         int  `json:"points"`
	GoalsFor       int  `json:"goalsFor"`
	GoalsAgainst   int  `json:"goalsAgainst"`
	GoalDifference int  `json:"goalDifference"`
}

type Standing struct {
	Stage string     `json:"stage"`
	Type  string     `json:"type"`
	Group string     `json:"group"`
	Table []TableRow `json:"table"`
}

type StandingsResponse struct {
	Competition Competition `json:"competition"`
	Standings   []Standing  `json:"standings"`
}

// --- Scorers ---

type Scorer struct {
	Player struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		Nationality string `json:"nationality"`
	} `json:"player"`
	Team         Team `json:"team"`
	Goals        int  `json:"goals"`
	Assists      *int `json:"assists"`
	Penalties    *int `json:"penalties"`
	PlayedMatches int `json:"playedMatches"`
}

type ScorersResponse struct {
	Competition Competition `json:"competition"`
	Scorers     []Scorer    `json:"scorers"`
}
