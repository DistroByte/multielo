package domain

import "time"

type MatchFilter struct {
	AfterDate  *time.Time
	BeforeDate *time.Time
	PlayerName string
	MinPlayers int
	MaxPlayers int
	Limit      int
	Offset     int
}

type MatchQueryResult struct {
	Matches []*Match
	Total   int
	HasMore bool
}

func (l *League) GetMatchesFiltered(filter MatchFilter) MatchQueryResult {
	l.mu.RLock()
	defer l.mu.RUnlock()

	allMatches := l.matchesRepo.List()
	filtered := make([]*Match, 0)

	for _, match := range allMatches {
		if filter.AfterDate != nil && match.Date.Before(*filter.AfterDate) {
			continue
		}
		if filter.BeforeDate != nil && match.Date.After(*filter.BeforeDate) {
			continue
		}
		if filter.PlayerName != "" {
			found := false
			for _, result := range match.Results {
				if result.Player.Name() == filter.PlayerName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		playerCount := len(match.Results)
		if filter.MinPlayers > 0 && playerCount < filter.MinPlayers {
			continue
		}
		if filter.MaxPlayers > 0 && playerCount > filter.MaxPlayers {
			continue
		}
		filtered = append(filtered, match)
	}

	total := len(filtered)
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = total - offset
	}

	end := offset + limit
	if end > total {
		end = total
	}

	paginated := filtered[offset:end]
	hasMore := end < total

	return MatchQueryResult{
		Matches: paginated,
		Total:   total,
		HasMore: hasMore,
	}
}
