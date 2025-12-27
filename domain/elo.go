package domain

import "math"

type ELOCalculator interface {
	Calculate(results []*MatchResult, cfg LeagueConfig) ([]MatchDiff, error)
}

type DefaultELOCalculator struct{}

func (DefaultELOCalculator) Calculate(results []*MatchResult, cfg LeagueConfig) ([]MatchDiff, error) {
	n := len(results)
	if n < 2 {
		return nil, ErrInvalidMatch
	}

	kValue := roundK(cfg.KFactor, n)
	changes := make([]MatchDiff, n)

	for i, result := range results {
		changes[i] = MatchDiff{Player: result.Player, Diff: 0}
	}

	for i, result := range results {
		currentELO := result.Player.ELO()
		currentPosition := result.Position

		for j, opponentResult := range results {
			if i == j {
				continue
			}

			opponentELO := opponentResult.Player.ELO()
			opponentPosition := opponentResult.Position

			var score float64
			if currentPosition < opponentPosition {
				score = 1.0
			} else if currentPosition == opponentPosition {
				score = 0.5
			} else {
				score = 0.0
			}

			expectedScore := expectedScore(currentELO, opponentELO)
			eloChange := kValue * (score - expectedScore)
			changes[i].Diff += int(eloChange)
		}
	}

	return changes, nil
}

func CalculateELOChanges(results []*MatchResult, cfg LeagueConfig) ([]MatchDiff, error) {
	return DefaultELOCalculator{}.Calculate(results, cfg)
}

func roundK(k int, n int) float64 {
	return math.Round(float64(k) / float64(n-1))
}

func expectedScore(aELO, bELO int) float64 {
	return 1.0 / (1.0 + math.Pow(10, float64(bELO-aELO)/400))
}
