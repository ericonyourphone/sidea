package app

import (
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ericonyourphone/sidea/spectrum"
)

type spectrumTickMsg time.Time

type spectrumView struct {
	levels, peaks [spectrum.Bands]float64
	status        string
}

func (m *Model) startSpectrumTick() tea.Cmd {
	if m.spectrumTicking || !m.spectrumVisible() {
		return nil
	}
	m.spectrumTicking = true
	return spectrumTick()
}

func spectrumTick() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(t time.Time) tea.Msg { return spectrumTickMsg(t) })
}

func (m Model) spectrumVisible() bool {
	return m.mode == ModeNowPlaying && m.spectrumMode && !m.quitting
}

func (m *Model) advanceSpectrum(now time.Time) {
	frame := spectrum.Default.Snapshot()
	active := m.track != nil && m.track.Playing && m.opts.PlayerName != "" && m.track.DeviceName == m.opts.PlayerName
	fresh := active && !frame.Updated.IsZero() && now.Sub(frame.Updated) < 300*time.Millisecond
	dt := 0.04
	if !m.spectrumLast.IsZero() {
		dt = max(0, min(0.15, now.Sub(m.spectrumLast).Seconds()))
	}
	m.spectrumLast = now
	for b := range spectrum.Bands {
		target := 0.0
		if fresh {
			target = frame.Levels[b]
		}
		// Quick attack, smooth release, a brief hold on each falling peak cap.
		m.spectrum.levels[b] = max(target, m.spectrum.levels[b]-dt*1.5)
		if m.spectrum.levels[b] >= m.spectrum.peaks[b] {
			m.spectrum.peaks[b] = m.spectrum.levels[b]
			m.spectrumHold[b] = now.Add(300 * time.Millisecond)
		} else if now.After(m.spectrumHold[b]) {
			m.spectrum.peaks[b] = max(m.spectrum.levels[b], m.spectrum.peaks[b]-dt*0.65)
		}
	}
	switch {
	case m.track == nil:
		m.spectrum.status = "No track playing"
	case !m.track.Playing:
		m.spectrum.status = "Paused"
	case !active:
		m.spectrum.status = "Press M to play through Side A on this Mac"
	case !fresh:
		m.spectrum.status = "Waiting for local audio"
	default:
		m.spectrum.status = "40 Hz                       16 kHz"
	}
}

// renderSpectrum uses the live palette, including user overrides. The dots
// and peak caps deliberately leave space between columns, like a classic analyzer.
func renderSpectrum(v spectrumView, w, h int) string {
	if w < 2 || h < 2 {
		return ""
	}
	bars := min(spectrum.Bands, w/2)
	graphH := h - 2
	dot := lipgloss.NewStyle().Foreground(ColorAccent).Background(ColorBg).Render("▪")
	peak := lipgloss.NewStyle().Foreground(ColorText).Background(ColorBg).Render("─")
	blank := lipgloss.NewStyle().Background(ColorBg).Render(" ")
	var rows []string
	for y := graphH - 1; y >= 0; y-- {
		var line strings.Builder
		for b := range bars {
			// Max-pool bands when a small terminal has room for fewer columns.
			level, cap := 0.0, 0.0
			for k := b * spectrum.Bands / bars; k < (b+1)*spectrum.Bands/bars; k++ {
				level = max(level, v.levels[k])
				cap = max(cap, v.peaks[k])
			}
			top := int(math.Ceil(cap*float64(graphH))) - 1
			switch {
			case cap > 0 && y == top:
				line.WriteString(peak)
			case float64(y) < level*float64(graphH):
				line.WriteString(dot)
			default:
				line.WriteString(blank)
			}
			line.WriteString(blank)
		}
		rows = append(rows, line.String())
	}
	rows = append(rows, lipgloss.NewStyle().Background(ColorBg).Render(strings.Repeat(" ", bars*2)))
	status := v.status
	if lipgloss.Width(status) > w {
		status = "Play locally: M"
	}
	if lipgloss.Width(status) > w {
		status = ""
	}
	rows = append(rows, lipgloss.NewStyle().Foreground(ColorTextDim).Background(ColorBg).Render(status))
	return strings.Join(rows, "\n")
}
