package domain

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type League struct {
	mu               sync.RWMutex
	playersRepo      PlayerRepository
	matchesRepo      MatchRepository
	calculator       ELOCalculator
	config           LeagueConfig
	validator        Validator
	leaderboardCache []*Player
	leaderboardDirty bool
	logger           Logger
	metrics          MetricsRecorder
	archiveCallback  ArchiveCallback
	playerELOHistory map[string][]int
	// Track consecutive missed races for decay
	consecutiveMisses map[string]int // player name -> count of consecutive misses
	// lastChanges records the most recent ELO adjustments with cause
	lastChanges []LastChange
}

func NewLeague() *League {
	return NewLeagueWithConfig(DefaultConfig())
}

func NewLeagueWithConfig(config LeagueConfig) *League {
	return NewLeagueWithDependencies(config, LeagueDependencies{})
}

type LeagueDependencies struct {
	Players         PlayerRepository
	Matches         MatchRepository
	Validator       Validator
	Calculator      ELOCalculator
	Logger          Logger
	Metrics         MetricsRecorder
	ArchiveCallback ArchiveCallback
}

func NewLeagueWithDependencies(config LeagueConfig, deps LeagueDependencies) *League {
	playersRepo := deps.Players
	if playersRepo == nil {
		playersRepo = NewInMemoryPlayerRepo()
	}
	matchesRepo := deps.Matches
	if matchesRepo == nil {
		matchesRepo = NewInMemoryMatchRepo()
	}
	validator := deps.Validator
	if validator == nil {
		validator = NewDefaultValidator(config)
	}
	calculator := deps.Calculator
	if calculator == nil {
		calculator = DefaultELOCalculator{}
	}
	logger := deps.Logger
	if logger == nil {
		logger = NoopLogger{}
	}
	metrics := deps.Metrics
	if metrics == nil {
		metrics = NoopMetrics{}
	}
	archiveCallback := deps.ArchiveCallback
	if archiveCallback == nil {
		archiveCallback = NoopArchiveCallback
	}

	return &League{
		playersRepo:       playersRepo,
		matchesRepo:       matchesRepo,
		calculator:        calculator,
		config:            config,
		validator:         validator,
		leaderboardDirty:  true,
		logger:            logger,
		metrics:           metrics,
		archiveCallback:   archiveCallback,
		playerELOHistory:  make(map[string][]int),
		consecutiveMisses: make(map[string]int),
	}
}

func (l *League) AddPlayer(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.validator.ValidatePlayerName(name); err != nil {
		return err
	}
	if l.playersRepo.Count() >= l.config.MaxPlayers {
		return ErrLeagueFull
	}
	if _, exists := l.playersRepo.Get(name); exists {
		return ELOError{Type: ErrorTypePlayerExists, Message: "player already exists", Player: name}
	}
	player, err := NewPlayer(name, l.config)
	if err != nil {
		return err
	}
	if err := l.playersRepo.Save(player); err != nil {
		return err
	}
	l.playerELOHistory[name] = []int{l.config.InitialELO}
	l.leaderboardDirty = true
	l.logger.Info("player_added", map[string]interface{}{"player": name})
	l.metrics.RecordPlayerAdded()
	return nil
}

func (l *League) GetPlayer(name string) (*Player, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	player, exists := l.playersRepo.Get(name)
	if !exists {
		return nil, ELOError{Type: ErrorTypePlayerNotFound, Message: "player not found", Player: name}
	}
	return player, nil
}

func (l *League) RemovePlayer(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, exists := l.playersRepo.Get(name)
	if !exists {
		return ELOError{Type: ErrorTypePlayerNotFound, Message: "player not found", Player: name}
	}
	_ = l.playersRepo.Delete(name)
	delete(l.playerELOHistory, name)
	l.leaderboardDirty = true
	return nil
}

func (l *League) GetPlayers() []*Player {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.playersRepo.List()
}

// GetConfig returns a copy of the league configuration.
func (l *League) GetConfig() LeagueConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.config
}

func (l *League) GetPlayerELOHistory(name string) []int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	history, exists := l.playerELOHistory[name]
	if !exists {
		return nil
	}
	result := make([]int, len(history))
	copy(result, history)
	return result
}

// SyncPlayerHistories ensures all players have ELO history entries for every race.
// This is useful after replaying matches to backfill history for players who joined late.
func (l *League) SyncPlayerHistories() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.playerELOHistory) == 0 {
		return
	}

	// Find the maximum history length
	maxLen := 0
	for _, history := range l.playerELOHistory {
		if len(history) > maxLen {
			maxLen = len(history)
		}
	}

	// Backfill shorter histories with the initial ELO
	for name, history := range l.playerELOHistory {
		if len(history) < maxLen {
			newHistory := make([]int, maxLen)
			// Fill the gap with initial ELO
			for i := 0; i < maxLen-len(history); i++ {
				newHistory[i] = l.config.InitialELO
			}
			// Append existing history
			copy(newHistory[maxLen-len(history):], history)
			l.playerELOHistory[name] = newHistory
		}
	}
}

func (l *League) GetLeaderboard() []*Player {
	l.mu.RLock()
	if l.leaderboardDirty {
		l.mu.RUnlock()
		l.mu.Lock()
		if l.leaderboardDirty {
			players := l.playersRepo.List()
			sort.Slice(players, func(i, j int) bool { return players[i].ELO() > players[j].ELO() })
			l.leaderboardCache = players
			l.leaderboardDirty = false
		}
		l.mu.Unlock()
		l.mu.RLock()
	}
	result := make([]*Player, len(l.leaderboardCache))
	copy(result, l.leaderboardCache)
	l.mu.RUnlock()
	return result
}

func (l *League) AddMatch(results []*MatchResult) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.validator.ValidateMatchResults(results); err != nil {
		return err
	}
	if l.matchesRepo.Count() >= l.config.MaxMatches {
		if l.config.AutoArchive {
			matches := l.matchesRepo.List()
			if len(matches) > 0 {
				archived := []*Match{matches[0]}
				if err := l.archiveCallback(context.Background(), archived); err != nil {
					l.logger.Error("archive_callback_failed", err, map[string]interface{}{"match_count": 1})
				}
			}
			l.matchesRepo.TrimOldest()
		} else {
			return ELOError{Type: ErrorTypeConflict, Message: "league has reached maximum match capacity", Context: map[string]interface{}{"max_matches": l.config.MaxMatches}}
		}
	}
	populatedResults := make([]*MatchResult, len(results))
	for i, result := range results {
		name := result.Player.Name()
		player, exists := l.playersRepo.Get(name)
		if !exists {
			if l.playersRepo.Count() >= l.config.MaxPlayers {
				return ErrLeagueFull
			}
			if err := l.validator.ValidatePlayerName(name); err != nil {
				return err
			}
			var err error
			player, err = NewPlayer(name, l.config)
			if err != nil {
				return err
			}
			if err := l.playersRepo.Save(player); err != nil {
				return err
			}
			l.leaderboardDirty = true
		}
		populatedResults[i] = &MatchResult{Position: result.Position, Player: player}
	}
	changes, err := l.calculator.Calculate(populatedResults, l.config)
	if err != nil {
		return err
	}
	// Collect changes for reporting
	l.lastChanges = nil
	for _, change := range changes {
		if err := change.Player.UpdateELO(change.Diff, l.validator); err != nil {
			return fmt.Errorf("failed to update player %s ELO: %w", change.Player.Name(), err)
		}
		position := getPlayerPosition(populatedResults, change.Player)
		won := position == 1
		change.Player.RecordMatch(position, won)
		l.playerELOHistory[change.Player.Name()] = append(l.playerELOHistory[change.Player.Name()], change.Player.ELO())
		// Reset consecutive misses on participation
		l.consecutiveMisses[change.Player.Name()] = 0
		l.lastChanges = append(l.lastChanges, LastChange{
			Name:     change.Player.Name(),
			Diff:     change.Diff,
			Cause:    "position",
			Position: position,
			Attended: true,
		})
	}

	allPlayers := l.playersRepo.List()
	for _, player := range allPlayers {
		participated := false
		for _, result := range populatedResults {
			if result.Player == player {
				participated = true
				break
			}
		}
		if !participated {
			// Apply decay penalty for missed race
			if l.config.DecayEnabled {
				misses := l.consecutiveMisses[player.Name()]
				penaltyPercent := l.config.DecayInitialPercent + (float64(misses) * l.config.DecayPerMiss)
				penaltyELO := int(float64(player.ELO()) * penaltyPercent / 100.0)
				if penaltyELO > 0 {
					newELO := player.ELO() - penaltyELO
					if newELO < l.config.MinELO {
						newELO = l.config.MinELO
					}
					_ = player.UpdateELO(-penaltyELO, l.validator)
					l.logger.Info("elo_decay_applied", map[string]interface{}{
						"player":           player.Name(),
						"penalty_elo":      penaltyELO,
						"consecutive_miss": misses,
						"new_elo":          newELO,
					})
					l.lastChanges = append(l.lastChanges, LastChange{
						Name:     player.Name(),
						Diff:     -penaltyELO,
						Cause:    "decay",
						Position: 0,
						Attended: false,
					})
				}
				l.consecutiveMisses[player.Name()] = misses + 1
			}
			l.playerELOHistory[player.Name()] = append(l.playerELOHistory[player.Name()], player.ELO())
		}
	}

	match := &Match{Results: populatedResults, Date: time.Now()}
	_ = l.matchesRepo.Add(match)
	l.leaderboardDirty = true
	l.logger.Info("match_added", map[string]interface{}{"player_count": len(populatedResults)})
	l.metrics.RecordMatchAdded(len(populatedResults))
	return nil
}

func getPlayerPosition(results []*MatchResult, player *Player) int {
	for _, result := range results {
		if result.Player == player {
			return result.Position
		}
	}
	return 0
}

// LastChange represents a single player's ELO change with its cause.
type LastChange struct {
	Name     string
	Diff     int
	Cause    string // "position" or "decay"
	Position int    // finishing position when attended, 0 otherwise
	Attended bool
}

// GetLastChanges returns a copy of the last recorded changes after AddMatch.
func (l *League) GetLastChanges() []LastChange {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.lastChanges) == 0 {
		return nil
	}
	out := make([]LastChange, len(l.lastChanges))
	copy(out, l.lastChanges)
	return out
}

func (l *League) ResetPlayers() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.playersRepo.Reset()
	l.playerELOHistory = make(map[string][]int)
	l.consecutiveMisses = make(map[string]int)
	l.leaderboardCache = nil
	l.leaderboardDirty = true
}

func (l *League) ResetMatches() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.matchesRepo.Reset()
	for name := range l.playerELOHistory {
		l.playerELOHistory[name] = []int{l.config.InitialELO}
	}
	l.consecutiveMisses = make(map[string]int)
}

func (l *League) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.playersRepo.Reset()
	l.matchesRepo.Reset()
	l.playerELOHistory = make(map[string][]int)
	l.consecutiveMisses = make(map[string]int)
	l.leaderboardCache = nil
	l.leaderboardDirty = true
}

func (l *League) GetMatches() []*Match {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.matchesRepo.List()
}

func (l *League) GetPlayerStats(name string) (*PlayerStats, error) {
	player, err := l.GetPlayer(name)
	if err != nil {
		return nil, err
	}
	return player.GetStats(), nil
}
func (l *League) GetPlayerELO(name string) (int, error) {
	player, err := l.GetPlayer(name)
	if err != nil {
		return 0, err
	}
	return player.ELO(), nil
}
