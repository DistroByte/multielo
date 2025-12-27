package multielo

import "testing"

type stubCalculator struct{ calls int }

func (s *stubCalculator) Calculate(results []*MatchResult, cfg LeagueConfig) ([]MatchDiff, error) {
	s.calls++
	diffs := make([]MatchDiff, len(results))
	for i, r := range results {
		diffs[i] = MatchDiff{Player: r.Player, Diff: 0}
	}
	return diffs, nil
}

func TestNewLeagueWithDependenciesUsesInjectedCalculator(t *testing.T) {
	calc := &stubCalculator{}
	l := NewLeagueWithDependencies(DefaultConfig(), LeagueDependencies{Calculator: calc})
	_ = l.AddPlayer("alpha")
	_ = l.AddPlayer("beta")
	alpha, _ := l.GetPlayer("alpha")
	beta, _ := l.GetPlayer("beta")
	_ = l.AddMatch([]*MatchResult{{Position: 1, Player: alpha}, {Position: 2, Player: beta}})
	if calc.calls != 1 {
		t.Fatalf("expected 1 calculator call, got %d", calc.calls)
	}
}

func TestLeagueServiceProvidesCommandAndQuerySeparation(t *testing.T) {
	svc := NewLeagueService(nil)
	_ = svc.AddPlayer("solo")
	_ = svc.AddPlayer("duo")
	p1, _ := svc.Player("solo")
	p2, _ := svc.Player("duo")
	_ = svc.RecordMatch([]*MatchResult{{Position: 1, Player: p1}, {Position: 2, Player: p2}})
	if len(svc.Players()) != 2 {
		t.Fatal("expected 2 players")
	}
	if svc.League() == nil {
		t.Fatal("expected league exposed")
	}
}
