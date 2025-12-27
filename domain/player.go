package domain

import (
	"strings"
	"sync"
	"time"
)

type Player struct {
	mu              sync.RWMutex
	name            string
	elo             int
	eloChange       int
	matchesPlayed   int
	matchesWon      int
	last5Finish     []int
	allTimeAvgPlace float64
	peakELO         int
	joined          time.Time
}

func (p *Player) Name() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.name
}

func (p *Player) ELO() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.elo
}

func (p *Player) ELOChange() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.eloChange
}

func (p *Player) MatchesPlayed() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.matchesPlayed
}

func (p *Player) MatchesWon() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.matchesWon
}

func (p *Player) Last5Finish() []int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]int, len(p.last5Finish))
	copy(result, p.last5Finish)
	return result
}

func (p *Player) AllTimeAvgPlace() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.allTimeAvgPlace
}

func (p *Player) PeakELO() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.peakELO
}
func (p *Player) Joined() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.joined
}

func (p *Player) UpdateELO(change int, validator Validator) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	newELO := p.elo + change
	if err := validator.ValidateELOValue(newELO); err != nil {
		return err
	}

	p.elo = newELO
	p.eloChange = change

	if p.elo > p.peakELO {
		p.peakELO = p.elo
	}

	return nil
}

func (p *Player) RecordMatch(position int, won bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.matchesPlayed++
	if won {
		p.matchesWon++
	}

	p.last5Finish = append(p.last5Finish, position)
	if len(p.last5Finish) > 5 {
		p.last5Finish = p.last5Finish[1:]
	}

	totalPositions := 0
	for _, pos := range p.last5Finish {
		totalPositions += pos
	}
	if len(p.last5Finish) > 0 {
		p.allTimeAvgPlace = float64(totalPositions) / float64(len(p.last5Finish))
	}
}

type PlayerStats struct {
	MatchesPlayed       int     `json:"matchesPlayed"`
	MatchesWon          int     `json:"matchesWon"`
	AllTimeAveragePlace float64 `json:"allTimeAveragePlace"`
	Last5Finish         []int   `json:"last5Finish"`
	PeakELO             int     `json:"peakELO"`
}

func (p *Player) GetStats() *PlayerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return &PlayerStats{
		MatchesPlayed:       p.matchesPlayed,
		MatchesWon:          p.matchesWon,
		AllTimeAveragePlace: p.allTimeAvgPlace,
		Last5Finish:         p.Last5Finish(),
		PeakELO:             p.peakELO,
	}
}

func NewPlayer(name string, config LeagueConfig) (*Player, error) {
	validator := NewDefaultValidator(config)
	if err := validator.ValidatePlayerName(name); err != nil {
		return nil, err
	}

	if err := validator.ValidateELOValue(config.InitialELO); err != nil {
		return nil, err
	}

	return &Player{
		name:            strings.TrimSpace(name),
		elo:             config.InitialELO,
		eloChange:       0,
		matchesPlayed:   0,
		matchesWon:      0,
		last5Finish:     make([]int, 0, 5),
		allTimeAvgPlace: 0.0,
		peakELO:         config.InitialELO,
		joined:          time.Now(),
	}, nil
}

type PlayerService interface {
	CreatePlayer(name string) (*Player, error)
	ValidatePlayer(player *Player) error
	UpdatePlayerELO(player *Player, change int) error
}
