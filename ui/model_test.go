package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/source"
	"github.com/muesli/termenv"
)

type stub struct {
	source.RichSource
	play, pause, next, previous int
	queued                      string
}

func (s *stub) Play(context.Context) error                    { s.play++; return nil }
func (s *stub) Pause(context.Context) error                   { s.pause++; return nil }
func (s *stub) Next(context.Context) error                    { s.next++; return nil }
func (s *stub) Previous(context.Context) error                { s.previous++; return nil }
func (s *stub) AddToQueue(_ context.Context, id string) error { s.queued = id; return nil }
func fixture() Model {
	m := NewModel(&stub{}, app.Options{PlayerName: "sidea (test)"})
	m.track = &source.Track{ID: "current", Name: "Snowy", Artist: "Toby Fox", Duration: 104 * time.Second, Playing: true, DeviceName: "sidea (test)"}
	m.pollPosition = 35 * time.Second
	m.pollAt = time.Now()
	m.width, m.height = 140, 30
	m.queue = []source.Track{{ID: "one", Name: "Fallen Down", Artist: "Toby Fox", Duration: 85 * time.Second}, {ID: "two", Name: "Home", Duration: 123 * time.Second}}
	m.polling, m.queueLoading = false, false
	return m
}
func send(m Model, k string) (Model, tea.Cmd) {
	key := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	switch k {
	case "tab":
		key = tea.KeyMsg{Type: tea.KeyTab}
	case "enter":
		key = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		key = tea.KeyMsg{Type: tea.KeyEsc}
	}
	next, cmd := m.Update(key)
	return next.(Model), cmd
}
func TestRenderingFitsAllSizesAndThemes(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	for _, name := range app.ThemeNames() {
		p, _ := app.ResolveTheme(name, app.Palette{})
		app.ApplyTheme(p)
		for _, w := range []int{60, 77, 78, 100, 109, 110, 140, 200} {
			for _, h := range []int{20, 24, 40} {
				for _, library := range []bool{false, true} {
					m := fixture()
					m.width, m.height = w, h
					m.focusLibrary = library
					m.track.Name = strings.Repeat("A very long 曲 title ", 10)
					out := m.View()
					lines := strings.Split(out, "\n")
					if len(lines) != h {
						t.Fatalf("%s %dx%d: got %d rows", name, w, h, len(lines))
					}
					for i, line := range lines {
						if lipgloss.Width(line) != w {
							t.Fatalf("%s %dx%d row %d has width %d: %q", name, w, h, i, lipgloss.Width(line), line)
						}
					}
				}
			}
		}
	}
}
func TestPlaybackButtonStateSurvivesUpdate(t *testing.T) {
	m := fixture()
	s := m.src.(*stub)
	m, cmd := send(m, " ")
	if !m.actionBusy || cmd == nil {
		t.Fatal("playback command did not set pending state")
	}
	msg := cmd()
	if s.pause != 1 || s.play != 0 {
		t.Fatal("wrong playback action")
	}
	result, follow := m.Update(msg)
	m = result.(Model)
	if m.actionBusy || follow == nil {
		t.Fatal("completion did not release control and schedule refresh")
	}
}
func TestMouseControlsMatchVisibleButtons(t *testing.T) {
	for i := range 4 {
		m := fixture()
		l := m.layout()
		s := m.src.(*stub)
		result, cmd := m.Update(tea.MouseMsg{X: l.controlsX + i*7 + 2, Y: l.controlsY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
		if !result.(Model).actionBusy || cmd == nil {
			t.Fatal("button not clickable", i)
		}
		cmd()
		calls := []int{s.play, s.pause, s.previous, s.next}
		if calls[i] != 1 {
			t.Fatal("wrong button action", i, calls)
		}
	}
}
func TestStaleCollectionDoesNotReplaceCurrent(t *testing.T) {
	m := fixture()
	m.generation = 2
	m.browsing = true
	m.title = "Current"
	result, _ := m.Update(rowsMsg{title: "Stale", generation: 1, rows: []row{{track: source.Track{Name: "Old"}}}})
	m = result.(Model)
	if m.title != "Current" || len(m.rows) != 0 {
		t.Fatal("stale request replaced current selection")
	}
}
func TestQueueRowAdvancesInOrder(t *testing.T) {
	m := fixture()
	m.rowIndex = 1
	m, cmd := send(m, "enter")
	if cmd == nil {
		t.Fatal("missing queue action")
	}
	msg := cmd().(actionMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if m.src.(*stub).next != 2 {
		t.Fatal("queue jump did not advance to selected song")
	}
}
func TestThemesAndCommandEntry(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	m := fixture()
	saved := ""
	m.opts.SaveTheme = func(s string) error { saved = s; return nil }
	m, _ = send(m, ":")
	if m.editing != "command" {
		t.Fatal("command entry did not open")
	}
	m.search.SetValue("theme nord")
	m, _ = send(m, "enter")
	if saved != "nord" || app.CurrentPalette() != app.Themes["nord"] {
		t.Fatal("theme command did not persist/apply")
	}
}
func TestSpectrumKeepsRunningWithoutFocusAndDecays(t *testing.T) {
	m := fixture()
	for b := range m.levels {
		m.levels[b] = 1
		m.peaks[b] = 1
	}
	result, _ := m.Update(tea.BlurMsg{})
	m = result.(Model)
	now := time.Now()
	for i := range 60 {
		result, _ = m.Update(tickMsg(now.Add(time.Duration(i) * 40 * time.Millisecond)))
		m = result.(Model)
	}
	if m.frame != 60 {
		t.Fatal("animation stopped when blurred")
	}
	for _, v := range m.levels {
		if v != 0 {
			t.Fatal("stale audio did not decay")
		}
	}
}

type rateError struct{}

func (rateError) Error() string          { return "rate limited" }
func (rateError) HTTPStatus() int        { return 429 }
func (rateError) RetryAfterSeconds() int { return 120 }
func TestRateLimitPreventsPollingAndControls(t *testing.T) {
	m := fixture()
	result, _ := m.Update(stateMsg{err: fmt.Errorf("wrapped: %w", rateError{})})
	m = result.(Model)
	if time.Until(m.cooldown) < 119*time.Second {
		t.Fatal("server retry duration was ignored")
	}
	_, cmd := send(m, "n")
	if cmd != nil {
		t.Fatal("sent command during cooldown")
	}
	if m.refresh() != nil {
		t.Fatal("sent refresh during cooldown")
	}
}
func TestPlaybackErrorIsVisible(t *testing.T) {
	m := fixture()
	result, _ := m.Update(actionMsg{err: errors.New("device unavailable")})
	m = result.(Model)
	if !strings.Contains(m.View(), "device unavailable") {
		t.Fatal("playback failure was hidden")
	}
}
func TestExternalNamesCannotBreakTerminalLayout(t *testing.T) {
	m := fixture()
	m.track.Name = "Title\n\x1b[31mred\x1b[0m\t\r"
	if strings.Contains(clean(m.track.Name), "\x1b") {
		t.Fatal("terminal escapes were retained")
	}
	out := m.View()
	if len(strings.Split(out, "\n")) != m.height {
		t.Fatal("external name broke layout")
	}
}

func TestNestedStylesRestoreThemeBackground(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	profile := lipgloss.ColorProfile()
	defer lipgloss.SetColorProfile(profile)
	lipgloss.SetColorProfile(termenv.TrueColor)
	app.ApplyTheme(app.Themes["nord"])
	out := paint(accent("Colored label")+"   padding", app.ColorText)
	parts := strings.Split(out, "\x1b[0m")
	for _, part := range parts[1:] {
		if part != "" && !strings.HasPrefix(part, "\x1b[") {
			t.Fatalf("nested style left padding on terminal background: %q", part)
		}
	}
}

type idleSource struct {
	stub
	activated string
}

func (s *idleSource) Play(context.Context) error {
	s.play++
	if s.activated == "" {
		return source.ErrNoActiveDevice
	}
	return nil
}
func (s *idleSource) Devices(context.Context) ([]source.Device, error) {
	return []source.Device{{ID: "local", Name: "sidea (test)"}, {ID: "phone", Name: "Phone"}}, nil
}
func (s *idleSource) TransferPlayback(_ context.Context, id string) error {
	s.activated = id
	return nil
}
func TestFirstPlaybackActivatesOnlyLocalDevice(t *testing.T) {
	src := &idleSource{}
	m := NewModel(src, app.Options{PlayerName: "sidea (test)"})
	msg := m.control(src.Play)().(actionMsg)
	if msg.err != nil {
		t.Fatal(msg.err)
	}
	if src.activated != "local" || src.play != 2 {
		t.Fatal("idle playback did not select and retry the local device", src.activated, src.play)
	}
}

type playlistPlayer struct {
	stub
	contextURI, trackURI string
}

func (s *playlistPlayer) PlayTrack(_ context.Context, contextURI, trackURI string) error {
	s.contextURI, s.trackURI = contextURI, trackURI
	return nil
}
func TestRestrictedPlaylistShowsReasonAndCanPlayContext(t *testing.T) {
	m := fixture()
	src := &playlistPlayer{}
	m.src = src
	m.browsing = true
	m.generation = 5
	m.contextURI = "spotify:playlist:restricted"
	m.collectionID = "restricted"
	m.loading = true
	result, _ := m.Update(rowsMsg{generation: 5, err: fmt.Errorf("%w: 403", source.ErrPlaylistRestricted)})
	m = result.(Model)
	m.noticeUntil = time.Now().Add(-time.Second)
	view := m.View()
	if !strings.Contains(view, "Spotify restricts this track list") || strings.Contains(view, "No tracks found") {
		t.Fatal("restriction was displayed as empty")
	}
	m, cmd := send(m, "enter")
	if cmd == nil {
		t.Fatal("restricted playlist cannot be played")
	}
	msg := cmd().(actionMsg)
	if msg.err != nil || src.contextURI != "spotify:playlist:restricted" || src.trackURI != "" {
		t.Fatal("incorrect playlist playback", src.contextURI, src.trackURI, msg.err)
	}
}
func TestFailedCollectionIsNotDisplayedAsEmpty(t *testing.T) {
	m := fixture()
	m.browsing = true
	m.generation = 1
	m.rows = nil
	result, _ := m.Update(rowsMsg{generation: 1, err: errors.New("network error")})
	m = result.(Model)
	m.noticeUntil = time.Now().Add(-time.Second)
	if !strings.Contains(m.View(), "Could not load tracks.") {
		t.Fatal("failed load was treated as empty")
	}
	result, _ = m.Update(rowsMsg{generation: 1, total: 0})
	m = result.(Model)
	if m.collectionErr != nil || !strings.Contains(m.View(), "This collection has no tracks") {
		t.Fatal("successful empty result did not clear failure")
	}
}
func TestOpeningNewCollectionClearsOldError(t *testing.T) {
	m := fixture()
	m.collectionErr = source.ErrPlaylistRestricted
	m.libraryIndex = 1
	// Creating the command should clear the previous collection's error;
	// the stub need not execute a Spotify request for this state check.
	cmd := m.openLibrary()
	if cmd == nil || m.collectionErr != nil || !m.loading {
		t.Fatal("old collection error leaked into next request")
	}
}

func TestPaginationErrorDoesNotReplaceSelectedTrackPlayback(t *testing.T) {
	m := fixture()
	src := &playlistPlayer{}
	m.src = src
	m.browsing = true
	m.contextURI = "spotify:playlist:known"
	m.collectionErr = source.ErrPlaylistRestricted
	m.rows = []row{{track: source.Track{ID: "one", URI: "spotify:track:one"}}}
	m.rowIndex = 0
	cmd := m.openRow()
	if cmd == nil {
		t.Fatal("loaded track was not playable")
	}
	result := cmd().(actionMsg)
	if result.err != nil || src.trackURI != "spotify:track:one" {
		t.Fatal("pagination error replaced a known track with context playback", result.err, src.trackURI)
	}
}
