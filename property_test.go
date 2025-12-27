package multielo

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
)

type matchInput struct {
	Count uint8
	Seed  int64
}

// Generate implements quick.Generator to create bounded random inputs.
func (matchInput) Generate(r *rand.Rand, size int) reflect.Value {
	return reflect.ValueOf(matchInput{
		Count: uint8(2 + r.Intn(5)), // 2..6 players
		Seed:  r.Int63(),
	})
}

func TestELOConservationProperty(t *testing.T) {
	config := DefaultConfig()
	config.MaxMatches = 1000

	prop := func(mi matchInput) bool {
		l := NewLeagueWithConfig(config)
		n := int(mi.Count)
		names := make([]string, n)
		for i := 0; i < n; i++ {
			names[i] = fmt.Sprintf("p%d", i)
			if err := l.AddPlayer(names[i]); err != nil {
				return false
			}
		}

		r := rand.New(rand.NewSource(mi.Seed))
		perm := r.Perm(n)
		results := make([]*MatchResult, n)
		for pos, idx := range perm {
			p, _ := l.GetPlayer(names[idx])
			results[pos] = &MatchResult{Position: pos + 1, Player: p}
		}

		preSum := 0
		for _, name := range names {
			elo, _ := l.GetPlayerELO(name)
			preSum += elo
		}

		if err := l.AddMatch(results); err != nil {
			return false
		}

		postSum := 0
		minELO := config.MaxELO
		maxELO := config.MinELO
		for _, name := range names {
			elo, _ := l.GetPlayerELO(name)
			postSum += elo
			if elo < minELO {
				minELO = elo
			}
			if elo > maxELO {
				maxELO = elo
			}
		}

		sumDiff := postSum - preSum
		if sumDiff < -1 || sumDiff > 1 {
			return false
		}
		if minELO < config.MinELO || maxELO > config.MaxELO {
			return false
		}
		return true
	}

	if err := quick.Check(prop, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatalf("property failed: %v", err)
	}
}
