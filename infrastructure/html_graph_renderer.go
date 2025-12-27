package infrastructure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/distrobyte/multielo/domain"
)

type htmlGraphRenderer struct{}

func NewHTMLGraphRenderer() domain.GraphRenderer {
	return &htmlGraphRenderer{}
}

var htmlColors = []string{
	"#FF0000", "#0000FF", "#00AA00", "#FF8800", "#FF00FF",
	"#00FFFF", "#FFFF00", "#AA0000", "#0000AA", "#00AA99",
	"#FF00AA", "#AAFF00", "#FF6600", "#0066FF", "#00FF66",
	"#FF0066", "#6600FF", "#66FF00", "#0066AA", "#AA6600",
}

func (r *htmlGraphRenderer) Render(data domain.GraphData, filenamePrefix string) (string, error) {
	outputDir := data.Config.OutputDirectory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build player data with ELO history
	players := make([]domain.GraphPlayer, len(data.Players))
	copy(players, data.Players)
	sort.Slice(players, func(i, j int) bool { return players[i].ELO > players[j].ELO })

	playerELOHistory := make(map[string][]int)
	for _, player := range players {
		playerELOHistory[player.Name] = player.ELOHistory
	}

	playerParticipation := make(map[string]map[int]bool)
	playerPositions := make(map[string]map[int]int)
	for playerName := range playerELOHistory {
		playerParticipation[playerName] = make(map[int]bool)
		playerPositions[playerName] = make(map[int]int)
	}
	for raceIndex, match := range data.Matches {
		for _, result := range match.Results {
			playerParticipation[result.Name][raceIndex] = true
			playerPositions[result.Name][raceIndex] = result.Position
		}
	}

	html := generateHTML(players, playerELOHistory, playerParticipation, playerPositions, htmlColors)

	htmlFilename := filepath.Join(outputDir, filenamePrefix+".html")
	if err := os.WriteFile(htmlFilename, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("failed to write HTML file: %w", err)
	}

	return htmlFilename, nil
}

func generateHTML(players []domain.GraphPlayer, history map[string][]int, participation map[string]map[int]bool, positions map[string]map[int]int, colors []string) string {
	dashStyles := []string{"solid", "dot", "dash", "longdash", "dashdot", "longdashdot"}

	var traces []string
	playerList := make([]string, 0, len(players))
	for i, player := range players {
		playerList = append(playerList, player.Name)
		elos := history[player.Name]

		// Find the first race where player participated
		// Note: ELO history index 0 is the initial rating before the first match.
		// Participation is keyed by match index, so participation at match k maps to ELO index k+1.
		playerRaces := participation[player.Name]
		startIndex := 0
		for raceIndex := 1; raceIndex < len(elos); raceIndex++ {
			if playerRaces[raceIndex-1] {
				// Start from one race before first participation, or 0 if first participation is at race 1
				startIndex = raceIndex - 1
				if startIndex < 0 {
					startIndex = 0
				}
				break
			}
		}

		color := colors[i%len(colors)]
		dash := dashStyles[i%len(dashStyles)]

		// Build small line segments to allow per-miss fading
		// Keep a single legend entry by showing it on the first segment only
		showLegend := true
		consecutiveMisses := 0
		prevIndex := startIndex
		for raceIndex := startIndex + 1; raceIndex < len(elos); raceIndex++ {
			// Segment ending at ELO index raceIndex corresponds to match index raceIndex-1
			attended := playerRaces[raceIndex-1]
			if attended {
				consecutiveMisses = 0
			} else {
				consecutiveMisses++
			}

			// Compute opacity: full for attended, decreasing for consecutive misses
			opacity := 1.0
			if !attended {
				opacity = 0.85 - 0.2*float64(consecutiveMisses)
				if opacity < 0.1 {
					opacity = 0.1
				}
			}

			segX := fmt.Sprintf("%d, %d", prevIndex, raceIndex)
			segY := fmt.Sprintf("%d, %d", elos[prevIndex], elos[raceIndex])
			// Compute delta for this segment
			delta := elos[raceIndex] - elos[prevIndex]

			// Build trace; show hover only for decay segments
			var trace string
			if attended {
				trace = fmt.Sprintf(`{
	name: %q,
	legendgroup: %q,
	showlegend: %t,
	x: [%s],
	y: [%s],
	type: 'scatter',
	mode: 'lines',
	line: { color: %q, width: 2, dash: %q },
		hoverinfo: 'skip',
	opacity: %.2f,
		hovertemplate: '<b>%s</b><br>Rating: %%{y}<extra></extra>',
	visible: true
	}`, fmt.Sprintf("%s (%d)", player.Name, player.ELO), player.Name, showLegend, segX, segY, color, dash, opacity, player.Name)
			} else {
				trace = fmt.Sprintf(`{
		name: %q,
		legendgroup: %q,
		showlegend: %t,
		x: [%s],
		y: [%s],
		type: 'scatter',
		mode: 'lines',
		line: { color: %q, width: 2, dash: %q },
		opacity: %.2f,
		hovertemplate: '<b>%s</b><br>Rating: %%{y}<br>Decay: %+d<extra></extra>',
		visible: true
	}`, fmt.Sprintf("%s (%d)", player.Name, player.ELO), player.Name, showLegend, segX, segY, color, dash, opacity, player.Name, delta)
			}
			traces = append(traces, trace)

			// After first segment, hide further legend entries for this player
			if showLegend {
				showLegend = false
			}
			prevIndex = raceIndex
		}

		// Add circle markers only at attended races (no markers for missed)
		var attendedX []string
		var attendedY []string
		var attendedText []string
		for raceIndex := 1; raceIndex < len(elos); raceIndex++ {
			if playerRaces[raceIndex-1] {
				attendedX = append(attendedX, fmt.Sprintf("%d", raceIndex))
				attendedY = append(attendedY, fmt.Sprintf("%d", elos[raceIndex]))
				delta := elos[raceIndex] - elos[raceIndex-1]
				pos := positions[player.Name][raceIndex-1]
				attendedText = append(attendedText, fmt.Sprintf("Position: %d | Δ %+d", pos, delta))
			}
		}
		if len(attendedX) > 0 {
			// Quote text entries to produce a valid JS array of strings
			quoted := make([]string, len(attendedText))
			for i, t := range attendedText {
				quoted[i] = fmt.Sprintf("%q", t)
			}
			markerTrace := fmt.Sprintf(`{
	name: %q,
	legendgroup: %q,
	showlegend: false,
	x: [%s],
	y: [%s],
	type: 'scatter',
	mode: 'markers',
	marker: { color: %q, size: 6, symbol: 'circle' },
	text: [%s],
		    hovertemplate: '<b>%s</b><br>Rating: %%{y}<br>%s<extra></extra>',
	visible: true
  }`, fmt.Sprintf("%s markers", player.Name), player.Name, strings.Join(attendedX, ", "), strings.Join(attendedY, ", "), color, strings.Join(quoted, ", "), player.Name, "%{text}")
			traces = append(traces, markerTrace)
		}
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>ELO Progression</title>
  <script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
</head>
<body>
  <div id="eloChart" style="width:100%%;height:100vh;"></div>
  <script>
    const traces = [%s];
    const layout = {
      title: 'ELO Rating Progression',
      xaxis: { 
        title: 'Race',
        tickmode: 'linear',
        tick0: 0,
        dtick: 1
      },
      yaxis: { title: 'ELO' },
	hovermode: 'closest',
      hoverlabel: {
        namelength: -1,
        align: 'left'
      },
      margin: { l: 60, r: 60, t: 60, b: 100 }
    };
    Plotly.newPlot('eloChart', traces, layout, { responsive: true });
  </script>
</body>
</html>`,
		strings.Join(traces, ", "))

	return html
}
