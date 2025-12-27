package multielo

import (
	domain "github.com/distrobyte/multielo/domain"
	"github.com/distrobyte/multielo/infrastructure"
)

type ErrorType = domain.ErrorType
type ELOError = domain.ELOError

var (
	ErrInvalidPlayerName   = domain.ErrInvalidPlayerName
	ErrPlayerAlreadyExists = domain.ErrPlayerAlreadyExists
	ErrPlayerNotFound      = domain.ErrPlayerNotFound
	ErrELOOutOfBounds      = domain.ErrELOOutOfBounds
	ErrInvalidPosition     = domain.ErrInvalidPosition
	ErrInvalidMatch        = domain.ErrInvalidMatch
	ErrLeagueFull          = domain.ErrLeagueFull
	ErrNoPlayers           = domain.ErrNoPlayers
)

const (
	ErrorTypeValidation     = domain.ErrorTypeValidation
	ErrorTypeNotFound       = domain.ErrorTypeNotFound
	ErrorTypePlayerNotFound = domain.ErrorTypePlayerNotFound
	ErrorTypePlayerExists   = domain.ErrorTypePlayerExists
	ErrorTypeConflict       = domain.ErrorTypeConflict
	ErrorTypeInternal       = domain.ErrorTypeInternal
)

type LeagueConfig = domain.LeagueConfig

func DefaultConfig() LeagueConfig { return domain.DefaultConfig() }

type Validator = domain.Validator
type DefaultValidator = domain.DefaultValidator

func NewDefaultValidator(cfg LeagueConfig) *DefaultValidator { return domain.NewDefaultValidator(cfg) }

type Player = domain.Player
type PlayerStats = domain.PlayerStats
type Match = domain.Match
type MatchResult = domain.MatchResult
type MatchDiff = domain.MatchDiff
type League = domain.League
type LeagueDependencies = domain.LeagueDependencies
type LeagueCommands = domain.LeagueCommands
type LeagueQueries = domain.LeagueQueries
type LeagueService = domain.LeagueService
type ELOCalculator = domain.ELOCalculator
type Logger = domain.Logger
type MetricsRecorder = domain.MetricsRecorder
type ArchiveCallback = domain.ArchiveCallback
type MultiLeagueService = domain.MultiLeagueService
type MatchFilter = domain.MatchFilter
type MatchQueryResult = domain.MatchQueryResult
type GraphRenderer = domain.GraphRenderer
type LastChange = domain.LastChange

func NewLeague() *League                           { return domain.NewLeague() }
func NewLeagueWithConfig(cfg LeagueConfig) *League { return domain.NewLeagueWithConfig(cfg) }
func NewLeagueWithDependencies(cfg LeagueConfig, deps LeagueDependencies) *League {
	return domain.NewLeagueWithDependencies(cfg, deps)
}
func NewLeagueService(league *League) *LeagueService { return domain.NewLeagueService(league) }
func NewMultiLeagueService(cfg LeagueConfig, deps LeagueDependencies) *MultiLeagueService {
	return domain.NewMultiLeagueService(cfg, deps)
}
func NewPlayer(name string, cfg LeagueConfig) (*Player, error) { return domain.NewPlayer(name, cfg) }

func NewHTMLGraphRenderer() GraphRenderer        { return infrastructure.NewHTMLGraphRenderer() }
func NewGraphRenderer() GraphRenderer            { return infrastructure.NewGraphRenderer() }
func NewMultiGraphRenderer() GraphRenderer       { return infrastructure.NewMultiGraphRenderer() }
func SetGraphRenderer(r GraphRenderer)           { domain.SetGraphRenderer(r) }
func SyncPlayerHistories(league *League)         { league.SyncPlayerHistories() }
func GetLastChanges(league *League) []LastChange { return league.GetLastChanges() }

const InitialELO = 1000

func init() {
	// Default: generate HTML + PNG/SVG using a composite renderer.
	domain.SetGraphRenderer(infrastructure.NewMultiGraphRenderer())
}
