// Package ui is Side A's cassette-oriented terminal interface.
package ui

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/media"
	"github.com/ericonyourphone/sidea/source"
	"github.com/ericonyourphone/sidea/spectrum"
)

const pageSize = 50

type tickMsg time.Time
type pollMsg struct{}
type refreshMsg struct{}
type stateMsg struct {
	state *source.PlaybackState
	err   error
}
type libraryMsg struct {
	playlists []source.Playlist
	err       error
}
type queueMsg struct {
	tracks []source.Track
	key    string
	err    error
}
type rowsMsg struct {
	rows                      []row
	title, contextURI         string
	generation, offset, total int
	err                       error
}
type actionMsg struct{ err error }
type devicesMsg struct {
	devices []source.Device
	err     error
}

type row struct {
	track   source.Track
	albumID string
}
type libraryItem struct{ name, id, uri string }
type bounds struct{ x, y, w, h int }

func (b bounds) contains(x, y int) bool { return x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h }

// Model owns presentation state; Spotify calls run only in Bubble Tea commands.
type Model struct {
	searchOpen, searchEditing, searchLoading bool
	searchResults                            *source.SearchResults
	searchError                              error
	searchQuery                              string
	searchGeneration, searchTab              int
	searchIndex                              [3]int
	collectionErr                            error
	src                                      source.RichSource
	opts                                     app.Options
	ctx                                      context.Context
	cancel                                   context.CancelFunc
	width, height                            int
	track                                    *source.Track
	volume                                   int
	pollAt                                   time.Time
	pollPosition                             time.Duration
	polling, queueLoading                    bool
	cooldown                                 time.Time
	queue                                    []source.Track
	queueAt                                  time.Time
	library                                  []libraryItem
	libraryIndex                             int
	focusLibrary                             bool
	rows                                     []row
	rowIndex, rowOffset, total, generation   int
	browsing, loading                        bool
	collectionID, contextURI, title          string
	search                                   textinput.Model
	editing                                  string
	help, devicePicker                       bool
	devices                                  []source.Device
	deviceIndex                              int
	levels, peaks                            [spectrum.Bands]float64
	hold                                     [spectrum.Bands]time.Time
	lastFrame                                time.Time
	notice                                   string
	noticeUntil                              time.Time
	actionBusy                               bool
	demo                                     bool
	frame                                    int
}

func NewModel(src source.RichSource, opts app.Options) Model {
	ctx, cancel := context.WithCancel(context.Background())
	input := textinput.New()
	input.CharLimit = 160
	input.Prompt = "Search: "
	return Model{src: src, opts: opts, ctx: ctx, cancel: cancel, volume: 100, search: input,
		library: []libraryItem{{name: "Up next", id: "queue"}, {name: "Liked songs", id: "liked"}, {name: "Saved albums", id: "albums"}},
		title:   "UP NEXT", polling: true, queueLoading: true}
}
func (m Model) WithDemo() Model { m.demo = true; return m }
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchState(), m.fetchLibrary(), m.fetchQueue(), tick(), poll())
}
func tick() tea.Cmd {
	return tea.Tick(40*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func poll() tea.Cmd {
	return tea.Tick(2500*time.Millisecond, func(time.Time) tea.Msg { return pollMsg{} })
}
func (m Model) fetchState() tea.Cmd {
	return func() tea.Msg { s, e := m.src.CurrentPlayback(m.ctx); return stateMsg{s, e} }
}
func (m Model) fetchLibrary() tea.Cmd {
	return func() tea.Msg { p, e := m.src.Playlists(m.ctx); return libraryMsg{p, e} }
}
func (m Model) fetchQueue() tea.Cmd {
	key := ""
	if m.track != nil {
		key = m.track.ID
	}
	return func() tea.Msg { q, e := m.src.Queue(m.ctx); return queueMsg{q, key, e} }
}
func (m *Model) refresh() tea.Cmd {
	if time.Now().Before(m.cooldown) {
		return nil
	}
	var cmds []tea.Cmd
	if !m.polling {
		m.polling = true
		cmds = append(cmds, m.fetchState())
	}
	if !m.queueLoading {
		m.queueLoading = true
		cmds = append(cmds, m.fetchQueue())
	}
	return tea.Batch(cmds...)
}
func (m Model) control(fn func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		err := fn(m.ctx)
		// Starting a library track should work even when Spotify has no active
		// device yet. Only activate this player's own device, and only after a
		// user playback command receives Spotify's explicit no-device response.
		if !errors.Is(err, source.ErrNoActiveDevice) || m.opts.PlayerName == "" {
			return actionMsg{err}
		}
		devices, lookupErr := m.src.Devices(m.ctx)
		if lookupErr != nil {
			return actionMsg{lookupErr}
		}
		for _, device := range devices {
			if device.Name != m.opts.PlayerName {
				continue
			}
			if err = m.src.TransferPlayback(m.ctx, device.ID); err != nil {
				return actionMsg{err}
			}
			for attempt := 0; attempt < 2; attempt++ {
				select {
				case <-m.ctx.Done():
					return actionMsg{m.ctx.Err()}
				case <-time.After(time.Duration(attempt+1) * 350 * time.Millisecond):
				}
				err = fn(m.ctx)
				if !errors.Is(err, source.ErrPlaybackRestricted) && !errors.Is(err, source.ErrDeviceUnavailable) {
					break
				}
			}
			return actionMsg{err}
		}
		return actionMsg{err}
	}
}
func (m *Model) act(fn func(context.Context) error) tea.Cmd {
	if m.actionBusy || time.Now().Before(m.cooldown) {
		return nil
	}
	m.actionBusy = true
	return m.control(fn)
}
func (m *Model) notify(text string) { m.notice = text; m.noticeUntil = time.Now().Add(7 * time.Second) }
func (m *Model) failure(err error) {
	if err == nil {
		return
	}
	m.notify(err.Error())
	var status interface{ HTTPStatus() int }
	limited := errors.As(err, &status) && status.HTTPStatus() == 429 || strings.Contains(strings.ToLower(err.Error()), "rate limit")
	if limited {
		delay := 30 * time.Second
		var retry interface{ RetryAfterSeconds() int }
		if errors.As(err, &retry) {
			delay = max(delay, time.Duration(retry.RetryAfterSeconds())*time.Second)
		}
		m.cooldown = time.Now().Add(delay)
		m.notify("Spotify rate limit: waiting before retrying")
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.update(msg)
	return m, cmd
}
func (m *Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.searchOpen {
			m.search.Width = max(10, m.searchBounds().w-13)
		}
		return m, nil
	case tickMsg:
		m.frame++
		m.animate(time.Time(msg))
		return m, tick()
	case pollMsg:
		var cmd tea.Cmd
		if !m.polling && time.Now().After(m.cooldown) {
			m.polling = true
			cmd = m.fetchState()
		}
		return m, tea.Batch(cmd, poll())
	case refreshMsg:
		return m, m.refresh()
	case stateMsg:
		m.polling = false
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		old := ""
		if m.track != nil {
			old = m.track.ID
		}
		m.track = nil
		if msg.state != nil {
			m.volume = msg.state.Volume
			if msg.state.Track != nil {
				t := *msg.state.Track
				m.track = &t
				m.pollPosition = t.Position
				m.pollAt = time.Now()
			}
		}
		if m.opts.Media != nil {
			state := media.State{Empty: m.track == nil}
			if m.track != nil {
				t := m.track
				state = media.State{ID: t.ID, Title: t.Name, Artist: t.Artist, Album: t.Album, Duration: t.Duration, Position: t.Position, Playing: t.Playing}
			}
			m.opts.Media.Update(state)
		}
		if !m.queueLoading && time.Now().After(m.cooldown) && (m.track != nil && m.track.ID != old || time.Since(m.queueAt) > 30*time.Second) {
			m.queueLoading = true
			return m, m.fetchQueue()
		}
		return m, nil
	case libraryMsg:
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		m.library = m.library[:3]
		for _, p := range msg.playlists {
			if p.ID != "liked" {
				m.library = append(m.library, libraryItem{name: p.Name, id: p.ID, uri: p.URI})
			}
		}
		m.libraryIndex = min(m.libraryIndex, len(m.library)-1)
		return m, nil
	case queueMsg:
		m.queueLoading = false
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		current := ""
		if m.track != nil {
			current = m.track.ID
		}
		if current != msg.key {
			m.queueLoading = true
			return m, m.fetchQueue()
		}
		m.queue = msg.tracks
		m.queueAt = time.Now()
		if !m.browsing {
			m.rowIndex = min(m.rowIndex, max(0, len(m.queue)-1))
		}
		return m, nil
	case searchMsg:
		if !m.searchOpen || msg.generation != m.searchGeneration || msg.query != m.searchQuery {
			return m, nil
		}
		m.searchLoading = false
		m.searchError = msg.err
		m.searchResults = msg.results
		if msg.err != nil {
			m.failure(msg.err)
		}
		return m, nil
	case rowsMsg:
		if msg.generation != m.generation {
			return m, nil
		}
		m.loading = false
		m.collectionErr = msg.err
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		if msg.offset == 0 {
			m.rows = msg.rows
			m.rowIndex, m.rowOffset = 0, 0
		} else {
			m.rows = append(m.rows, msg.rows...)
		}
		m.total = msg.total
		m.title = msg.title
		m.contextURI = msg.contextURI
		return m, nil
	case actionMsg:
		m.actionBusy = false
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		return m, tea.Tick(650*time.Millisecond, func(time.Time) tea.Msg { return refreshMsg{} })
	case devicesMsg:
		if msg.err != nil {
			m.failure(msg.err)
			return m, nil
		}
		m.devices = msg.devices
		m.devicePicker = true
		m.deviceIndex = 0
		return m, nil
	case app.PlayerExitedMsg:
		m.failure(msg.Err)
		return m, nil
	case app.MediaMsg:
		switch msg.Command {
		case media.Toggle:
			return m, m.toggle()
		case media.Play:
			return m, m.act(m.src.Play)
		case media.Pause:
			return m, m.act(m.src.Pause)
		case media.Next:
			return m, m.act(m.src.Next)
		case media.Previous:
			return m, m.act(m.src.Previous)
		case media.Seek:
			return m, m.seek(msg.Position)
		}
	case tea.KeyMsg:
		return m.key(msg)
	case tea.MouseMsg:
		return m.mouse(msg)
	}
	if m.searchOpen && m.searchEditing || m.editing != "" {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) toggle() tea.Cmd {
	if m.track != nil && m.track.Playing {
		return m.act(m.src.Pause)
	}
	return m.act(m.src.Play)
}
func (m *Model) seek(pos time.Duration) tea.Cmd {
	if m.track == nil {
		return nil
	}
	pos = max(0, min(pos, m.track.Duration))
	return m.act(func(ctx context.Context) error { return m.src.Seek(ctx, pos) })
}
func (m Model) position() time.Duration {
	if m.track == nil {
		return 0
	}
	p := m.pollPosition
	if m.track.Playing && !m.pollAt.IsZero() {
		p += time.Since(m.pollAt)
	}
	return max(0, min(p, m.track.Duration))
}
func (m *Model) theme() tea.Cmd {
	names := app.ThemeNames()
	name := app.CurrentPalette()
	at := 0
	for i, n := range names {
		p, _ := app.ResolveTheme(n, m.opts.ColorOverrides)
		if p == name {
			at = i
			break
		}
	}
	return m.setTheme(names[(at+1)%len(names)])
}
func (m *Model) setTheme(name string) tea.Cmd {
	p, err := app.ResolveTheme(name, m.opts.ColorOverrides)
	if err != nil {
		m.failure(err)
		return nil
	}
	app.ApplyTheme(p)
	if m.opts.SaveTheme != nil {
		if err = m.opts.SaveTheme(name); err != nil {
			m.failure(err)
			return nil
		}
	}
	m.notify("Theme: " + name)
	return nil
}
func (m *Model) local() tea.Cmd {
	name := m.opts.PlayerName
	if name == "" {
		m.notify("Set up local playback with: sidea player login")
		return nil
	}
	return m.act(func(ctx context.Context) error {
		devices, err := m.src.Devices(ctx)
		if err != nil {
			return err
		}
		for _, d := range devices {
			if d.Name == name {
				return m.src.TransferPlayback(ctx, d.ID)
			}
		}
		return errors.New("Side A is still connecting; press M again in a moment")
	})
}
func (m *Model) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if k == "ctrl+c" {
		m.cancel()
		return *m, tea.Quit
	}
	if m.searchOpen {
		return m.searchKey(msg)
	}
	if m.editing != "" {
		switch k {
		case "esc":
			m.editing = ""
			m.search.Blur()
			return *m, nil
		case "enter":
			query := strings.TrimSpace(m.search.Value())
			kind := m.editing
			m.editing = ""
			m.search.Blur()
			if query == "" {
				return *m, nil
			}
			if kind == "command" {
				if strings.HasPrefix(query, "theme ") {
					return *m, m.setTheme(strings.TrimSpace(strings.TrimPrefix(query, "theme ")))
				}
				if query == "queue" {
					m.browsing = false
					m.focusLibrary = false
					return *m, m.refresh()
				}
				if query == "help" {
					m.help = true
					return *m, nil
				}
				m.notify("Commands: theme <name>, queue, help")
				return *m, nil
			}
			return *m, nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return *m, cmd
	}
	if m.help {
		if k == "?" || k == "esc" || k == "q" {
			m.help = false
		}
		return *m, nil
	}
	if m.devicePicker {
		switch k {
		case "esc", "q":
			m.devicePicker = false
		case "up", "k":
			m.deviceIndex = max(0, m.deviceIndex-1)
		case "down", "j":
			m.deviceIndex = min(max(0, len(m.devices)-1), m.deviceIndex+1)
		case "enter":
			if len(m.devices) > 0 {
				id := m.devices[m.deviceIndex].ID
				m.devicePicker = false
				return *m, m.act(func(ctx context.Context) error { return m.src.TransferPlayback(ctx, id) })
			}
		}
		return *m, nil
	}
	switch k {
	case "q":
		m.cancel()
		return *m, tea.Quit
	case "?":
		m.help = true
	case " ":
		return *m, m.toggle()
	case "n":
		return *m, m.act(m.src.Next)
	case "p":
		return *m, m.act(m.src.Previous)
	case "M":
		return *m, m.local()
	case "D":
		return *m, func() tea.Msg { d, e := m.src.Devices(m.ctx); return devicesMsg{d, e} }
	case "t":
		return *m, m.theme()
	case "/":
		return *m, m.openSearch()
	case ":":
		m.search.SetValue("")
		m.search.Width = max(10, m.width-16)
		m.editing = "command"
		m.search.Prompt = ":"
		return *m, m.search.Focus()
	case "tab", "h", "left", "l", "right":
		if k == "h" || k == "left" {
			m.focusLibrary = true
		} else if k == "l" || k == "right" {
			m.focusLibrary = false
		} else {
			m.focusLibrary = !m.focusLibrary
		}
	case "up", "k":
		m.move(-1)
	case "down", "j":
		return *m, m.move(1)
	case "pgdown", "ctrl+d":
		return *m, m.move(8)
	case "pgup", "ctrl+u":
		m.move(-8)
	case "enter":
		if m.focusLibrary {
			return *m, m.openLibrary()
		}
		return *m, m.openRow()
	case "esc", "Q":
		m.browsing = false
		m.focusLibrary = false
		m.rowIndex, m.rowOffset = 0, 0
		return *m, m.refresh()
	case "]":
		return *m, m.seek(m.position() + 5*time.Second)
	case "[":
		return *m, m.seek(m.position() - 5*time.Second)
	case "+", "=", "-":
		volume := min(100, m.volume+5)
		if k == "-" {
			volume = max(0, m.volume-5)
		}
		return *m, m.act(func(ctx context.Context) error { return m.src.SetVolume(ctx, volume) })
	case "a":
		rows := m.visibleRows()
		if !m.focusLibrary && m.rowIndex < len(rows) && rows[m.rowIndex].track.ID != "" {
			id := rows[m.rowIndex].track.ID
			return *m, m.act(func(ctx context.Context) error { return m.src.AddToQueue(ctx, id) })
		}
	}
	return *m, nil
}

func (m Model) visibleRows() []row {
	if m.browsing {
		return m.rows
	}
	r := make([]row, len(m.queue))
	for i, t := range m.queue {
		r[i] = row{track: t}
	}
	return r
}
func (m *Model) move(n int) tea.Cmd {
	if m.focusLibrary {
		m.libraryIndex = max(0, min(len(m.library)-1, m.libraryIndex+n))
		return nil
	}
	count := len(m.visibleRows())
	m.rowIndex = max(0, min(max(0, count-1), m.rowIndex+n))
	if m.browsing && !m.loading && len(m.rows) < m.total && m.rowIndex >= len(m.rows)-5 && m.collectionID != "search" && m.collectionID != "albums" && !strings.HasPrefix(m.collectionID, "album:") && !strings.HasPrefix(m.collectionID, "artist:") {
		m.loading = true
		return m.fetchCollection(len(m.rows))
	}
	return nil
}
func (m *Model) openLibrary() tea.Cmd {
	if time.Now().Before(m.cooldown) {
		return nil
	}
	item := m.library[m.libraryIndex]
	m.focusLibrary = false
	m.rowIndex, m.rowOffset = 0, 0
	if item.id == "queue" {
		m.browsing = false
		return m.refresh()
	}
	m.generation++
	m.collectionErr = nil
	m.browsing, m.loading = true, true
	m.collectionID = item.id
	m.contextURI = item.uri
	m.title = item.name
	m.rows = nil
	m.total = 0
	return m.fetchCollection(0)
}
func (m Model) fetchCollection(offset int) tea.Cmd {
	id, title, uri, gen := m.collectionID, m.title, m.contextURI, m.generation
	return func() tea.Msg {
		var rows []row
		var err error
		total := 0
		switch {
		case id == "albums":
			var albums []source.SavedAlbum
			albums, err = m.src.SavedAlbums(m.ctx)
			for _, a := range albums {
				rows = append(rows, row{track: source.Track{Name: a.Name, Artist: a.Artist}, albumID: a.ID})
			}
			total = len(rows)
		case strings.HasPrefix(id, "artist:"):
			var artist *source.ArtistPage
			artist, err = m.src.GetArtist(m.ctx, strings.TrimPrefix(id, "artist:"))
			if artist != nil {
				title = artist.Name
				uri = "spotify:artist:" + artist.ID
				for _, t := range artist.Tracks {
					rows = append(rows, row{track: t})
				}
				for _, a := range artist.Albums {
					rows = append(rows, row{track: source.Track{Name: a.Name, Artist: artist.Name}, albumID: a.ID})
				}
				total = len(rows)
			}
		case strings.HasPrefix(id, "album:"):
			var album *source.AlbumPage
			album, err = m.src.GetAlbum(m.ctx, strings.TrimPrefix(id, "album:"))
			if album != nil {
				for _, t := range album.Tracks {
					rows = append(rows, row{track: t})
				}
				title = album.Name
				uri = "spotify:album:" + album.ID
				total = len(rows)
			}
		default:
			var tracks []source.Track
			if id == "liked" {
				tracks, total, err = m.src.LikedTracksPage(m.ctx, offset, pageSize)
			} else {
				tracks, total, err = m.src.PlaylistTracksPage(m.ctx, id, offset, pageSize)
			}
			for _, t := range tracks {
				rows = append(rows, row{track: t})
			}
		}
		return rowsMsg{rows: rows, title: title, contextURI: uri, generation: gen, offset: offset, total: total, err: err}
	}
}
func (m *Model) openRow() tea.Cmd {
	if m.browsing && m.collectionErr != nil && !m.loading && len(m.rows) == 0 {
		if errors.Is(m.collectionErr, source.ErrPlaylistRestricted) && m.contextURI != "" {
			uri := m.contextURI
			return m.act(func(ctx context.Context) error { return m.src.PlayTrack(ctx, uri, "") })
		}
		if m.collectionID == "search" {
			m.notify("Press / to retry your search")
			return nil
		}
		m.collectionErr = nil
		m.loading = true
		return m.fetchCollection(0)
	}
	rows := m.visibleRows()
	if m.rowIndex >= len(rows) || m.loading {
		return nil
	}
	r := rows[m.rowIndex]
	if r.albumID != "" {
		m.generation++
		m.collectionErr = nil
		m.collectionID = "album:" + r.albumID
		m.loading = true
		m.rows = nil
		return m.fetchCollection(0)
	}
	if !m.browsing {
		steps := m.rowIndex + 1
		return m.act(func(ctx context.Context) error {
			for i := 0; i < steps; i++ {
				if err := m.src.Next(ctx); err != nil {
					return err
				}
				if i < steps-1 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(180 * time.Millisecond):
					}
				}
			}
			return nil
		})
	}
	uri := r.track.URI
	if uri == "" {
		uri = "spotify:track:" + r.track.ID
	}
	contextURI := m.contextURI
	return m.act(func(ctx context.Context) error {
		if contextURI != "" && m.collectionID != "liked" {
			return m.src.PlayTrack(ctx, contextURI, uri)
		}
		return m.src.PlayTrackDirect(ctx, uri)
	})
}

func (m *Model) animate(now time.Time) {
	frame := spectrum.Default.Snapshot()
	local := m.track != nil && m.track.Playing && m.track.DeviceName == m.opts.PlayerName && m.opts.PlayerName != ""
	fresh := local && !frame.Updated.IsZero() && now.Sub(frame.Updated) < 300*time.Millisecond
	dt := .04
	if !m.lastFrame.IsZero() {
		dt = max(0, min(.15, now.Sub(m.lastFrame).Seconds()))
	}
	m.lastFrame = now
	for b := range spectrum.Bands {
		target := 0.0
		if fresh {
			target = frame.Levels[b]
		}
		if m.demo && m.track != nil && m.track.Playing {
			target = .15 + .65*math.Abs(math.Sin(float64(m.frame)*.025+float64(b)*.24))*math.Exp(-float64(b)/60)
		}
		m.levels[b] = max(target, m.levels[b]-dt*1.5)
		if m.levels[b] >= m.peaks[b] {
			m.peaks[b] = m.levels[b]
			m.hold[b] = now.Add(300 * time.Millisecond)
		} else if now.After(m.hold[b]) {
			m.peaks[b] = max(m.levels[b], m.peaks[b]-dt*.7)
		}
	}
}

func (m Model) emptyState() []string {
	if !m.browsing {
		if m.queueLoading {
			return []string{"Loading queue…"}
		}
		return []string{"Queue is empty"}
	}
	if m.collectionErr != nil {
		switch {
		case errors.Is(m.collectionErr, source.ErrPlaylistRestricted):
			return []string{"Spotify restricts this track list.", "Enter: play this playlist", "Q: show the playback queue"}
		case errors.Is(m.collectionErr, source.ErrInsufficientScope):
			return []string{"Spotify needs library permission.", "Run: sidea auth"}
		case errors.Is(m.collectionErr, source.ErrForbidden):
			return []string{"Spotify denied access to these tracks.", "Enter: retry"}
		default:
			return []string{"Could not load tracks.", "Enter: retry"}
		}
	}
	switch m.collectionID {
	case "search":
		return []string{"No matching tracks found"}
	case "albums":
		return []string{"No saved albums found"}
	default:
		return []string{"This collection has no tracks"}
	}
}
