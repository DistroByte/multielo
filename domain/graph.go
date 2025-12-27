package domain

type GraphRenderer interface {
	Render(data GraphData, filenamePrefix string) (string, error)
}

type GraphData struct {
	Config  LeagueConfig
	Players []GraphPlayer
	Matches []GraphMatch
}

type GraphPlayer struct {
	Name       string
	ELO        int
	ELOHistory []int
}

type GraphMatch struct {
	Results []GraphMatchResult
}

type GraphMatchResult struct {
	Position int
	Name     string
}

var graphRenderer GraphRenderer

func SetGraphRenderer(r GraphRenderer) {
	graphRenderer = r
}

func (l *League) GenerateGraph() (string, error) {
	return l.GenerateGraphWithFilename("elo")
}

func (l *League) GenerateGraphWithFilename(filenamePrefix string) (string, error) {
	l.mu.RLock()
	if l.playersRepo.Count() == 0 {
		l.mu.RUnlock()
		return "", ErrNoPlayers
	}
	if l.matchesRepo.Count() == 0 {
		l.mu.RUnlock()
		return "", ELOError{Type: ErrorTypeValidation, Message: "no matches to plot"}
	}

	data := GraphData{Config: l.config}

	players := l.playersRepo.List()
	data.Players = make([]GraphPlayer, 0, len(players))
	for _, p := range players {
		data.Players = append(data.Players, GraphPlayer{
			Name:       p.Name(),
			ELO:        p.ELO(),
			ELOHistory: l.playerELOHistory[p.Name()],
		})
	}

	matches := l.matchesRepo.List()
	data.Matches = make([]GraphMatch, 0, len(matches))
	for _, m := range matches {
		gm := GraphMatch{Results: make([]GraphMatchResult, 0, len(m.Results))}
		for _, r := range m.Results {
			gm.Results = append(gm.Results, GraphMatchResult{Position: r.Position, Name: r.Player.Name()})
		}
		data.Matches = append(data.Matches, gm)
	}
	l.mu.RUnlock()

	if graphRenderer == nil {
		return "", ELOError{Type: ErrorTypeInternal, Message: "graph renderer not configured"}
	}
	return graphRenderer.Render(data, filenamePrefix)
}
