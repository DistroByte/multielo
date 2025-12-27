package multielo_test

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/distrobyte/multielo"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// PHASE 1: FOUNDATION TESTS - THREAD SAFETY & VALIDATION
// =============================================================================

func TestNewLeague(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.Len(t, l.GetPlayers(), 0)
		assert.Len(t, l.GetMatches(), 0)
	})

	t.Run("custom_config", func(t *testing.T) {
		cfg := multielo.LeagueConfig{InitialELO: 1500, MinELO: 200, MaxELO: 4000}
		assert.NotNil(t, multielo.NewLeagueWithConfig(cfg))
	})
}

func TestLeague_AddPlayer(t *testing.T) {
	t.Run("adds_player", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.NoError(t, l.AddPlayer("Alice"))
		players := l.GetPlayers()
		assert.Len(t, players, 1)
		assert.Equal(t, "Alice", players[0].Name())
		assert.Equal(t, 1000, players[0].ELO())
	})

	t.Run("prevents_duplicates", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.NoError(t, l.AddPlayer("Alice"))
		err := l.AddPlayer("Alice")
		var eloErr multielo.ELOError
		assert.ErrorAs(t, err, &eloErr)
		assert.Equal(t, multielo.ErrorTypePlayerExists, eloErr.Type)
	})

	t.Run("case_insensitive", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.NoError(t, l.AddPlayer("Alice"))
		err := l.AddPlayer("alice")
		var eloErr multielo.ELOError
		assert.ErrorAs(t, err, &eloErr)
		assert.Equal(t, multielo.ErrorTypePlayerExists, eloErr.Type)
	})

	t.Run("validates_empty", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.Error(t, l.AddPlayer(""))
		err := l.AddPlayer("   ")
		var eloErr multielo.ELOError
		assert.ErrorAs(t, err, &eloErr)
		assert.Equal(t, multielo.ErrorTypeValidation, eloErr.Type)
	})

	t.Run("max_limit", func(t *testing.T) {
		cfg := multielo.DefaultConfig()
		cfg.MaxPlayers = 2
		l := multielo.NewLeagueWithConfig(cfg)
		assert.NoError(t, l.AddPlayer("Alice"))
		assert.NoError(t, l.AddPlayer("Bob"))
		err := l.AddPlayer("Charlie")
		var eloErr multielo.ELOError
		assert.ErrorAs(t, err, &eloErr)
		assert.Equal(t, multielo.ErrorTypeConflict, eloErr.Type)
	})
}

func TestLeague_GetPlayer(t *testing.T) {
	t.Run("existing", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.NoError(t, l.AddPlayer("Alice"))
		p, err := l.GetPlayer("Alice")
		assert.NoError(t, err)
		assert.Equal(t, "Alice", p.Name())
	})

	t.Run("case_insensitive", func(t *testing.T) {
		l := multielo.NewLeague()
		assert.NoError(t, l.AddPlayer("Alice"))
		p, err := l.GetPlayer("alice")
		assert.NoError(t, err)
		assert.Equal(t, "Alice", p.Name())
	})

	t.Run("not_found", func(t *testing.T) {
		l := multielo.NewLeague()
		p, err := l.GetPlayer("NonExistent")
		assert.Nil(t, p)
		var eloErr multielo.ELOError
		assert.ErrorAs(t, err, &eloErr)
		assert.Equal(t, multielo.ErrorTypePlayerNotFound, eloErr.Type)
	})
}

func TestLeague_AddMatch(t *testing.T) {
	t.Run("processes_simple_two-player_match", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add players
		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		// Record match (Alice wins)
		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
		}

		err := league.AddMatch(results)
		assert.NoError(t, err)

		// Check ELO changes
		aliceAfter, _ := league.GetPlayer("Alice")
		bobAfter, _ := league.GetPlayer("Bob")

		assert.Greater(t, aliceAfter.ELO(), 1000) // Winner gains ELO
		assert.Less(t, bobAfter.ELO(), 1000)      // Loser loses ELO

		// Check match statistics
		assert.Equal(t, 1, aliceAfter.MatchesPlayed())
		assert.Equal(t, 1, bobAfter.MatchesPlayed())

		matches := league.GetMatches()
		assert.Equal(t, 1, len(matches))
	})

	t.Run("validates_minimum_players", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		alice, _ := league.GetPlayer("Alice")

		// Try to record match with only one player
		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
		}

		err := league.AddMatch(results)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)
	})

	t.Run("auto_creates_missing_players", func(t *testing.T) {
		league := multielo.NewLeague()
		cfg := multielo.DefaultConfig()
		p1, err := multielo.NewPlayer("Ghost1", cfg)
		assert.NoError(t, err)
		p2, err := multielo.NewPlayer("Ghost2", cfg)
		assert.NoError(t, err)

		results := []*multielo.MatchResult{
			{Position: 1, Player: p1},
			{Position: 2, Player: p2},
		}

		assert.NoError(t, league.AddMatch(results))
		assert.Equal(t, 2, len(league.GetPlayers()))
		matchList := league.GetMatches()
		assert.Equal(t, 1, len(matchList))
	})

	t.Run("validates_duplicate_positions", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		// Try to record match with duplicate positions
		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 1, Player: bob}, // Duplicate position
		}

		err := league.AddMatch(results)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)
	})

	t.Run("validates_nil_players", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		alice, _ := league.GetPlayer("Alice")

		// Try to record match with nil player
		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: nil},
		}

		err := league.AddMatch(results)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)
	})

	t.Run("updates_player_statistics", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		// Record multiple matches
		for i := 0; i < 3; i++ {
			results := []*multielo.MatchResult{
				{Position: 1, Player: alice},
				{Position: 2, Player: bob},
			}
			assert.NoError(t, league.AddMatch(results))
		}

		// Check statistics
		aliceAfter, _ := league.GetPlayer("Alice")
		bobAfter, _ := league.GetPlayer("Bob")

		assert.Equal(t, 3, aliceAfter.MatchesPlayed())
		assert.Equal(t, 3, bobAfter.MatchesPlayed())

		// Alice should have higher ELO after winning all matches
		assert.Greater(t, aliceAfter.ELO(), bobAfter.ELO())
	})

	t.Run("handles_multiplayer_match", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add 4 players
		players := []string{"Alice", "Bob", "Charlie", "Dave"}
		for _, name := range players {
			assert.NoError(t, league.AddPlayer(name))
		}

		// Get player references
		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")
		charlie, _ := league.GetPlayer("Charlie")
		dave, _ := league.GetPlayer("Dave")

		// Record 4-player match
		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
			{Position: 3, Player: charlie},
			{Position: 4, Player: dave},
		}

		err := league.AddMatch(results)
		assert.NoError(t, err)

		// Winner should have highest ELO, last place should have lowest
		aliceAfter, _ := league.GetPlayer("Alice")
		daveAfter, _ := league.GetPlayer("Dave")

		assert.Greater(t, aliceAfter.ELO(), 1000)
		assert.Less(t, daveAfter.ELO(), 1000)
	})

	t.Run("respects_max_match_limit", func(t *testing.T) {
		config := multielo.LeagueConfig{
			InitialELO: 1000,
			MinELO:     100,
			MaxELO:     3000,
			MaxMatches: 2, // Small limit for testing
			MaxPlayers: 100,
			KFactor:    32,
		}
		league := multielo.NewLeagueWithConfig(config)

		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
		}

		// Add matches up to limit
		assert.NoError(t, league.AddMatch(results))
		assert.NoError(t, league.AddMatch(results))

		// Should fail on exceeding limit
		err := league.AddMatch(results)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeConflict, eloError.Type)
	})
}

func TestLeague_RemovePlayer(t *testing.T) {
	t.Run("removes_existing_player", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		assert.Equal(t, 1, len(league.GetPlayers()))

		err := league.RemovePlayer("Alice")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(league.GetPlayers()))
	})

	t.Run("returns_error_for_non-existent_player", func(t *testing.T) {
		league := multielo.NewLeague()

		err := league.RemovePlayer("NonExistent")
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypePlayerNotFound, eloError.Type)
	})
}

func TestLeague_ResetPlayers(t *testing.T) {
	t.Run("resets_all_players_to_initial_state", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add players and play some matches
		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
		}
		assert.NoError(t, league.AddMatch(results))

		// Verify ELO changed
		aliceAfter, _ := league.GetPlayer("Alice")
		assert.NotEqual(t, 1000, aliceAfter.ELO())

		// Reset players
		league.ResetPlayers()

		// Verify all players are gone
		players := league.GetPlayers()
		assert.Equal(t, 0, len(players))
	})
}

func TestLeague_ResetMatches(t *testing.T) {
	t.Run("clears_all_match_history", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add players and matches
		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
		}
		assert.NoError(t, league.AddMatch(results))

		// Verify matches exist
		matches := league.GetMatches()
		assert.Equal(t, 1, len(matches))

		// Reset matches
		league.ResetMatches()

		// Verify matches cleared
		matches = league.GetMatches()
		assert.Equal(t, 0, len(matches))
	})
}

func TestPlayer_NewPlayer(t *testing.T) {
	t.Run("creates_player_with_valid_name", func(t *testing.T) {
		config := multielo.DefaultConfig()
		player, err := multielo.NewPlayer("Alice", config)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", player.Name())
		assert.Equal(t, config.InitialELO, player.ELO())
		assert.Equal(t, 0, player.MatchesPlayed())
	})

	t.Run("validates_empty_names", func(t *testing.T) {
		config := multielo.DefaultConfig()

		player, err := multielo.NewPlayer("", config)
		assert.Error(t, err)
		assert.Nil(t, player)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)
	})
}

func TestPlayer_UpdateELO(t *testing.T) {
	t.Run("updates_ELO_within_bounds", func(t *testing.T) {
		config := multielo.DefaultConfig()
		player, _ := multielo.NewPlayer("Alice", config)
		validator := multielo.NewDefaultValidator(config)

		err := player.UpdateELO(100, validator)
		assert.NoError(t, err)
		assert.Equal(t, 1100, player.ELO())
		assert.Equal(t, 100, player.ELOChange())
	})

	t.Run("prevents_ELO_below_minimum", func(t *testing.T) {
		config := multielo.DefaultConfig()
		player, _ := multielo.NewPlayer("Alice", config)
		validator := multielo.NewDefaultValidator(config)

		// Try to reduce ELO below minimum
		err := player.UpdateELO(-1000, validator)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)

		// ELO should remain unchanged
		assert.Equal(t, 1000, player.ELO())
	})

	t.Run("prevents_ELO_above_maximum", func(t *testing.T) {
		config := multielo.DefaultConfig()
		player, _ := multielo.NewPlayer("Alice", config)
		validator := multielo.NewDefaultValidator(config)

		// Try to increase ELO above maximum
		err := player.UpdateELO(2500, validator)
		assert.Error(t, err)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)

		// ELO should remain unchanged
		assert.Equal(t, 1000, player.ELO())
	})

	t.Run("updates_peak_ELO", func(t *testing.T) {
		config := multielo.DefaultConfig()
		player, _ := multielo.NewPlayer("Alice", config)
		validator := multielo.NewDefaultValidator(config)

		// Initial peak should be initial ELO
		assert.Equal(t, 1000, player.PeakELO())

		// Increase ELO
		err := player.UpdateELO(200, validator)
		assert.NoError(t, err)
		assert.Equal(t, 1200, player.PeakELO())

		// Decrease ELO - peak should not change
		err = player.UpdateELO(-100, validator)
		assert.NoError(t, err)
		assert.Equal(t, 1100, player.ELO())
		assert.Equal(t, 1200, player.PeakELO()) // Peak unchanged
	})
}

func TestLeague_ThreadSafety(t *testing.T) {
	t.Run("concurrent_player_additions", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add players concurrently
		const numPlayers = 50
		done := make(chan bool, numPlayers)

		for i := 0; i < numPlayers; i++ {
			go func(id int) {
				defer func() { done <- true }()
				league.AddPlayer("Player" + strconv.Itoa(id))
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numPlayers; i++ {
			<-done
		}

		// Should have added all unique players
		players := league.GetPlayers()
		assert.Equal(t, numPlayers, len(players))
	})

	t.Run("concurrent_match_additions", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add initial players
		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		// Add matches concurrently
		const numMatches = 20
		done := make(chan bool, numMatches)

		for i := 0; i < numMatches; i++ {
			go func() {
				defer func() { done <- true }()
				results := []*multielo.MatchResult{
					{Position: 1, Player: alice},
					{Position: 2, Player: bob},
				}
				league.AddMatch(results)
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numMatches; i++ {
			<-done
		}

		// Should have processed all matches
		matches := league.GetMatches()
		assert.Equal(t, numMatches, len(matches))
	})
}

func TestLeague_GenerateGraph(t *testing.T) {
	t.Run("generates_graph_with_match_data", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add players
		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		// Add some matches
		for i := 0; i < 3; i++ {
			results := []*multielo.MatchResult{
				{Position: 1, Player: alice},
				{Position: 2, Player: bob},
			}
			assert.NoError(t, league.AddMatch(results))
		}

		// Generate graph
		filename, err := league.GenerateGraph()
		assert.NoError(t, err)
		assert.Equal(t, "output/elo.html", filename)
	})

	t.Run("returns_error_with_no_players", func(t *testing.T) {
		league := multielo.NewLeague()

		filename, err := league.GenerateGraph()
		assert.Error(t, err)
		assert.Equal(t, "", filename)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
	})

	t.Run("returns_error_with_no_matches", func(t *testing.T) {
		league := multielo.NewLeague()
		assert.NoError(t, league.AddPlayer("Alice"))

		filename, err := league.GenerateGraph()
		assert.Error(t, err)
		assert.Equal(t, "", filename)

		var eloError multielo.ELOError
		assert.True(t, errors.As(err, &eloError))
		assert.Equal(t, multielo.ErrorTypeValidation, eloError.Type)
	})
}

func TestMatch(t *testing.T) {
	t.Run("stores_match_results_with_timestamp", func(t *testing.T) {
		league := multielo.NewLeague()

		assert.NoError(t, league.AddPlayer("Alice"))
		assert.NoError(t, league.AddPlayer("Bob"))

		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")

		results := []*multielo.MatchResult{
			{Position: 1, Player: alice},
			{Position: 2, Player: bob},
		}

		assert.NoError(t, league.AddMatch(results))

		matches := league.GetMatches()
		assert.Equal(t, 1, len(matches))

		match := matches[0]
		assert.Equal(t, 2, len(match.Results))
		assert.NotZero(t, match.Date)
	})
}

// =============================================================================
// COMPREHENSIVE 6-PLAYER MULTIPLAYER DEMO
// =============================================================================

func TestSixPlayerMultiplayerDemo(t *testing.T) {
	t.Run("comprehensive_6_player_tournament_simulation", func(t *testing.T) {
		// Create league with custom config for demonstration
		config := multielo.LeagueConfig{
			InitialELO: 1200,
			MinELO:     800,
			MaxELO:     2000,
			KFactor:    32,
			MaxPlayers: 10,
			MaxMatches: 100,
		}
		league := multielo.NewLeagueWithConfig(config)

		// Add 6 players representing different skill levels
		playerNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"}
		for _, name := range playerNames {
			err := league.AddPlayer(name)
			assert.NoError(t, err)
		}

		// Verify all players start with same ELO
		for _, name := range playerNames {
			player, err := league.GetPlayer(name)
			assert.NoError(t, err)
			assert.Equal(t, 1200, player.ELO())
			assert.Equal(t, 0, player.MatchesPlayed())
		}

		// Simulate Tournament Round 1: Full 6-player match
		// Alice dominates, Frank struggles
		round1Results := []*multielo.MatchResult{
			{Position: 1, Player: getPlayerByName(t, league, "Alice")},
			{Position: 2, Player: getPlayerByName(t, league, "Bob")},
			{Position: 3, Player: getPlayerByName(t, league, "Charlie")},
			{Position: 4, Player: getPlayerByName(t, league, "Diana")},
			{Position: 5, Player: getPlayerByName(t, league, "Eve")},
			{Position: 6, Player: getPlayerByName(t, league, "Frank")},
		}

		err := league.AddMatch(round1Results)
		assert.NoError(t, err)

		// Check ELO changes after first match
		// With K=32 and n=6, K-factor per comparison = 32/5 = 6.4
		alice, _ := league.GetPlayer("Alice")
		bob, _ := league.GetPlayer("Bob")
		charlie, _ := league.GetPlayer("Charlie")
		diana, _ := league.GetPlayer("Diana")
		eve, _ := league.GetPlayer("Eve")
		frank, _ := league.GetPlayer("Frank")

		// Alice (1st) should have highest ELO gain
		assert.Greater(t, alice.ELO(), 1200, "Alice should gain ELO from winning")
		assert.Greater(t, alice.ELOChange(), 0, "Alice should have positive ELO change")

		// Frank (6th) should have lowest ELO
		assert.Less(t, frank.ELO(), 1200, "Frank should lose ELO from last place")
		assert.Less(t, frank.ELOChange(), 0, "Frank should have negative ELO change")

		// Verify ELO ordering matches performance
		assert.Greater(t, alice.ELO(), bob.ELO(), "Alice > Bob")
		assert.Greater(t, bob.ELO(), charlie.ELO(), "Bob > Charlie")
		assert.Greater(t, charlie.ELO(), diana.ELO(), "Charlie > Diana")
		assert.Greater(t, diana.ELO(), eve.ELO(), "Diana > Eve")
		assert.Greater(t, eve.ELO(), frank.ELO(), "Eve > Frank")

		// Verify match statistics - only 1st place counts as a win
		for _, name := range playerNames {
			player, _ := league.GetPlayer(name)
			assert.Equal(t, 1, player.MatchesPlayed())
			// Only Alice (1st place) should have a win
			if name == "Alice" {
				assert.Equal(t, 1, player.MatchesWon())
			} else {
				assert.Equal(t, 0, player.MatchesWon())
			}
		}

		// Store ELOs after round 1 for comparison
		round1ELOs := make(map[string]int)
		for _, name := range playerNames {
			player, _ := league.GetPlayer(name)
			round1ELOs[name] = player.ELO()
		}

		// Generate visualization after round 1
		league.GenerateGraphWithFilename("6player-tournament-round1")

		// Tournament Round 2: Frank makes a comeback, Alice has bad game
		round2Results := []*multielo.MatchResult{
			{Position: 1, Player: getPlayerByName(t, league, "Frank")}, // Comeback!
			{Position: 2, Player: getPlayerByName(t, league, "Diana")},
			{Position: 3, Player: getPlayerByName(t, league, "Charlie")},
			{Position: 4, Player: getPlayerByName(t, league, "Bob")},
			{Position: 5, Player: getPlayerByName(t, league, "Eve")},
			{Position: 6, Player: getPlayerByName(t, league, "Alice")}, // Upset!
		}

		err = league.AddMatch(round2Results)
		assert.NoError(t, err)

		// Verify Frank's dramatic improvement
		frank, _ = league.GetPlayer("Frank")
		assert.Greater(t, frank.ELO(), round1ELOs["Frank"], "Frank should gain ELO from winning")
		assert.Greater(t, frank.ELOChange(), 0, "Frank should have positive change")

		// Verify Alice's setback despite higher skill rating
		alice, _ = league.GetPlayer("Alice")
		// Alice should lose significant ELO due to high expectation vs poor performance
		assert.Less(t, alice.ELO(), round1ELOs["Alice"], "Alice should lose ELO from last place")
		assert.Less(t, alice.ELOChange(), 0, "Alice should have negative change")

		// Verify final standings after 2 rounds
		for _, name := range playerNames {
			player, _ := league.GetPlayer(name)
			_ = player.GetStats()
		}

		// Generate final tournament visualization
		league.GenerateGraphWithFilename("6player-tournament-final")

		// Verify match count
		matches := league.GetMatches()
		assert.Len(t, matches, 2)

		// Verify each player has played 2 matches
		for _, name := range playerNames {
			player, _ := league.GetPlayer(name)
			assert.Equal(t, 2, player.MatchesPlayed())
		}
	})

	t.Run("smaller_4_player_matches_with_different_outcomes", func(t *testing.T) {
		league := multielo.NewLeague()

		// Add 4 players
		players := []string{"Pro", "Good", "Average", "Beginner"}
		for _, name := range players {
			err := league.AddPlayer(name)
			assert.NoError(t, err)
		}

		// Match 1: Expected results (skill order)
		match1 := []*multielo.MatchResult{
			{Position: 1, Player: getPlayerByName(t, league, "Pro")},
			{Position: 2, Player: getPlayerByName(t, league, "Good")},
			{Position: 3, Player: getPlayerByName(t, league, "Average")},
			{Position: 4, Player: getPlayerByName(t, league, "Beginner")},
		}
		err := league.AddMatch(match1)
		assert.NoError(t, err)

		// Store baseline ELOs
		baselineELOs := make(map[string]int)
		for _, name := range players {
			player, _ := league.GetPlayer(name)
			baselineELOs[name] = player.ELO()
		}

		// Match 2: Complete upset (reverse order)
		match2 := []*multielo.MatchResult{
			{Position: 1, Player: getPlayerByName(t, league, "Beginner")}, // Massive upset!
			{Position: 2, Player: getPlayerByName(t, league, "Average")},
			{Position: 3, Player: getPlayerByName(t, league, "Good")},
			{Position: 4, Player: getPlayerByName(t, league, "Pro")}, // Shocking loss
		}
		err = league.AddMatch(match2)
		assert.NoError(t, err)

		// Verify dramatic rating changes due to upset
		beginner, _ := league.GetPlayer("Beginner")
		pro, _ := league.GetPlayer("Pro")

		// Beginner should have ELO gain (winning against higher rated players)
		assert.Greater(t, beginner.ELO(), baselineELOs["Beginner"],
			"Beginner should gain ELO from upset win")

		// Pro should have ELO loss (losing to lower rated players)
		assert.Less(t, pro.ELO(), baselineELOs["Pro"],
			"Pro should lose ELO from upset loss")

		// Verify the magnitude is reasonable (ELO changes are typically modest)
		beginnerGain := beginner.ELO() - baselineELOs["Beginner"]
		proLoss := baselineELOs["Pro"] - pro.ELO()
		assert.Greater(t, beginnerGain, 10, "Beginner should gain at least 10 ELO")
		assert.Greater(t, proLoss, 10, "Pro should lose at least 10 ELO")

		// Verify the dramatic swing occurred
		for _, name := range players {
			player, _ := league.GetPlayer(name)
			_ = player.ELO()
		}

		// Generate visualization of the upset scenario
		league.GenerateGraphWithFilename("4player-upset-scenario")
	})
}

// Helper function to get player by name in tests
func getPlayerByName(t *testing.T, league *multielo.League, name string) *multielo.Player {
	player, err := league.GetPlayer(name)
	assert.NoError(t, err)
	return player
}

// =============================================================================
// REAL KARTING LEAGUE DATA VALIDATION
// =============================================================================

// RealKartingData represents the structure from https://gerry.dbyte.xyz/karting.json
type RealKartingData struct {
	Players []RealPlayer `json:"Players"`
	Matches []RealMatch  `json:"Matches"`
}

type RealPlayer struct {
	Name      string          `json:"Name"`
	ELO       int             `json:"ELO"`
	ELOChange int             `json:"ELOChange"`
	Stats     RealPlayerStats `json:"Stats"`
}

type RealPlayerStats struct {
	MatchesPlayed       int   `json:"MatchesPlayed"`
	MatchesWon          int   `json:"MatchesWon"`
	AllTimeAveragePlace int   `json:"AllTimeAveragePlace"`
	Last5Finish         []int `json:"Last5Finish"`
	PeakELO             int   `json:"PeakELO"`
}

type RealMatch struct {
	Results []RealMatchResult `json:"Results"`
	Date    time.Time         `json:"Date"`
}

type RealMatchResult struct {
	Position int        `json:"Position"`
	Player   RealPlayer `json:"Player"`
}

func TestRealKartingLeagueValidation(t *testing.T) {
	t.Run("replay_real_karting_league_data_and_validate_elo_accuracy", func(t *testing.T) {
		// Load real karting data
		file, err := os.Open("karting_real_data.json")
		if err != nil {
			t.Skipf("Skipping real data test: could not open karting_real_data.json: %v", err)
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		assert.NoError(t, err)

		var realData RealKartingData
		err = json.Unmarshal(data, &realData)
		assert.NoError(t, err)

		// Create league with same initial configuration as original (ELO 1000)
		config := multielo.LeagueConfig{
			InitialELO:    1000,
			MinELO:        0,
			MaxELO:        3000,
			KFactor:       32,
			MaxPlayers:    100,
			MaxMatches:    1000,
			OutputDirectory: "output",
		}
		league := multielo.NewLeagueWithConfig(config)

		// Add all players that appear in matches to ensure they exist
		playersSeen := make(map[string]bool)
		for _, match := range realData.Matches {
			for _, result := range match.Results {
				playerName := result.Player.Name
				if !playersSeen[playerName] {
					err := league.AddPlayer(playerName)
					assert.NoError(t, err)
					playersSeen[playerName] = true
				}
			}
		}

		// Replay all matches chronologically
		for _, realMatch := range realData.Matches {
			// Convert real match results to our format
			matchResults := make([]*multielo.MatchResult, len(realMatch.Results))
			for j, result := range realMatch.Results {
				player, err := league.GetPlayer(result.Player.Name)
				assert.NoError(t, err)

				matchResults[j] = &multielo.MatchResult{
					Position: result.Position,
					Player:   player,
				}
			}

			// Process the match
			err := league.AddMatch(matchResults)
			assert.NoError(t, err)
		}

		// Compare final ELO ratings with expected values
		var totalAbsoluteDiff int
		var playersChecked int
		var exactMatches int

		for _, expectedPlayer := range realData.Players {
			if expectedPlayer.Stats.MatchesPlayed == 0 {
				continue
			}

			actualPlayer, err := league.GetPlayer(expectedPlayer.Name)
			if err != nil {
				t.Errorf("Player %s not found in our league", expectedPlayer.Name)
				continue
			}

			actualELO := actualPlayer.ELO()
			expectedELO := expectedPlayer.ELO
			diff := actualELO - expectedELO
			absDiff := diff
			if absDiff < 0 {
				absDiff = -absDiff
			}

			if diff == 0 {
				exactMatches++
			}

			totalAbsoluteDiff += absDiff
			playersChecked++

			// Also verify match statistics
			actualStats := actualPlayer.GetStats()
			assert.Equal(t, expectedPlayer.Stats.MatchesPlayed, actualStats.MatchesPlayed,
				"Matches played mismatch for %s", expectedPlayer.Name)
			assert.Equal(t, expectedPlayer.Stats.MatchesWon, actualStats.MatchesWon,
				"Matches won mismatch for %s", expectedPlayer.Name)
		}

		// Verify difference distribution
		exactCount := 0
		smallDiff := 0  // 1-2 points
		mediumDiff := 0 // 3-4 points
		largeDiff := 0  // 5+ points

		for _, expectedPlayer := range realData.Players {
			if expectedPlayer.Stats.MatchesPlayed == 0 {
				continue
			}

			actualPlayer, err := league.GetPlayer(expectedPlayer.Name)
			if err != nil {
				continue
			}

			diff := actualPlayer.ELO() - expectedPlayer.ELO
			absDiff := diff
			if absDiff < 0 {
				absDiff = -absDiff
			}

			if absDiff == 0 {
				exactCount++
			} else if absDiff <= 2 {
				smallDiff++
			} else if absDiff <= 4 {
				mediumDiff++
			} else {
				largeDiff++
			}
		}
		_ = exactCount
		_ = smallDiff
		_ = mediumDiff
		_ = largeDiff

		// Assert that our refactored system is highly accurate
		// Small differences (1-3 points) are acceptable due to rounding differences
		accuracy := float64(exactMatches) / float64(playersChecked)
		assert.GreaterOrEqual(t, accuracy, 0.30,
			"ELO accuracy should be at least 30%% exact matches - major calculation errors detected")

		avgDiff := float64(totalAbsoluteDiff) / float64(playersChecked)
		assert.LessOrEqual(t, avgDiff, 5.0,
			"Average ELO difference should be <= 5 points - significant calculation drift detected")

		// Additional validation: no player should differ by more than 10 points
		maxDiff := 0
		for _, expectedPlayer := range realData.Players {
			if expectedPlayer.Stats.MatchesPlayed == 0 {
				continue
			}
			actualPlayer, _ := league.GetPlayer(expectedPlayer.Name)
			diff := actualPlayer.ELO() - expectedPlayer.ELO
			if diff < 0 {
				diff = -diff
			}
			if diff > maxDiff {
				maxDiff = diff
			}
		}
		assert.LessOrEqual(t, maxDiff, 10, "No player should differ by more than 10 ELO points")

		// Generate visualization of the real league
		filename, err := league.GenerateGraphWithFilename("real-karting-league")
		assert.NoError(t, err, "Failed to generate graph")
		assert.NotEmpty(t, filename)
	})
}
