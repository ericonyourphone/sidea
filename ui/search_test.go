package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/source"
	"github.com/muesli/termenv"
)

type searchSource struct {
	stub
	queries                      []string
	playedURI, artistID, albumID string
	err                          error
}

func (s *searchSource) Search(_ context.Context, q string) (*source.SearchResults, error) {
	s.queries = append(s.queries, q)
	if s.err != nil {
		return nil, s.err
	}
	return &source.SearchResults{
		Tracks:  []source.Track{{ID: "song", URI: "spotify:track:song", Name: "First Song", Artist: "An Artist"}},
		Artists: []source.SearchArtist{{ID: "artist", Name: "An Artist"}},
		Albums:  []source.SearchAlbum{{ID: "album", Name: "An Album", Artist: "An Artist"}},
	}, nil
}
func (s *searchSource) PlayTrackDirect(_ context.Context, uri string) error {
	s.playedURI = uri
	return nil
}
func (s *searchSource) GetArtist(_ context.Context, id string) (*source.ArtistPage, error) {
	s.artistID = id
	return &source.ArtistPage{ID: id, Name: "An Artist", Tracks: []source.Track{{ID: "top", Name: "Top Song"}}, Albums: []source.ArtistAlbum{{ID: "album", Name: "An Album"}}}, nil
}
func (s *searchSource) GetAlbum(_ context.Context, id string) (*source.AlbumPage, error) {
	s.albumID = id
	return &source.AlbumPage{ID: id, Name: "An Album", Tracks: []source.Track{{ID: "album-song", Name: "Album Song"}}}, nil
}
func searchFixture(t *testing.T) (Model, *searchSource) {
	t.Helper()
	m := fixture()
	src := &searchSource{}
	m.src = src
	m.browsing = true
	m.title = "Keep this collection"
	m.rows = []row{{track: source.Track{Name: "Saved selection"}}}
	m, _ = send(m, "/")
	m.search.SetValue("test query")
	var cmd tea.Cmd
	m, cmd = send(m, "enter")
	if cmd == nil || !m.searchLoading {
		t.Fatal("search did not start")
	}
	result, _ := m.Update(cmd())
	return result.(Model), src
}
func TestSearchOverlayIncludesAllCategoriesAndPreservesCollection(t *testing.T) {
	m, src := searchFixture(t)
	if len(src.queries) != 1 || src.queries[0] != "test query" {
		t.Fatal("incorrect search request")
	}
	if !m.searchOpen || m.searchLoading || m.searchCount(searchSongs) != 1 || m.searchCount(searchArtists) != 1 || m.searchCount(searchAlbums) != 1 {
		t.Fatal("missing search categories")
	}
	m, _ = send(m, "esc")
	if m.searchOpen || m.title != "Keep this collection" || len(m.rows) != 1 || m.rows[0].track.Name != "Saved selection" {
		t.Fatal("closing search changed the underlying collection")
	}
}
func TestSearchSongPlaysDirectlyAndKeepsResults(t *testing.T) {
	m, src := searchFixture(t)
	m, cmd := send(m, "enter")
	if cmd == nil || !m.searchOpen || !m.actionBusy {
		t.Fatal("song selection did not start playback")
	}
	msg := cmd().(actionMsg)
	if msg.err != nil || src.playedURI != "spotify:track:song" {
		t.Fatal("wrong track played", msg.err, src.playedURI)
	}
}
func TestSearchAlbumAndArtistOpenAvailableTracks(t *testing.T) {
	for _, tab := range []int{searchAlbums, searchArtists} {
		m, src := searchFixture(t)
		m.searchTab = tab
		m, cmd := send(m, "enter")
		if m.searchOpen || !m.loading || cmd == nil {
			t.Fatal("result did not open a detail view")
		}
		result, _ := m.Update(cmd())
		m = result.(Model)
		if m.loading || len(m.rows) == 0 {
			t.Fatal("detail tracks did not load")
		}
		if tab == searchAlbums && (src.albumID != "album" || m.contextURI != "spotify:album:album") {
			t.Fatal("incorrect album")
		}
		if tab == searchArtists && (src.artistID != "artist" || len(m.rows) != 2 || m.rows[1].albumID != "album") {
			t.Fatal("artist tracks/albums missing")
		}
	}
}
func TestSearchTabKeyboardAndMouse(t *testing.T) {
	m, _ := searchFixture(t)
	m, _ = send(m, "tab")
	if m.searchTab != searchAlbums {
		t.Fatal("Tab did not select Albums")
	}
	m, _ = send(m, "tab")
	if m.searchTab != searchArtists {
		t.Fatal("Tab did not wrap to Artists")
	}
	_, zones := m.searchTabs()
	z := zones[searchSongs]
	result, _ := m.Update(tea.MouseMsg{X: z.x + 2, Y: z.y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = result.(Model)
	if m.searchTab != searchSongs {
		t.Fatal("tab click missed")
	}
	b := m.searchBounds()
	result, cmd := m.Update(tea.MouseMsg{X: b.x + 5, Y: b.y + 5, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = result.(Model)
	if cmd == nil || !m.actionBusy {
		t.Fatal("clicking song result did not play")
	}
}
func TestStaleAndClosedSearchResponsesAreIgnored(t *testing.T) {
	m, _ := searchFixture(t)
	old := m.searchGeneration
	m.search.SetValue("new query")
	cmd := m.submitSearch()
	if cmd == nil {
		t.Fatal("new search missing")
	}
	result, _ := m.Update(searchMsg{generation: old, query: "test query", results: &source.SearchResults{Tracks: []source.Track{{Name: "Stale"}}}})
	m = result.(Model)
	if m.searchResults != nil || !m.searchLoading {
		t.Fatal("old result overwrote new search")
	}
	m.closeSearch()
	result, _ = m.Update(cmd())
	m = result.(Model)
	if m.searchOpen || m.searchResults != nil {
		t.Fatal("closed search was reopened by a late result")
	}
}
func TestSearchErrorAndNoResultsAreDistinct(t *testing.T) {
	m, src := searchFixture(t)
	src.err = errors.New("Spotify connection failed")
	cmd := m.submitSearch()
	result, _ := m.Update(cmd())
	m = result.(Model)
	if !strings.Contains(m.View(), "Search could not be completed") || strings.Contains(m.searchView(), "No results in this category") {
		t.Fatal("search error was shown as no matches")
	}
	m.searchError = nil
	m.searchResults = &source.SearchResults{}
	if !strings.Contains(m.searchView(), "No results in this category") {
		t.Fatal("missing empty category state")
	}
}
func TestSearchOverlayFitsEveryThemeAndTerminalSize(t *testing.T) {
	old := app.CurrentPalette()
	defer app.ApplyTheme(old)
	profile := lipgloss.ColorProfile()
	defer lipgloss.SetColorProfile(profile)
	lipgloss.SetColorProfile(termenv.TrueColor)
	for _, theme := range app.ThemeNames() {
		p, _ := app.ResolveTheme(theme, app.Palette{})
		app.ApplyTheme(p)
		for _, size := range [][2]int{{60, 20}, {78, 24}, {110, 24}, {140, 30}, {200, 50}} {
			m, _ := searchFixture(t)
			m.width, m.height = size[0], size[1]
			m.search.SetValue(strings.Repeat("long query 曲 ", 20))
			out := m.View()
			lines := strings.Split(out, "\n")
			if len(lines) != m.height {
				t.Fatalf("%s %v: wrong height %d", theme, size, len(lines))
			}
			for i, line := range lines {
				if lipgloss.Width(line) != m.width {
					t.Fatalf("%s %v row %d: width %d", theme, size, i, lipgloss.Width(line))
				}
			}
		}
	}
}
func TestSearchCooldownDoesNotIssueRequest(t *testing.T) {
	m, src := searchFixture(t)
	m.cooldown = time.Now().Add(time.Minute)
	if cmd := m.submitSearch(); cmd != nil {
		t.Fatal("search request sent during cooldown")
	}
	if len(src.queries) != 1 || m.searchError == nil {
		t.Fatal("missing cooldown explanation")
	}
}
