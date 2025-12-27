package domain

import (
	"strings"
)

type Validator interface {
	ValidatePlayerName(name string) error
	ValidateELOValue(elo int) error
	ValidateMatchResults(results []*MatchResult) error
	ValidatePosition(position int) error
}

type LeagueConfig struct {
	InitialELO          int    `json:"initial_elo"`
	MinELO              int    `json:"min_elo"`
	MaxELO              int    `json:"max_elo"`
	MaxMatches          int    `json:"max_matches"`
	MaxPlayers          int    `json:"max_players"`
	KFactor             int    `json:"k_factor"`
	AutoArchive         bool   `json:"auto_archive"`
	MaxPlayerHistoryLen int    `json:"max_player_history_len"`
	OutputDirectory     string `json:"output_directory"`
	// ELO decay for missed races
	DecayEnabled        bool    `json:"decay_enabled"`
	DecayInitialPercent float64 `json:"decay_initial_percent"`  // e.g., 2.0 for 2% of current ELO
	DecayPerMiss        float64 `json:"decay_per_miss_percent"` // e.g., 0.5 for additional 0.5% per consecutive miss
}

func DefaultConfig() LeagueConfig {
	return LeagueConfig{
		InitialELO:          1000,
		MinELO:              100,
		MaxELO:              3000,
		MaxMatches:          10000,
		MaxPlayers:          1000,
		KFactor:             32,
		AutoArchive:         false,
		MaxPlayerHistoryLen: 100,
		OutputDirectory:     "output",
		DecayEnabled:        false,
		DecayInitialPercent: 2.0,
		DecayPerMiss:        0.5,
	}
}

type DefaultValidator struct {
	config LeagueConfig
}

func NewDefaultValidator(config LeagueConfig) *DefaultValidator {
	return &DefaultValidator{config: config}
}

func (v *DefaultValidator) ValidatePlayerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrInvalidPlayerName
	}
	if len(name) > 50 {
		return ELOError{
			Type:    ErrorTypeValidation,
			Message: "player name too long",
			Context: map[string]interface{}{"length": len(name)},
		}
	}
	return nil
}

func (v *DefaultValidator) ValidateELOValue(elo int) error {
	if elo < v.config.MinELO {
		return ELOError{
			Type:    ErrorTypeValidation,
			Message: "ELO below minimum",
			Context: map[string]interface{}{"elo": elo, "min": v.config.MinELO},
		}
	}
	if elo > v.config.MaxELO {
		return ELOError{
			Type:    ErrorTypeValidation,
			Message: "ELO exceeds maximum",
			Context: map[string]interface{}{"elo": elo, "max": v.config.MaxELO},
		}
	}
	return nil
}

func (v *DefaultValidator) ValidateMatchResults(results []*MatchResult) error {
	if len(results) < 2 {
		return ELOError{
			Type:    ErrorTypeValidation,
			Message: "match requires at least 2 players",
			Context: map[string]interface{}{"player_count": len(results)},
		}
	}

	positions := make(map[int]bool)
	for _, result := range results {
		if err := v.ValidatePosition(result.Position); err != nil {
			return err
		}
		if positions[result.Position] {
			return ELOError{
				Type:    ErrorTypeValidation,
				Message: "duplicate position in match",
				Context: map[string]interface{}{"position": result.Position},
			}
		}
		positions[result.Position] = true

		if result.Player == nil {
			return ELOError{
				Type:    ErrorTypeValidation,
				Message: "player cannot be nil",
				Context: map[string]interface{}{"position": result.Position},
			}
		}
	}
	return nil
}

func (v *DefaultValidator) ValidatePosition(position int) error {
	if position <= 0 {
		return ELOError{
			Type:    ErrorTypeValidation,
			Message: "position must be positive",
			Context: map[string]interface{}{"position": position},
		}
	}
	return nil
}
