package infrastructure

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/distrobyte/multielo/domain"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

type graphRenderer struct{}

func NewGraphRenderer() domain.GraphRenderer { return &graphRenderer{} }

var colors = []color.Color{
	color.RGBA{R: 255, A: 255},
	color.RGBA{G: 255, A: 255},
	color.RGBA{B: 255, A: 255},
	color.RGBA{R: 255, G: 255, A: 255},
	color.RGBA{R: 255, B: 255, A: 255},
	color.RGBA{G: 255, B: 255, A: 255},
}

func (r *graphRenderer) Render(data domain.GraphData, filenamePrefix string) (string, error) {
	p := plot.New()
	p.Title.Text = "ELO over time"
	p.X.Label.Text = "Races"
	p.Y.Label.Text = "ELO"
	p.Add(plotter.NewGrid())

	p.X.Tick.Marker = RaceTicker{}
	p.Y.Tick.Marker = ELOTicker{}

	players := make([]domain.GraphPlayer, len(data.Players))
	copy(players, data.Players)
	sort.Slice(players, func(i, j int) bool { return players[i].ELO > players[j].ELO })

	minELO, maxELO := players[len(players)-1].ELO, players[0].ELO
	p.Y.Min = float64(minELO - 50)
	p.Y.Max = float64(maxELO + 50)
	p.X.Min = 0

	playerELOHistory := make(map[string][]int)
	for _, player := range players {
		playerELOHistory[player.Name] = player.ELOHistory
	}

	for j, player := range players {
		history := playerELOHistory[player.Name]
		xys := make(plotter.XYs, len(history))
		for i, elo := range history {
			xys[i] = plotter.XY{X: float64(i), Y: float64(elo)}
		}
		line, points, err := plotter.NewLinePoints(xys)
		if err != nil {
			return "", fmt.Errorf("failed to create line plot: %w", err)
		}
		line.Color = colors[j%len(colors)]
		points.Shape = draw.CircleGlyph{}
		points.Color = colors[j%len(colors)]
		p.Add(line, points)
		p.Legend.Add(fmt.Sprintf("%s (%d)", player.Name, player.ELO), line)
	}

	outputDir := data.Config.OutputDirectory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	svgFilename := filepath.Join(outputDir, filenamePrefix+".svg")
	pngFilename := filepath.Join(outputDir, filenamePrefix+".png")
	if err := p.Save(30*vg.Centimeter, 20*vg.Centimeter, svgFilename); err != nil {
		return "", fmt.Errorf("failed to save SVG: %w", err)
	}
	if err := p.Save(30*vg.Centimeter, 20*vg.Centimeter, pngFilename); err != nil {
		return "", fmt.Errorf("failed to save PNG: %w", err)
	}
	return pngFilename, nil
}

type RaceTicker struct{}
type ELOTicker struct{}

func (t RaceTicker) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for i := min + 1; i < max; i += 3 {
		ticks = append(ticks, plot.Tick{Value: i, Label: strconv.Itoa(int(i))})
	}
	return ticks
}

func (t ELOTicker) Ticks(min, max float64) []plot.Tick {
	var ticks []plot.Tick
	for i := min; i < max; i += 10 {
		ticks = append(ticks, plot.Tick{Value: i, Label: strconv.Itoa(int(i))})
	}
	return ticks
}
