package domain

import "time"

type Match struct {
	Results []*MatchResult
	Date    time.Time
}

type MatchResult struct {
	Position int
	Player   *Player
}

type MatchDiff struct {
	Player *Player
	Diff   int
}

type MatchService interface {
	ValidateMatch(results []*MatchResult) error
	CalculateELOChanges(results []*MatchResult, config LeagueConfig) ([]MatchDiff, error)
	RecordMatch(results []*MatchResult) (*Match, error)
}
