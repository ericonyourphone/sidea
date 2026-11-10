package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/source"
	"github.com/muesli/termenv"
)

func TestPlayingTrackAndSelectionHaveIndependentMarkers(t *testing.T) {
	m := fixture()
	m.browsing = true
	m.focusLibrary = false
	m.rowIndex = 1
	m.rows = []row{{track: *m.track}, {track: source.Track{ID: "different", Name: "Selected track"}}}
	out := ansi.Strip(m.View())
	var playing, selected string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "▶  Snowy") {
			playing = line
		}
		if strings.Contains(line, ">  Selected track") {
			selected = line
		}
	}
	if playing == "" || selected == "" {
		t.Fatal("playback and cursor were not marked separately", out)
	}
	m.focusLibrary = true
	if !strings.Contains(ansi.Strip(m.View()), "▶  Snowy") {
		t.Fatal("playing marker disappeared when library got focus")
	}
	m.rowIndex = 0
	m.focusLibrary = false
	if !strings.Contains(ansi.Strip(m.View()), ">▶ Snowy") {
		t.Fatal("selected playing track lost one of its markers")
	}
}
func TestPlayingIdentityUsesIDsAndURIsNotNames(t *testing.T) {
	m := fixture()
	if m.isCurrentTrack(source.Track{ID: "other", Name: m.track.Name}) {
		t.Fatal("duplicate title was mistaken for playback")
	}
	if !m.isCurrentTrack(source.Track{ID: m.track.ID}) {
		t.Fatal("current ID was not recognized")
	}
	m.track.ID = ""
	m.track.URI = "spotify:track:current"
	if !m.isCurrentTrack(source.Track{URI: m.track.URI}) || m.isCurrentTrack(source.Track{}) {
		t.Fatal("URI/empty identity handling failed")
	}
}
func TestQueueIncludesCurrentTrackWithoutChangingSkipOffsets(t *testing.T) {
	m := fixture()
	out := ansi.Strip(m.rightView(m.layout()))
	if !strings.Contains(out, "▶  Snowy") {
		t.Fatal("queue omitted the current song")
	}
	l := m.layout()
	_, cmd := m.Update(tea.MouseMsg{X: l.right.x + 5, Y: l.listY - 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if cmd != nil {
		t.Fatal("clicking the nonselectable current track skipped a song")
	}
	_, cmd = m.Update(tea.MouseMsg{X: l.right.x + 5, Y: l.listY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if cmd == nil {
		t.Fatal("first upcoming track was not clickable")
	}
	cmd()
	if m.src.(*stub).next != 1 {
		t.Fatal("current row changed the queue skip offset")
	}
}
func TestCassetteLabelsAnimationAndTheme(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	profile := lipgloss.ColorProfile()
	defer lipgloss.SetColorProfile(profile)
	lipgloss.SetColorProfile(termenv.TrueColor)
	var previous string
	for _, theme := range []string{"spotify", "nord", "catppuccin-latte"} {
		p, _ := app.ResolveTheme(theme, app.Palette{})
		app.ApplyTheme(p)
		a := strings.Join(cassette("Artist", "Song", 48, 16, 0, true, 52), "\n")
		b := strings.Join(cassette("Artist", "Song", 48, 16, 4, true, 52), "\n")
		if !strings.Contains(ansi.Strip(a), "Artist") || !strings.Contains(ansi.Strip(a), "Song") {
			t.Fatal("cassette labels missing")
		}
		if a == b {
			t.Fatal("reels do not turn")
		}
		c := strings.Join(cassette("Artist", "Song", 48, 16, 0, false, 52), "\n")
		d := strings.Join(cassette("Artist", "Song", 48, 16, 40, false, 52), "\n")
		if c != d {
			t.Fatal("paused reels still turn")
		}
		if previous == a {
			t.Fatal("cassette ignored the theme")
		}
		previous = a
	}
}
func TestCenteredVisualizerIsSymmetricAndRespondsToEnergy(t *testing.T) {
	m := fixture()
	m.levels[0] = 1
	m.levels[11] = .5
	rows := m.spectrumView(6, 5)
	masks := make([][]bool, 3)
	for i := range masks {
		masks[i] = make([]bool, 10)
	}
	for y, line := range rows {
		cells := []rune(ansi.Strip(line))
		for b := range 3 {
			switch cells[b*2] {
			case '█':
				masks[b][y*2], masks[b][y*2+1] = true, true
			case '▀':
				masks[b][y*2] = true
			case '▄':
				masks[b][y*2+1] = true
			}
		}
	}
	counts := []int{0, 0, 0}
	for b, mask := range masks {
		for y := 0; y < 9; y++ {
			if mask[y] != mask[8-y] {
				t.Fatal("bar is not mirrored around its center", b, mask)
			}
			if mask[y] {
				counts[b]++
			}
		}
	}
	if counts[0] <= counts[1] || counts[1] <= counts[2] {
		t.Fatal("bar heights do not reflect energy", counts)
	}
	if strings.Contains(strings.Join(rows, ""), "Hz") {
		t.Fatal("old analyzer axis labels remain")
	}
}
func TestPlayingRowUsesThemeAccentWhileCursorIsNeutral(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	profile := lipgloss.ColorProfile()
	defer lipgloss.SetColorProfile(profile)
	lipgloss.SetColorProfile(termenv.TrueColor)
	app.ApplyTheme(app.Themes["nord"])
	m := fixture()
	current := m.trackLine(row{track: *m.track}, 40, false, true, 0)
	selected := m.trackLine(row{track: source.Track{Name: "Other"}}, 40, true, false, 1)
	if !strings.Contains(current, "38;2;163;190;140") {
		t.Fatal("playing row does not use the theme accent")
	}
	if strings.Contains(selected, "38;2;163;190;140") {
		t.Fatal("cursor still looks like the playing track")
	}
}
