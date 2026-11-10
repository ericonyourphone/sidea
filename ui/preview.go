package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/source"
)

// DemoPreview renders a deterministic demo frame without a terminal, audio,
// timers, or network access when used with the built-in demo source.
func DemoPreview(src source.RichSource, width, height int, query ...string) string {
	m := NewModel(src, app.Options{}).WithDemo()
	defer m.cancel()
	m.width, m.height = width, height
	for _, cmd := range []tea.Cmd{m.fetchState(), m.fetchLibrary()} {
		result, _ := m.Update(cmd())
		m = result.(Model)
	}
	result, _ := m.Update(m.fetchQueue()())
	m = result.(Model)
	m.frame = 80
	m.animate(time.Now())
	if len(query) > 0 && query[0] != "" {
		m.openSearch()
		m.search.SetValue(query[0])
		if cmd := m.submitSearch(); cmd != nil {
			result, _ := m.Update(cmd())
			m = result.(Model)
		}
	}
	return m.View()
}
