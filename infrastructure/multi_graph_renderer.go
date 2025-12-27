package infrastructure

import (
	"path/filepath"
	"strings"

	"github.com/distrobyte/multielo/domain"
)

type multiRenderer struct{
	renderers []domain.GraphRenderer
}

func NewMultiGraphRenderer() domain.GraphRenderer {
	return &multiRenderer{renderers: []domain.GraphRenderer{
		NewHTMLGraphRenderer(),
		NewGraphRenderer(),
	}}
}

func (m *multiRenderer) Render(data domain.GraphData, filenamePrefix string) (string, error) {
	var primaryPath string
	for _, r := range m.renderers {
		p, err := r.Render(data, filenamePrefix)
		if err != nil {
			return "", err
		}
		if primaryPath == "" {
			primaryPath = p
		}
		// Prefer HTML path as primary if available
		if strings.EqualFold(filepath.Ext(p), ".html") {
			primaryPath = p
		}
	}
	return primaryPath, nil
}
