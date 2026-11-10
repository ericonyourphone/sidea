package ui

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/spectrum"
)

type layout struct {
	library, deck, right                                             bounds
	controlsY, controlsX, controlsH, progressY, listY, listH, graphH int
}

func (m Model) layout() layout {
	h := max(1, m.height-3)
	l := layout{}
	switch {
	case m.width >= 110:
		lw := min(28, max(22, m.width/5))
		dw := min(64, max(36, (m.width-lw)/2))
		l.library = bounds{0, 1, lw, h}
		l.deck = bounds{lw, 1, dw, h}
		l.right = bounds{lw + dw, 1, m.width - lw - dw, h}
	case m.width >= 78:
		left := min(40, m.width/2)
		if m.focusLibrary {
			l.library = bounds{0, 1, left, h}
		} else {
			l.deck = bounds{0, 1, left, h}
		}
		l.right = bounds{left, 1, m.width - left, h}
	default:
		if m.focusLibrary {
			l.library = bounds{0, 1, m.width, h}
		} else {
			l.right = bounds{0, 1, m.width, h}
		}
	}
	if l.deck.w > 0 {
		cassetteH := min(16, max(8, l.deck.h-10))
		l.controlsY = l.deck.y + 1 + 2 + cassetteH + 1
		l.controlsX = l.deck.x + (l.deck.w-26)/2
		l.controlsH = 3
	}
	l.progressY = l.right.y + 4
	inner := max(0, l.right.h-2)
	l.graphH = min(5, max(3, (inner-10)/3))
	l.listY = l.right.y + 8
	l.listH = max(1, inner-8-l.graphH)
	if !m.browsing && m.track != nil {
		l.listY++
		l.listH = max(1, l.listH-1)
	}
	return l
}
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = ansi.Truncate(s, w, "")
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}
func label(s string, w int) string { return ansi.Truncate(clean(s), max(0, w), "…") }
func paint(s string, c lipgloss.Color) string {
	style := lipgloss.NewStyle().Foreground(c).Background(app.ColorBg)
	// A nested color reset must restore the parent palette before its
	// trailing padding, rather than exposing the terminal's own background.
	sample := style.Render("X")
	prefix, suffix, ok := strings.Cut(sample, "X")
	if !ok {
		return style.Render(s)
	}
	return prefix + strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+prefix) + suffix
}
func accent(s string) string        { return paint(s, app.ColorAccent) }
func dim(s string) string           { return paint(s, app.ColorTextSec) }
func center(s string, w int) string { return strings.Repeat(" ", max(0, (w-lipgloss.Width(s))/2)) + s }
func duration(d int64) string {
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%d:%02d", d/60, d%60)
}

// box always occupies exactly w columns and h rows, including its border.
func box(title string, rows []string, b bounds, active bool) string {
	if b.w < 2 || b.h < 2 {
		return ""
	}
	color := app.ColorBorder
	if active {
		color = app.ColorAccent
	}
	border := func(s string) string { return paint(s, color) }
	top := "─ " + label(title, b.w-6) + " "
	lines := []string{border("┌" + fit(top, b.w-2) + "┐")}
	for y := 0; y < b.h-2; y++ {
		line := ""
		if y < len(rows) {
			line = rows[y]
		}
		lines = append(lines, border("│")+paint(fit(line, b.w-2), app.ColorText)+border("│"))
	}
	lines = append(lines, border("└"+strings.Repeat("─", b.w-2)+"┘"))
	return strings.Join(lines, "\n")
}
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.width < 60 || m.height < 20 {
		return paint(fit("Side A needs a terminal of at least 60 × 20", m.width), app.ColorText)
	}
	if m.help {
		return m.helpView()
	}
	if m.devicePicker {
		return m.deviceView()
	}
	l := m.layout()
	header := accent(" SIDE A ") + dim(" a cassette for your Spotify")
	if m.demo {
		header += accent("  [DEMO]")
	}
	theme := dim(" [t] " + m.themeName() + " ")
	header = paint(fit(header, max(0, m.width-lipgloss.Width(theme)))+theme, app.ColorText)
	var panes []string
	if l.library.w > 0 {
		panes = append(panes, m.libraryView(l.library))
	}
	if l.deck.w > 0 && l.right.w > 0 {
		panes = append(panes, m.playerView(l))
	} else if l.right.w > 0 {
		panes = append(panes, m.rightView(l))
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, panes...)
	footer := m.footer()
	view := header + "\n" + body + "\n" + footer
	if m.searchOpen {
		view = overlay(view, m.searchView(), m.searchBounds(), m.width, m.height)
	}
	return view
}
func (m Model) libraryView(b bounds) string {
	var rows []string
	visible := b.h - 2
	offset := max(0, m.libraryIndex-visible+1)
	for i := offset; i < len(m.library) && len(rows) < visible; i++ {
		item := m.library[i]
		prefix := "  "
		if i == m.libraryIndex {
			prefix = "> "
		}
		line := prefix + label(item.name, b.w-5)
		if i == m.libraryIndex {
			line = accent(line)
		} else {
			line = dim(line)
		}
		rows = append(rows, line)
	}
	return box("LIBRARY", rows, b, m.focusLibrary)
}
func (m Model) playerView(l layout) string {
	deck := strings.Split(m.deckView(l), "\n")
	right := strings.Split(m.rightView(l), "\n")
	rows := make([]string, l.deck.h-2)
	for i := range rows {
		rows[i] = ansi.Cut(deck[i+1], 1, l.deck.w-1) + "  " + ansi.Cut(right[i+1], 1, l.right.w-1)
	}
	return box("NOW PLAYING", rows, bounds{x: l.deck.x, y: l.deck.y, w: l.deck.w + l.right.w, h: l.deck.h}, !m.focusLibrary)
}
func (m Model) deckView(l layout) string {
	b := l.deck
	w := b.w - 2
	artist, song := "YOUR NEXT FAVORITE", "Pick a song from the library"
	if m.track != nil {
		artist, song = m.track.Artist, m.track.Name
	}
	rows := []string{"", ""}
	ch := min(16, max(8, b.h-10))
	cw := min(56, w-4)
	playing := m.track != nil && m.track.Playing
	rows = append(rows, cassette(artist, song, cw, ch, m.frame, playing, w)...)
	rows = append(rows, "")
	glyphs := []string{"▶", "Ⅱ", "◀◀", "▶▶"}
	for y := 0; y < 3; y++ {
		var buttons []string
		for i, g := range glyphs {
			text := "┌───┐"
			if y == 1 {
				text = "│" + fit(center(g, 3), 3) + "│"
			}
			if y == 2 {
				text = "└───┘"
			}
			style := lipgloss.NewStyle().Foreground(app.ColorText).Background(app.ColorSurface)
			if i == 0 && playing || i == 1 && !playing {
				style = style.Foreground(app.ColorBg).Background(app.ColorAccent)
			}
			buttons = append(buttons, style.Render(text))
		}
		rows = append(rows, center(strings.Join(buttons, "  "), w))
	}
	status := "READY"
	if m.track != nil {
		status = "PAUSED"
		if playing {
			status = "PLAYING"
		}
	}
	rows = append(rows, "", center(dim(status), w))
	return box("CASSETTE", rows, b, false)
}

func (m Model) rightView(l layout) string {
	b := l.right
	w := b.w - 4
	inner := b.h - 2
	rows := make([]string, inner)
	title, artist := "Nothing playing", "Choose a song, or press M to play here"
	if m.track != nil {
		title = m.track.Name
		artist = m.track.Artist
	}
	rows[0] = " " + accent(label(title, w))
	rows[1] = " " + dim(label(artist, w))
	position, total := m.position(), int64(0)
	if m.track != nil {
		total = int64(m.track.Duration.Seconds())
	}
	barW := max(1, w)
	ratio := 0.0
	if total > 0 {
		ratio = min(1, position.Seconds()/float64(total))
	}
	filled := min(barW, int(ratio*float64(barW)))
	rows[3] = " " + accent(strings.Repeat("━", filled)) + dim(strings.Repeat("─", barW-filled))
	times := duration(int64(position.Seconds()))
	end := duration(total)
	rows[4] = " " + dim(times+strings.Repeat(" ", max(1, w-len(times)-len(end)))+end)
	section := "QUEUE"
	if m.browsing {
		section = strings.ToUpper(m.title)
	}
	if m.loading || !m.browsing && m.queueLoading {
		section += " · loading"
	}
	rows[6] = " " + dim(label(section, w))
	list := m.visibleRows()
	offset := max(0, m.rowIndex-l.listH+1)
	listStart := l.listY - b.y - 1
	if !m.browsing && m.track != nil {
		rows[7] = " " + m.trackLine(row{track: *m.track}, w, false, true, 0)
	}
	for n := 0; n < l.listH; n++ {
		at := offset + n
		if at >= len(list) {
			break
		}
		r := list[at]
		selected := !m.focusLibrary && at == m.rowIndex
		playing := m.browsing && m.isCurrentTrack(r.track)
		rows[listStart+n] = " " + m.trackLine(r, w, selected, playing, at+1)
	}
	if len(list) == 0 && !m.loading {
		for i, line := range m.emptyState() {
			if i >= l.listH {
				break
			}
			rows[listStart+i] = " " + dim(label(line, w))
		}
	}
	graphStart := inner - l.graphH
	graph := m.spectrumView(w, l.graphH)
	for i, line := range graph {
		if graphStart+i >= 0 && graphStart+i < inner {
			rows[graphStart+i] = " " + line
		}
	}
	return box("NOW PLAYING", rows, b, !m.focusLibrary)
}
func (m Model) spectrumView(w, h int) []string {
	active := m.track != nil && m.track.Playing
	status := ""
	if m.track == nil {
		status = "No audio playing"
	} else if !m.track.Playing {
		status = "Paused"
	} else if !m.demo && m.track.DeviceName != m.opts.PlayerName {
		status = "M: listen here for live spectrum"
	}
	graphH := h
	if status != "" {
		graphH = max(1, h-1)
	}
	bars := min(spectrum.Bands, max(1, w/2))
	middle := graphH - 1
	rows := make([]string, 0, h)
	for y := 0; y < graphH; y++ {
		var line strings.Builder
		for b := 0; b < bars; b++ {
			level := 0.0
			for k := b * spectrum.Bands / bars; k < (b+1)*spectrum.Bands/bars; k++ {
				level = max(level, m.levels[k])
			}
			amplitude := int(math.Round(level * level * float64(middle)))
			top, bottom := false, false
			if active {
				top = absInt(y*2-middle) <= amplitude
				bottom = absInt(y*2+1-middle) <= amplitude
			}
			cell := " "
			switch {
			case top && bottom:
				cell = "█"
			case top:
				cell = "▀"
			case bottom:
				cell = "▄"
			}
			line.WriteString(accent(cell) + " ")
		}
		rows = append(rows, line.String())
	}
	if status != "" {
		rows = append(rows, dim(label(status, w)))
	}
	return rows
}
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (m Model) themeName() string {
	for _, name := range app.ThemeNames() {
		p, _ := app.ResolveTheme(name, m.opts.ColorOverrides)
		if p == app.CurrentPalette() {
			return name
		}
	}
	return "custom"
}

func (m Model) footer() string {
	device := "No active device"
	if m.track != nil && m.track.DeviceName != "" {
		device = m.track.DeviceName
	}
	status := fmt.Sprintf(" %s · volume %d%%", clean(device), m.volume)
	if m.actionBusy {
		status = " Working…"
	}
	if !m.noticeUntil.IsZero() && time.Now().Before(m.noticeUntil) {
		status = " " + clean(m.notice)
	}
	line1 := dim(fit(status, m.width))
	hints := " Tab library · Space play/pause · n/p skip · / search · t theme · M listen here · ? help · q quit"
	if m.editing != "" {
		hints = " " + m.search.View()
	}
	return line1 + "\n" + paint(fit(hints, m.width), app.ColorTextSec)
}

func (m Model) helpView() string {
	lines := []string{" SIDE A · a cassette for your Spotify", "", " Tab / h / l     Switch between library and songs", " j/k or arrows   Move selection", " Enter           Open library item / play selected song", " Space           Play / pause", " n / p           Next / previous", " [ / ]           Seek backward / forward 5 seconds", " + / -           Adjust Spotify volume", " M               Listen through Side A on this Mac", " D               Choose a Spotify device", " /               Search artists, songs, and albums", " a               Add selected track to queue", " Q / Esc         Return to Up next", " t               Cycle color themes", " :theme <name>   Choose a theme by name", " q / Ctrl+C      Quit", "", " ? / Esc         Close help"}
	return box("HELP", lines, bounds{w: m.width, h: m.height}, true)
}
func (m Model) deviceView() string {
	rows := []string{" Pick a device and press Enter", ""}
	for i, d := range m.devices {
		prefix := "   "
		if i == m.deviceIndex {
			prefix = " > "
		}
		line := prefix + clean(d.Name)
		if d.IsActive {
			line += " (active)"
		}
		if i == m.deviceIndex {
			line = accent(line)
		}
		rows = append(rows, line)
	}
	if len(m.devices) == 0 {
		rows = append(rows, " No devices online yet. Press Esc and try again.")
	}
	return box("SPOTIFY DEVICES · Esc closes", rows, bounds{w: m.width, h: m.height}, true)
}

func (m *Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.searchOpen {
		return m.searchMouse(msg)
	}
	if m.help || m.devicePicker || m.editing != "" {
		return *m, nil
	}
	if msg.Action != tea.MouseActionPress {
		return *m, nil
	}
	l := m.layout()
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		m.focusLibrary = l.library.contains(msg.X, msg.Y)
	}
	if msg.Button == tea.MouseButtonWheelUp {
		m.move(-1)
		return *m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return *m, m.move(1)
	}
	if msg.Button != tea.MouseButtonLeft {
		return *m, nil
	}
	if msg.Y == 0 {
		return *m, m.theme()
	}
	if l.library.contains(msg.X, msg.Y) {
		offset := max(0, m.libraryIndex-(l.library.h-2)+1)
		at := msg.Y - l.library.y - 1 + offset
		if at >= 0 && at < len(m.library) {
			m.libraryIndex = at
			m.focusLibrary = true
			return *m, m.openLibrary()
		}
		return *m, nil
	}
	if l.deck.w > 0 && msg.Y >= l.controlsY && msg.Y < l.controlsY+l.controlsH && msg.X >= l.controlsX && msg.X < l.controlsX+26 && (msg.X-l.controlsX)%7 < 5 {
		index := (msg.X - l.controlsX) / 7
		switch index {
		case 0:
			return *m, m.act(m.src.Play)
		case 1:
			return *m, m.act(m.src.Pause)
		case 2:
			return *m, m.act(m.src.Previous)
		case 3:
			return *m, m.act(m.src.Next)
		}
	}
	if l.right.contains(msg.X, msg.Y) {
		m.focusLibrary = false
		if msg.Y == l.progressY && m.track != nil {
			ratio := float64(msg.X-l.right.x-2) / float64(max(1, l.right.w-4))
			return *m, m.seek(time.Duration(max(0, min(1, ratio)) * float64(m.track.Duration)))
		}
		if msg.Y >= l.listY && msg.Y < l.listY+l.listH {
			offset := max(0, m.rowIndex-l.listH+1)
			at := msg.Y - l.listY + offset
			if at < len(m.visibleRows()) {
				m.rowIndex = at
				return *m, m.openRow()
			}
		}
	}
	return *m, nil
}
