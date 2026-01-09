package domain

import (
	"context"
	"sync"
	"time"
)

type MultiLeagueService struct {
	mu      sync.RWMutex
	leagues map[string]*League
	config  LeagueConfig
	deps    LeagueDependencies
}

func NewMultiLeagueService(config LeagueConfig, deps LeagueDependencies) *MultiLeagueService {
	return &MultiLeagueService{
		leagues: make(map[string]*League),
		config:  config,
		deps:    deps,
	}
}

func (m *MultiLeagueService) CreateLeague(ctx context.Context, leagueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.leagues[leagueID]; exists {
		return ELOError{
			Type:    ErrorTypeConflict,
			Message: "league already exists",
			Context: map[string]interface{}{"league_id": leagueID},
		}
	}

	league := NewLeagueWithDependencies(m.config, m.deps)
	m.leagues[leagueID] = league
	return nil
}

func (m *MultiLeagueService) GetLeague(ctx context.Context, leagueID string) (*League, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	league, exists := m.leagues[leagueID]
	if !exists {
		return nil, ELOError{
			Type:    ErrorTypeNotFound,
			Message: "league not found",
			Context: map[string]interface{}{"league_id": leagueID},
		}
	}
	return league, nil
}

func (m *MultiLeagueService) DeleteLeague(ctx context.Context, leagueID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.leagues[leagueID]; !exists {
		return ELOError{
			Type:    ErrorTypeNotFound,
			Message: "league not found",
			Context: map[string]interface{}{"league_id": leagueID},
		}
	}

	delete(m.leagues, leagueID)
	return nil
}

func (m *MultiLeagueService) ListLeagueIDs(ctx context.Context) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.leagues))
	for id := range m.leagues {
		ids = append(ids, id)
	}
	return ids
}

func (m *MultiLeagueService) AddPlayer(ctx context.Context, leagueID, playerName string) error {
	league, err := m.GetLeague(ctx, leagueID)
	if err != nil {
		return err
	}
	return league.AddPlayer(playerName)
}

func (m *MultiLeagueService) RecordMatch(ctx context.Context, leagueID string, results []*MatchResult, date time.Time) error {
	league, err := m.GetLeague(ctx, leagueID)
	if err != nil {
		return err
	}
	return league.AddMatch(results, date)
}

func (m *MultiLeagueService) GetLeaderboard(ctx context.Context, leagueID string) ([]*Player, error) {
	league, err := m.GetLeague(ctx, leagueID)
	if err != nil {
		return nil, err
	}
	return league.GetLeaderboard(), nil
}
