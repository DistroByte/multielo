package multielo

import (
	"context"
	"testing"
	"time"
)

type testLogger struct {
	infoCalls  int
	errorCalls int
}

func (l *testLogger) Info(msg string, fields map[string]interface{})             { l.infoCalls++ }
func (l *testLogger) Error(msg string, err error, fields map[string]interface{}) { l.errorCalls++ }
func (l *testLogger) Debug(msg string, fields map[string]interface{})            {}

type testMetrics struct {
	matchesAdded int
	playersAdded int
}

func (m *testMetrics) RecordMatchAdded(playerCount int)                { m.matchesAdded++ }
func (m *testMetrics) RecordPlayerAdded()                              { m.playersAdded++ }
func (m *testMetrics) RecordELOChange(playerName string, delta int)    {}
func (m *testMetrics) RecordError(operation string, errType ErrorType) {}

func TestObservabilityHooks(t *testing.T) {
	logger, metrics := &testLogger{}, &testMetrics{}
	l := NewLeagueWithDependencies(DefaultConfig(), LeagueDependencies{Logger: logger, Metrics: metrics})
	_ = l.AddPlayer("observer")
	if logger.infoCalls != 1 || metrics.playersAdded != 1 {
		t.Fatalf("expected 1 log+metric, got logs=%d metrics=%d", logger.infoCalls, metrics.playersAdded)
	}
	p, _ := l.GetPlayer("observer")
	_ = l.AddMatch([]*MatchResult{{Position: 1, Player: p}, {Position: 2, Player: p}})
	if metrics.matchesAdded != 1 {
		t.Fatalf("expected 1 match metric, got %d", metrics.matchesAdded)
	}
}

func TestArchiveCallback(t *testing.T) {
	count := 0
	cfg := DefaultConfig()
	cfg.MaxMatches, cfg.AutoArchive = 2, true
	l := NewLeagueWithDependencies(cfg, LeagueDependencies{
		ArchiveCallback: func(ctx context.Context, m []*Match) error { count += len(m); return nil },
	})
	_ = l.AddPlayer("a")
	_ = l.AddPlayer("b")
	a, _ := l.GetPlayer("a")
	b, _ := l.GetPlayer("b")
	for i := 0; i < 3; i++ {
		_ = l.AddMatch([]*MatchResult{{Position: 1, Player: a}, {Position: 2, Player: b}})
	}
	if count != 1 {
		t.Fatalf("expected 1 archived, got %d", count)
	}
}

func TestMatchFiltering(t *testing.T) {
	l := NewLeague()
	_ = l.AddPlayer("x")
	_ = l.AddPlayer("y")
	_ = l.AddPlayer("z")
	x, _ := l.GetPlayer("x")
	y, _ := l.GetPlayer("y")
	z, _ := l.GetPlayer("z")
	_ = l.AddMatch([]*MatchResult{{Position: 1, Player: x}, {Position: 2, Player: y}})
	time.Sleep(time.Millisecond)
	_ = l.AddMatch([]*MatchResult{{Position: 1, Player: z}, {Position: 2, Player: y}, {Position: 3, Player: x}})

	if r := l.GetMatchesFiltered(MatchFilter{PlayerName: "x"}); r.Total != 2 {
		t.Fatalf("expected 2 matches, got %d", r.Total)
	}
	if r := l.GetMatchesFiltered(MatchFilter{MinPlayers: 3}); r.Total != 1 {
		t.Fatalf("expected 1 match, got %d", r.Total)
	}
	if r := l.GetMatchesFiltered(MatchFilter{Limit: 1}); len(r.Matches) != 1 || !r.HasMore {
		t.Fatal("expected 1 match with hasMore=true")
	}
}

func TestMultiLeagueService(t *testing.T) {
	svc := NewMultiLeagueService(DefaultConfig(), LeagueDependencies{})
	ctx := context.Background()

	_ = svc.CreateLeague(ctx, "league1")
	if err := svc.CreateLeague(ctx, "league1"); err == nil {
		t.Fatal("expected duplicate error")
	}
	_ = svc.AddPlayer(ctx, "league1", "alpha")
	if err := svc.AddPlayer(ctx, "nonexistent", "beta"); err == nil {
		t.Fatal("expected not found error")
	}
	if ids := svc.ListLeagueIDs(ctx); len(ids) != 1 {
		t.Fatalf("expected 1 league, got %v", ids)
	}
	if board, _ := svc.GetLeaderboard(ctx, "league1"); len(board) != 1 {
		t.Fatalf("expected 1 player, got %d", len(board))
	}
	_ = svc.DeleteLeague(ctx, "league1")
	if _, err := svc.GetLeague(ctx, "league1"); err == nil {
		t.Fatal("expected error after delete")
	}
}
