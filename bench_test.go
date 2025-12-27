package multielo

import (
	"testing"
)

func setupLeagueForBenchmark(playerCount int) (*League, []*MatchResult) {
	cfg := DefaultConfig()
	cfg.MaxMatches = 1 << 30 // effectively unbounded for benchmark runs
	l := NewLeagueWithConfig(cfg)

	players := make([]*Player, playerCount)
	for i := 0; i < playerCount; i++ {
		name := string(rune('A'+i%26)) + string(rune('a'+(i/26)))
		_ = l.AddPlayer(name)
		p, _ := l.GetPlayer(name)
		players[i] = p
	}

	results := make([]*MatchResult, playerCount)
	for i := 0; i < playerCount; i++ {
		results[i] = &MatchResult{Position: i + 1, Player: players[i]}
	}
	return l, results
}

// Benchmark: Small match (8 players)
func BenchmarkAddMatchSmall(b *testing.B) {
	l, results := setupLeagueForBenchmark(8)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := l.AddMatch(results); err != nil {
			b.Fatalf("add match failed: %v", err)
		}
		if i%1000 == 0 {
			l.ResetMatches()
		}
	}
}

// Benchmark: Medium match (20 players)
func BenchmarkAddMatchMedium(b *testing.B) {
	l, results := setupLeagueForBenchmark(20)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := l.AddMatch(results); err != nil {
			b.Fatalf("add match failed: %v", err)
		}
		if i%1000 == 0 {
			l.ResetMatches()
		}
	}
}

// Benchmark: Large match (50 players)
func BenchmarkAddMatchLarge(b *testing.B) {
	l, results := setupLeagueForBenchmark(50)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := l.AddMatch(results); err != nil {
			b.Fatalf("add match failed: %v", err)
		}
		if i%100 == 0 {
			l.ResetMatches()
		}
	}
}

// Benchmark: GetPlayer lookups (should be O(1))
func BenchmarkGetPlayer(b *testing.B) {
	l, _ := setupLeagueForBenchmark(100)
	testName := "Aa" // Known player from setup

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.GetPlayer(testName)
		if err != nil {
			b.Fatalf("get player failed: %v", err)
		}
	}
}

// Benchmark: GetLeaderboard with leaderboard caching
func BenchmarkGetLeaderboard(b *testing.B) {
	l, results := setupLeagueForBenchmark(30)

	// Add some matches to populate the league
	for i := 0; i < 10; i++ {
		_ = l.AddMatch(results)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.GetLeaderboard()
	}
}

// Benchmark: GetPlayerStats
func BenchmarkGetPlayerStats(b *testing.B) {
	l, results := setupLeagueForBenchmark(20)

	// Add some matches to build up stats
	for i := 0; i < 50; i++ {
		_ = l.AddMatch(results)
	}

	player, _ := l.GetPlayer("Aa")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = player.GetStats()
	}
}

// Benchmark: League initialization with many players
func BenchmarkLeagueInitialization(b *testing.B) {
	const playerCount = 100
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cfg := DefaultConfig()
		cfg.MaxMatches = 1 << 30
		l := NewLeagueWithConfig(cfg)

		for j := 0; j < playerCount; j++ {
			name := string(rune('A'+(j%26))) + string(rune('a'+(j/26)))
			_ = l.AddPlayer(name)
		}
	}
}

// Benchmark: Service layer AddPlayer
func BenchmarkServiceAddPlayer(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := DefaultConfig()
		cfg.MaxMatches = 1 << 30
		cfg.MaxPlayers = 10 // Just need a few players per iteration
		l := NewLeagueWithConfig(cfg)
		service := NewLeagueService(l)

		for j := 0; j < 10; j++ {
			name := "Player" + string(rune('A'+(j%26)))
			if err := service.AddPlayer(name); err != nil {
				b.Fatalf("add player failed: %v", err)
			}
		}
	}
}

// Benchmark: Match filtering
func BenchmarkMatchFiltering(b *testing.B) {
	l, results := setupLeagueForBenchmark(10)

	// Add many matches
	for i := 0; i < 100; i++ {
		_ = l.AddMatch(results)
	}

	filter := MatchFilter{
		Limit:  10,
		Offset: 0,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = l.GetMatchesFiltered(filter)
	}
}
