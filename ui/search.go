package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/ericonyourphone/sidea/source"
)

const (
	searchArtists = iota
	searchSongs
	searchAlbums
)

type searchMsg struct {
	generation int
	query      string
	results    *source.SearchResults
	err        error
}

func (m *Model) openSearch() tea.Cmd {
	m.searchOpen = true
	m.searchEditing = true
	m.searchTab = searchSongs
	m.searchGeneration++
	m.searchResults = nil
	m.searchError = nil
	m.searchLoading = false
	m.searchQuery = ""
	m.searchIndex = [3]int{}
	m.search.SetValue("")
	m.search.Prompt = "Search: "
	m.search.Width = max(10, m.searchBounds().w-13)
	return m.search.Focus()
}
func (m *Model) closeSearch() {
	m.searchOpen = false
	m.searchEditing = false
	m.searchGeneration++
	m.search.Blur()
}
func (m *Model) submitSearch() tea.Cmd {
	query := strings.TrimSpace(m.search.Value())
	if query == "" {
		return nil
	}
	if time.Now().Before(m.cooldown) {
		m.searchGeneration++
		m.searchLoading = false
		m.searchError = fmt.Errorf("Spotify rate limit: please wait before searching again")
		return nil
	}
	m.searchGeneration++
	gen := m.searchGeneration
	m.searchQuery = query
	m.searchLoading = true
	m.searchError = nil
	m.searchResults = nil
	m.searchIndex = [3]int{}
	m.searchEditing = false
	m.search.Blur()
	src, ctx := m.src, m.ctx
	return func() tea.Msg {
		r, e := src.Search(ctx, query)
		return searchMsg{generation: gen, query: query, results: r, err: e}
	}
}
func (m Model) searchCount(tab int) int {
	r := m.searchResults
	if r == nil {
		return 0
	}
	switch tab {
	case searchArtists:
		return len(r.Artists)
	case searchSongs:
		return len(r.Tracks)
	case searchAlbums:
		return len(r.Albums)
	}
	return 0
}
func (m *Model) searchMove(n int) {
	tab := m.searchTab
	m.searchIndex[tab] = max(0, min(max(0, m.searchCount(tab)-1), m.searchIndex[tab]+n))
}
func (m *Model) searchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		m.closeSearch()
		return *m, nil
	}
	if m.searchEditing {
		switch key {
		case "enter":
			return *m, m.submitSearch()
		case "tab", "down":
			if m.searchResults != nil {
				m.searchEditing = false
				m.search.Blur()
				return *m, nil
			}
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return *m, cmd
	}
	switch key {
	case "q":
		m.closeSearch()
	case "/", "i", "ctrl+f":
		m.searchEditing = true
		return *m, m.search.Focus()
	case "tab", "right", "l":
		m.searchTab = (m.searchTab + 1) % 3
	case "shift+tab", "left", "h":
		m.searchTab = (m.searchTab + 2) % 3
	case "up", "k":
		m.searchMove(-1)
	case "down", "j":
		m.searchMove(1)
	case "pgdown", "ctrl+d":
		m.searchMove(6)
	case "pgup", "ctrl+u":
		m.searchMove(-6)
	case "enter":
		if m.searchError != nil {
			return *m, m.submitSearch()
		}
		return *m, m.openSearchResult()
	}
	return *m, nil
}
func (m *Model) openSearchResult() tea.Cmd {
	if m.searchLoading || m.searchResults == nil || m.searchCount(m.searchTab) == 0 {
		return nil
	}
	i := m.searchIndex[m.searchTab]
	r := m.searchResults
	switch m.searchTab {
	case searchSongs:
		track := r.Tracks[i]
		uri := track.URI
		if uri == "" {
			uri = "spotify:track:" + track.ID
		}
		return m.act(func(ctx context.Context) error { return m.src.PlayTrackDirect(ctx, uri) })
	case searchArtists:
		artist := r.Artists[i]
		m.generation++
		m.collectionErr = nil
		m.browsing, m.loading = true, true
		m.collectionID = "artist:" + artist.ID
		m.contextURI = "spotify:artist:" + artist.ID
		m.title = artist.Name
		m.rows = nil
		m.total = 0
		m.rowIndex, m.rowOffset = 0, 0
		m.focusLibrary = false
		m.closeSearch()
		return m.fetchCollection(0)
	case searchAlbums:
		album := r.Albums[i]
		m.generation++
		m.collectionErr = nil
		m.browsing, m.loading = true, true
		m.collectionID = "album:" + album.ID
		m.contextURI = "spotify:album:" + album.ID
		m.title = album.Name
		m.rows = nil
		m.total = 0
		m.rowIndex, m.rowOffset = 0, 0
		m.focusLibrary = false
		m.closeSearch()
		return m.fetchCollection(0)
	}
	return nil
}
func (m Model) searchBounds() bounds {
	w := min(82, max(20, m.width-6))
	h := min(22, max(10, m.height-4))
	return bounds{(m.width - w) / 2, (m.height - h) / 2, w, h}
}
func (m Model) searchListHeight() int { return max(1, m.searchBounds().h-9) }
func (m Model) searchTabs() ([]string, []bounds) {
	b := m.searchBounds()
	names := []string{"Artists", "Songs", "Albums"}
	tabs := make([]string, 3)
	zones := make([]bounds, 3)
	x := b.x + 2
	for i, name := range names {
		text := fmt.Sprintf(" %s (%d) ", name, m.searchCount(i))
		zones[i] = bounds{x, b.y + 3, len(text), 1}
		x += len(text) + 1
		if m.searchTab == i {
			tabs[i] = accent("[" + strings.TrimSpace(text) + "]")
		} else {
			tabs[i] = dim(text)
		}
	}
	return tabs, zones
}
func (m Model) searchView() string {
	b := m.searchBounds()
	w := b.w - 4
	rows := make([]string, b.h-2)
	input := m.search
	input.Width = max(10, w-9)
	rows[0] = " " + input.View()
	tabs, _ := m.searchTabs()
	rows[2] = " " + strings.Join(tabs, " ")
	index, count := m.searchIndex[m.searchTab], m.searchCount(m.searchTab)
	visible := m.searchListHeight()
	offset := max(0, index-visible+1)
	switch {
	case m.searchLoading:
		rows[4] = " " + dim("Searching Spotify…")
	case m.searchError != nil:
		rows[4] = " " + dim("Search could not be completed.")
		if visible > 1 {
			rows[5] = " " + dim(label(m.searchError.Error(), w))
		}
	case m.searchResults == nil:
		rows[4] = " " + dim("Type an artist, song, or album, then Enter.")
	case count == 0:
		rows[4] = " " + dim("No results in this category.")
	default:
		for n := 0; n < visible && offset+n < count; n++ {
			i := offset + n
			title, detail := "", ""
			switch m.searchTab {
			case searchArtists:
				title = m.searchResults.Artists[i].Name
				detail = "artist"
			case searchSongs:
				t := m.searchResults.Tracks[i]
				title = t.Name
				detail = t.Artist
			case searchAlbums:
				a := m.searchResults.Albums[i]
				title = a.Name
				detail = a.Artist
			}
			prefix := "  "
			if i == index {
				prefix = "> "
			}
			line := prefix + label(title+" · "+detail, w-2)
			if i == index {
				line = accent(line)
			} else {
				line = dim(line)
			}
			rows[4+n] = " " + line
		}
	}
	hint := "Tab category · ↑/↓ select · Enter open · / edit · Esc close"
	if m.searchTab == searchSongs {
		hint = "Tab category · ↑/↓ select · Enter play · / edit · Esc close"
	}
	if m.searchEditing {
		hint = "Enter search · Esc close"
	}
	if m.actionBusy {
		hint = "Starting playback…"
	}
	rows[len(rows)-1] = " " + dim(label(hint, w))
	return box("SPOTIFY SEARCH · Esc closes", rows, b, true)
}

// overlay replaces complete cell ranges and preserves ANSI colors on both
// sides. The background screen keeps its dimensions and selection intact.
func overlay(base, panel string, b bounds, width, height int) string {
	lines := strings.Split(base, "\n")
	popup := strings.Split(panel, "\n")
	for y, line := range popup {
		at := b.y + y
		if at < 0 || at >= height || at >= len(lines) {
			continue
		}
		lines[at] = fit(ansi.Cut(lines[at], 0, b.x), b.x) + fit(line, b.w) + fit(ansi.Cut(lines[at], b.x+b.w, width), max(0, width-b.x-b.w))
	}
	return strings.Join(lines, "\n")
}
func (m *Model) searchMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	b := m.searchBounds()
	if msg.Action != tea.MouseActionPress {
		return *m, nil
	}
	if !b.contains(msg.X, msg.Y) {
		if msg.Button == tea.MouseButtonLeft {
			m.closeSearch()
		}
		return *m, nil
	}
	if msg.Button == tea.MouseButtonWheelUp {
		m.searchEditing = false
		m.search.Blur()
		m.searchMove(-1)
		return *m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		m.searchEditing = false
		m.search.Blur()
		m.searchMove(1)
		return *m, nil
	}
	if msg.Button != tea.MouseButtonLeft {
		return *m, nil
	}
	if msg.Y == b.y && msg.X >= b.x+b.w-14 {
		m.closeSearch()
		return *m, nil
	}
	if msg.Y == b.y+1 {
		m.searchEditing = true
		return *m, m.search.Focus()
	}
	_, zones := m.searchTabs()
	for i, z := range zones {
		if z.contains(msg.X, msg.Y) {
			m.searchTab = i
			m.searchEditing = false
			m.search.Blur()
			return *m, nil
		}
	}
	listY := b.y + 5
	if msg.Y >= listY && msg.Y < listY+m.searchListHeight() {
		offset := max(0, m.searchIndex[m.searchTab]-m.searchListHeight()+1)
		i := offset + msg.Y - listY
		if i < m.searchCount(m.searchTab) {
			m.searchIndex[m.searchTab] = i
			m.searchEditing = false
			m.search.Blur()
			return *m, m.openSearchResult()
		}
	}
	return *m, nil
}
