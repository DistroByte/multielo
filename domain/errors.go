package domain

import (
	"errors"
	"fmt"
)

type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "VALIDATION"
	ErrorTypeNotFound       ErrorType = "NOT_FOUND"
	ErrorTypePlayerNotFound ErrorType = "PLAYER_NOT_FOUND"
	ErrorTypePlayerExists   ErrorType = "PLAYER_EXISTS"
	ErrorTypeConflict       ErrorType = "CONFLICT"
	ErrorTypeInternal       ErrorType = "INTERNAL"
)

type ELOError struct {
	Type    ErrorType
	Message string
	Player  string
	Cause   error
	Context map[string]interface{}
}

func (e ELOError) Error() string {
	if e.Player != "" {
		return fmt.Sprintf("[%s] %s (player: %s)", e.Type, e.Message, e.Player)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e ELOError) Unwrap() error {
	return e.Cause
}

var (
	ErrInvalidPlayerName   = ELOError{Type: ErrorTypeValidation, Message: "player name cannot be empty or whitespace only"}
	ErrPlayerAlreadyExists = ELOError{Type: ErrorTypePlayerExists, Message: "player already exists"}
	ErrPlayerNotFound      = ELOError{Type: ErrorTypePlayerNotFound, Message: "player not found"}
	ErrELOOutOfBounds      = ELOError{Type: ErrorTypeValidation, Message: "ELO value out of allowed range"}
	ErrInvalidPosition     = ELOError{Type: ErrorTypeValidation, Message: "match position must be positive"}
	ErrInvalidMatch        = ELOError{Type: ErrorTypeValidation, Message: "invalid match data"}
	ErrLeagueFull          = ELOError{Type: ErrorTypeConflict, Message: "league has reached maximum player capacity"}
	ErrNoPlayers           = ELOError{Type: ErrorTypeValidation, Message: "no players in league"}
)

var (
	ErrMatchNotFound      = errors.New("match not found")
	ErrInvalidELOChange   = errors.New("invalid elo change")
	ErrInvalidPlayer      = errors.New("invalid player")
	ErrInvalidPlayerStats = errors.New("invalid player stats")
	ErrInvalidLeague      = errors.New("invalid league")
)
