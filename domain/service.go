package domain

import "time"

type LeagueCommands interface {
	AddPlayer(name string) error
	RemovePlayer(name string) error
	RecordMatch(results []*MatchResult) error
	Reset()
	ResetPlayers()
	ResetMatches()
}

type LeagueQueries interface {
	Player(name string) (*Player, error)
	Players() []*Player
	Leaderboard() []*Player
	Matches() []*Match
	PlayerStats(name string) (*PlayerStats, error)
	PlayerELO(name string) (int, error)
}

type LeagueService struct {
	league *League
}

func NewLeagueService(league *League) *LeagueService {
	if league == nil {
		league = NewLeague()
	}
	return &LeagueService{league: league}
}

func (s *LeagueService) AddPlayer(name string) error    { return s.league.AddPlayer(name) }
func (s *LeagueService) RemovePlayer(name string) error { return s.league.RemovePlayer(name) }
func (s *LeagueService) RecordMatch(results []*MatchResult, date time.Time) error {
	return s.league.AddMatch(results, date)
}
func (s *LeagueService) Reset()        { s.league.Reset() }
func (s *LeagueService) ResetPlayers() { s.league.ResetPlayers() }
func (s *LeagueService) ResetMatches() { s.league.ResetMatches() }

func (s *LeagueService) Player(name string) (*Player, error) { return s.league.GetPlayer(name) }
func (s *LeagueService) Players() []*Player                  { return s.league.GetPlayers() }
func (s *LeagueService) Leaderboard() []*Player              { return s.league.GetLeaderboard() }
func (s *LeagueService) Matches() []*Match                   { return s.league.GetMatches() }
func (s *LeagueService) PlayerStats(name string) (*PlayerStats, error) {
	return s.league.GetPlayerStats(name)
}
func (s *LeagueService) PlayerELO(name string) (int, error) { return s.league.GetPlayerELO(name) }

func (s *LeagueService) League() *League { return s.league }
