package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/source"
)

func (m Model) isCurrentTrack(t source.Track) bool {
	if m.track == nil {
		return false
	}
	return t.ID != "" && m.track.ID != "" && t.ID == m.track.ID || t.URI != "" && m.track.URI != "" && t.URI == m.track.URI
}

// Playback and keyboard selection are separate states: playback has a
// persistent accent marker; selection has a neutral surface and > cursor.
func (m Model) trackLine(r row, w int, selected, playing bool, number int) string {
	marker := fmt.Sprintf("%02d ", number)
	if selected {
		marker = ">  "
	}
	if playing {
		marker = "▶  "
		if m.track != nil && !m.track.Playing {
			marker = "Ⅱ  "
		}
		if selected {
			marker = ">▶ "
			if m.track != nil && !m.track.Playing {
				marker = ">Ⅱ "
			}
		}
	}
	right := duration(int64(r.track.Duration.Seconds()))
	if r.albumID != "" {
		right = "album"
	}
	nameW := max(1, w-4-len(right))
	name := label(r.track.Name, nameW)
	line := marker + fit(name, nameW) + " " + right
	style := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorBg)
	if selected {
		style = style.Background(app.ColorSurface)
	}
	if playing {
		style = style.Foreground(app.ColorAccent).Bold(true)
	}
	return style.Render(line)
}
