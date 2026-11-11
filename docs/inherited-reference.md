# Inherited interface reference

This documentation is retained for the legacy `app/` package and its
documentation checks. For Side A’s actual interface and controls, use the
project’s top-level README.md.

---

<p align="center">
  <img src="assets/logo.png" width="128" alt="sidea logo">
</p>

<h1 align="center">sidea</h1>

<p align="center">
  A vim-modal Spotify client for the terminal.
</p>

<p align="center">
  <a href="#install">Install</a> &middot;
  <a href="#features">Features</a> &middot;
  <a href="#keybindings">Keybindings</a> &middot;
  <a href="#commands">Commands</a> &middot;
  <a href="#privacy--network">Privacy</a>
</p>

<p align="center">
  <img src="demo/recordings/full-tour.gif" alt="sidea demo" width="800">
</p>

Browse your library, search, queue and play without leaving the home row.

- **Plays by itself or drives the Spotify app.** sidea can be the player (no
  Spotify app needed), or the remote for your desktop app, phone or speakers;
  `M` switches between the two, carrying on from the same point.
- **Vim all the way down:** `j`/`k`, counts, `gg`/`G`, visual mode, `:`
  commands, `/` to filter anything.
- **Now Playing** with full-screen album art and
  time-synced lyrics.
- **Media keys** and the system's Now Playing (Control Center, MPRIS).
- **Scriptable:** every action is also a subcommand, for tmux, waybar and
  hotkeys.
- **Talks only to Spotify**, plus lrclib for lyrics and a daily check for a
  new release, both of which you can turn off. No telemetry.

Requires **Spotify Premium**. Runs on macOS, Linux and Windows.

## Install

See the top-level README for how to build and install Side A.

## Quick Start

```bash
sidea
```

The first run is a guided setup of about a minute: connect your Spotify
account in the browser, then choose where music plays (your Spotify apps, or
right here in sidea). Press `?` for help at any time.

<p align="center">
  <img src="demo/recordings/setup.gif" alt="guided setup" width="700">
</p>

Something not working? `sidea doctor` checks everything and says what to fix.

## Playing & Devices

sidea plays music in one of two ways, and `M` switches between them on this
computer, carrying on from the same point:

- **In sidea.** sidea runs its own Spotify Connect device, `sidea (<host>)`,
  through [librespot](https://github.com/librespot-org/librespot): no Spotify
  app needed. Setup offers to install librespot with your package manager
  (Homebrew, pacman, Nix or cargo) and signs it in; after that it starts in
  the background whenever nothing else is playing.
- **As a remote** for the Spotify app on this computer, or your phone and
  speakers: nothing to install. If the Spotify app is installed but closed,
  `M` opens it and hands playback over once it's online.

`D` picks any device on your account (its **play here with sidea** row sets
up sidea's player if you skipped it), `:device <name>` switches by name, and
`:device -` goes back to the previous one. If nothing is playing when you
press play, sidea picks for you: its own player when it's set up, the only
device when there's one, or it asks.

<p align="center">
  <img src="demo/recordings/no-device.gif" alt="picking a device when none is active" width="800">
</p>

Setup can be re-run or changed at any time:

```bash
sidea auth            # guided setup again
sidea auth --own      # set up / switch to your own Spotify app
sidea auth --shared   # use the shared app
sidea auth --client-id <ID>
sidea player          # is sidea's own player set up?
sidea player login    # set it up (or sign it in again)
sidea player off      # stop using it
sidea logout          # forget the saved logins
```

**Your own Spotify app or the shared one.** Setup recommends creating your own
(free) Spotify developer app; sidea lists the exact clicks, opens the dashboard
and checks the Client ID as you paste it. The shared one needs no setup, but
Spotify throttles it for everyone using it. See
[Using your own Spotify app](#using-your-own-spotify-app).

**Over SSH** (or whenever the browser is on another machine): open the sign-in
link anywhere, approve, and paste the address the browser lands on into
sidea. That page won't load, which is expected; sidea only needs its address.

> **About the player login.** Spotify only lets its own apps stream audio, so
> librespot — an open-source, unofficial Spotify Connect client (it also powers
> spotifyd, raspotify and ncspot) — signs in as the Spotify desktop app, and
> asks for the same broad permissions that app does. You approve on Spotify's
> own sign-in page; your password never reaches sidea or librespot. The
> resulting login token is saved only on your machine, in sidea's config
> directory (`~/.config/sidea/librespot/`, or `%AppData%\sidea\librespot\` on
> Windows), readable by you alone, and is sent nowhere but Spotify. Being
> unofficial, librespot could stop working if Spotify changes something; the
> player is opt-in for that reason.

## Features

### Vim Navigation

Navigate everything without leaving the home row — `j`/`k` to move, `gg`/`G` to jump, `h`/`l` to switch panes.

<p align="center">
  <img src="demo/recordings/navigation.gif" alt="vim navigation" width="800">
</p>

### Small Terminals

Below 80 columns sidea shows one pane at a time — `h`/`l`/`Tab` switch between library and tracks — and the track table drops columns rather than overflowing. Now Playing art scales to fit. Handy for a tmux side pane.

<p align="center">
  <img src="demo/recordings/narrow.gif" alt="narrow terminal layout" width="480">
</p>

### Now Playing

Full-screen album art rendered with Unicode half-blocks over a gradient drawn from the cover, with synced lyrics a key away. (Try `v`, too.)

<p align="center">
  <img src="demo/recordings/nowplaying.gif" alt="now playing view" width="800">
</p>

### Media Keys

The keyboard's ⏯ ⏭ ⏮ keys control sidea, and the system's Now Playing (Control Center on macOS; GNOME, KDE and `playerctl` through MPRIS on Linux) shows the track, its cover and progress, with working buttons and scrubbing. They control whatever sidea is playing on: its own player, your phone or a speaker. If another music app is open it may take the keys first. Over SSH they're left alone: the keys belong to whoever is at the machine.

### Queue

`gq` shows the queue in the main pane, as the Spotify app does: **Now playing**, **Next in queue** (the tracks you added) and **Next from** the playlist (or **Up next**, when sidea can't tell them apart, as with shuffle on). Enter on a track skips forward to it, so the rest of the queue stays; the view follows along as tracks change.

<p align="center">
  <img src="demo/recordings/queue.gif" alt="the queue" width="800">
</p>

### Synced Lyrics

Press `l` in Now Playing (or run `:lyrics` from anywhere) for time-synced lyrics (via [lrclib](https://lrclib.net) — no account, no API key, nothing to set up). The current line lights up and the rest gently fade as the song plays.

<p align="center">
  <img src="demo/recordings/lyrics.gif" alt="time-synced lyrics" width="800">
</p>

### Playlists

`o` on any track offers **Add to Playlist…** (type to filter your playlists) and, inside one of your own playlists, **Remove from …**. Create a new one with `:playlist new <name>`. Changes show up immediately in the sidebar and the open playlist.

<p align="center">
  <img src="demo/recordings/playlists.gif" alt="adding a track to a playlist" width="800">
</p>

### Search

Find tracks, artists, and albums across Spotify.

<p align="center">
  <img src="demo/recordings/search.gif" alt="search" width="800">
</p>

### Artist & Album Browsing

Artist pages open with **Play** and **Shuffle**, then the popular tracks and the albums and singles; albums open in full. `ga` and `gA` jump to the selected track's album or artist, and `b` goes back, like a browser.

<p align="center">
  <img src="demo/recordings/browsing.gif" alt="artist and album browsing" width="800">
</p>

### Your Library

Under **Liked Songs** in the Library, **Albums · Artists · Podcasts** opens Recently Played and the albums, artists and shows you've saved and follow (also `:recent`, `:albums`, `:artists`, `:podcasts`). Enter opens one, and `/` filters by name or artist.

<p align="center">
  <img src="demo/recordings/library.gif" alt="saved albums and followed artists" width="800">
</p>

### Podcasts

**Podcasts** (under Albums · Artists · Podcasts, or `:podcasts`) lists the shows you follow. Each opens to its episodes, newest first; episodes play, queue, like and add to playlists like tracks do.

<p align="center">
  <img src="demo/recordings/podcasts.gif" alt="podcasts" width="800">
</p>

### Command Mode

Vim-style commands for volume, shuffle, repeat, device switching, and more.

<p align="center">
  <img src="demo/recordings/commands.gif" alt="command mode" width="800">
</p>

## Keybindings

### Navigation

| Key              | Action              |
|------------------|---------------------|
| `j` / `k`        | Move down / up      |
| `gg`             | Go to top           |
| `G`              | Go to bottom        |
| `Ctrl+u` / `Ctrl+d` | Half page up / down |
| `5j` / `12G` / `12gg` / `6]` | Counts, as in vim |
| `Esc`            | Clear a filter or a half-typed count |

### Panes

| Key              | Action              |
|------------------|---------------------|
| `h` / `l`        | Focus left / right pane |
| `Tab`            | Cycle pane          |

### Go-to (g prefix)

| Key   | Action                      |
|-------|-----------------------------|
| `gl`  | Go to library               |
| `gq`  | The queue: now playing, what you queued, and what plays next from the playlist; Enter skips forward to a track |
| `gc`  | Jump to currently playing track (loads its playlist/album if you've navigated away) |
| `ga`  | Album of the selected track (in Now Playing: the playing one) |
| `gA`  | Artist of the selected track |

### Playback

| Key              | Action              |
|------------------|---------------------|
| `Space`          | Play / pause        |
| `Enter`          | Play selected       |
| `n` / `p`        | Next / previous track |
| `[` / `]`        | Seek -5s / +5s (`6]` is +30s) |
| `-` / `+`        | Volume -5 / +5      |
| `m`              | Mute / unmute       |
| `S`              | Toggle shuffle      |
| `r`              | Cycle repeat        |

### Actions

| Key   | Action              |
|-------|---------------------|
| `o`   | Actions menu for the selection (each row has a number; `a`, `f`, `+`, `d` work in it too) |
| `O`   | Actions menu for the **playing** track, from anywhere |
| `a`   | Add the selection to the queue |
| `f`   | Like / unlike the selection |
| `F`   | Like / unlike the **playing** track, from anywhere |
| `v`   | Visual mode (select a range) |
| `/`   | Filter current view |
| `s`   | Spotify search      |
| `D`   | Device switcher     |
| `M`   | Play in sidea ⇄ control the Spotify app |
| `:`   | Command mode        |

### Visual mode

`v` in the track list starts a selection; move with `j`/`k` (counts work) and act on every selected track:

| Key   | Action                         |
|-------|--------------------------------|
| `a`   | Queue them, in order           |
| `f`   | Like them                      |
| `o`   | Add them to a playlist         |
| `d`   | Remove them from your playlist |
| `Esc` / `v` | Cancel                   |

### Other

| Key              | Action              |
|------------------|---------------------|
| `N`              | Now Playing view (playback, `s`, `D`, `M`, `:` and `?` work in it too) |
| `f` / `a` / `o`  | Like / queue / actions for the **playing** track (in Now Playing) |
| `v`              | Toggle vinyl mode (in Now Playing) |
| `l`              | Toggle synced lyrics (in Now Playing) |
| `Backspace` / `b` | Go back           |
| `?`              | Help: the essentials, then `Tab` for every key and command |
| `,`              | Settings: theme, album art, media keys, lyrics, update check, saved as you change them |
| `q` / `Ctrl+c`   | Quit               |
| `Esc`            | Close / cancel      |

In search, `Tab` moves from the query to the results, where `j`/`k` move and `a`, `f` and `o` act on the selected track; `Tab` or `/` goes back to typing.

## Commands

Enter command mode by pressing `:`, then type a command. As you type, the likeliest command shows dimmed after the cursor (and then what it takes); `Tab` or `→` accepts it. The playback commands read like the CLI's (`sidea next`, `sidea seek 1:30`).

| Command                 | Description          |
|-------------------------|----------------------|
| `:play` / `:pause` / `:toggle` | Resume / pause / play-pause |
| `:next` / `:prev`       | Next / previous track |
| `:seek ±N` / `:seek M:SS` | Seek by N seconds, or to a time |
| `:vol <0-100>` / `:vol ±N` | Set or change the volume |
| `:mute`                 | Mute / unmute        |
| `:like [on\|off]`       | Like / unlike the playing track |
| `:shuffle [on\|off]`    | Shuffle              |
| `:repeat off\|all\|one` | Set repeat mode      |
| `:device`               | Open device switcher |
| `:device -`             | Back to the previous device |
| `:device <name>`        | Switch to the device whose name contains `<name>` |
| `:search <query>`       | Search Spotify       |
| `:recent`               | Recently played      |
| `:albums`               | Saved albums         |
| `:artists`              | Followed artists     |
| `:podcasts`             | Podcasts             |
| `:lyrics`               | Now Playing, on the synced lyrics |
| `:vinyl`                | Now Playing, in vinyl mode |
| `:theme <name>`         | Switch colour theme (saved to config) |
| `:playlist new <name>`  | Create a playlist    |
| `:settings`             | Settings             |
| `:h`                    | Help                 |
| `:q`                    | Quit                 |

## Configuration

Press `,` (or `:settings`) to change the theme, album art, media keys, lyrics and the update check from inside sidea; each change is saved to the config file as you make it.

sidea reads `~/.config/sidea/config.json` — `%AppData%\sidea\config.json` on Windows, `$XDG_CONFIG_HOME/sidea` when that's set; `sidea config` prints the path. Everything is optional:

```json
{
  "theme": "catppuccin-mocha",
  "colors": { "accent": "#F5C2E7" },
  "keys":   { "next": "n,>", "prev": "p,<", "quit": "ctrl+q" }
}
```

**Themes** — built in: `spotify` (default), `catppuccin-mocha`, `catppuccin-latte`, `gruvbox`, `tokyonight`, `nord`, `dracula`, `rose-pine` (`sidea themes` lists them; sidea doesn't paint the terminal background, so pick a light theme like `catppuccin-latte` for a light terminal). Switch live with `:theme <name>` — it's saved to your config. `colors` overrides individual palette entries on top of any theme: `accent`, `bg`, `surface`, `text`, `text_sec`, `text_dim`, `border`, `error`, `mode_search`, `mode_filter`, `overlay`.

<p align="center">
  <img src="demo/recordings/themes.gif" alt="switching themes with :theme" width="800">
</p>

**Keys** — `keys` maps an action to a comma-separated list of keys (Bubbletea names: `j`, `down`, `ctrl+d`, `space`, `enter`, `esc`, `tab`, `backspace`, `f1`…). Actions: `up down bottom half_up half_down focus_left focus_right cycle_pane enter play_pause next prev seek_fwd seek_back vol_up vol_down mute shuffle repeat visual add_queue like like_playing actions actions_playing devices mode back filter search command help now_playing settings quit`. The `g`-prefix motions (`gg gl gq gc ga gA`) and `Esc` are fixed. The `?` help overlay always shows your live bindings. If a custom key is misspelt, has no keys, or is already another action's, sidea says so when it starts and uses the rest.

**Player** — `"player": "librespot"` (set by `sidea player login`) runs sidea's own playback device; `"player_path"` points at a librespot binary that isn't on `PATH`. librespot's log goes to the `SIDEA_LOG` file when set.

**Lyrics** — `"lyrics": false` turns synced lyrics off (see [Privacy & Network](#privacy--network)).

**Update check** — once a day at most, when it starts, sidea asks GitHub for the latest release; if it's newer, sidea says so and, as it quits, prints the command that upgrades it (`brew upgrade sidea`, `scoop update sidea` or `go install …`, by how it was installed). `"update_check": false` turns it off.

**Media keys** — on by default on macOS and Linux; `"media_keys": false` turns them and the Now Playing entry off.

**Art** — `"art": "kitty"` draws the status-bar and Now Playing covers as real images using the kitty graphics protocol (kitty, Ghostty); `"auto"` does that only when the terminal is detected as supporting it (not inside tmux), and `"blocks"` (the default) keeps the Unicode half-block art. `SIDEA_ART` overrides the setting for one run, e.g. `SIDEA_ART=kitty sidea`. Experimental: please report terminals where it misbehaves.

**Cache** — album art is cached under `$XDG_CACHE_HOME/sidea/images` (or your OS cache dir), capped at 64 MB and pruned on launch, so relaunching paints the library instantly. Delete the directory to clear it.

**Colour fallback** — colours degrade automatically on 256-colour terminals and are dropped under `NO_COLOR`; album art uses the 256-colour cube where true colour isn't available and is skipped entirely on monochrome terminals.

## Scripting & Status Bars

Every playback action is also a plain subcommand, so sidea slots into tmux,
waybar/polybar, hotkey daemons and shell scripts without opening the TUI.

```
sidea status                      # ▶ Let It Happen — Tame Impala
sidea status --json               # {"playing":true,"title":...,"progress":37,...}
sidea status --waybar             # {"text":"…","alt":"playing","class":"playing",...}
sidea status --format '{icon} {title} [{position}/{duration}]'
sidea play | pause | toggle | next | prev
sidea play daft punk get lucky    # search, then play the first match
sidea seek +10 | seek -10 | seek 1:30
sidea vol 40 | vol +5 | vol -5
sidea shuffle [on|off] | repeat off|all|one
sidea like                        # toggle Liked Songs for the playing track
sidea queue instant crush
sidea devices | device "living room"
sidea search radiohead [--json]
```

`status` prints nothing when idle (so bars stay blank) and exits non-zero on
errors; `--json` always emits an object. Placeholders for `--format`:
`{title} {artist} {album} {position} {duration} {progress} {state} {icon}
{device} {volume} {shuffle} {repeat} {liked} {uri} {id}`. If no Spotify device
is active, commands activate the only available one automatically, or tell you
which ones to choose from.

<p align="center">
  <img src="demo/recordings/cli.gif" alt="sidea CLI subcommands" width="800">
</p>

**tmux** (`~/.tmux.conf`):

```
set -g status-right '#(sidea status --format "{icon} {title} — {artist}") | %H:%M'
set -g status-interval 5
```

**waybar** (`~/.config/waybar/config`) — `--waybar` emits the
`text/alt/class/tooltip/percentage` object waybar expects:

```json
"custom/spotify": {
  "exec": "sidea status --waybar",
  "return-type": "json",
  "format": "{icon} {}",
  "format-icons": {"playing": "", "paused": "", "idle": ""},
  "on-click": "sidea toggle",
  "on-scroll-up": "sidea vol +5",
  "on-scroll-down": "sidea vol -5",
  "interval": 5
}
```

**Hotkeys** (skhd on macOS / sxhkd on Linux):

```
cmd + alt - space : sidea toggle
cmd + alt - right : sidea next
cmd + alt - left  : sidea prev
cmd + alt - l     : sidea like
```

## Privacy & Network

sidea has no telemetry and no analytics. It connects to:

| Host | For |
|---|---|
| `accounts.spotify.com`, `api.spotify.com` | Signing in, and everything you do (Spotify's Web API) |
| Spotify's image servers (`*.scdn.co`) | Album and playlist covers |
| `lrclib.net` | Synced lyrics: the track's title, artist, album and length are sent to look them up |
| `api.github.com` | Once a day at most, at launch: the latest sidea release, to say when there's a newer one. Nothing is sent but the request |
| Spotify's streaming servers | Only sidea's own player (librespot), when you use it |

Sign-in uses PKCE: there's no client secret, and your password goes only to
Spotify's own sign-in page. Logins and settings are stored in sidea's config
directory (`sidea config` prints it), readable by you alone.

**Talking only to Spotify.** `"lyrics": false` in `config.json` stops sidea
contacting lrclib, and `"update_check": false` stops it contacting GitHub, so
it talks only to Spotify (useful on a work network). If a
proxy or firewall blocks lrclib, sidea leaves it alone for ten minutes instead
of asking on every track; `l` tries again sooner.

**Proxies.** sidea uses `HTTPS_PROXY` / `NO_PROXY` when they're set; on macOS
it otherwise picks up the HTTPS proxy from System Settings, as other apps do.
A proxy chosen by a PAC script can't be followed: set `HTTPS_PROXY` yourself
(`sidea doctor` says when that's needed), or `HTTPS_PROXY=` (empty) to ignore
macOS's. sidea's own player (librespot) doesn't use these settings.

## Using Your Own Spotify App

sidea works out of the box with no configuration — it ships with a shared client ID used by several open-source Spotify clients. Spotify rate-limits that shared app as a whole, so at busy times you may see *Spotify rate limit* toasts or sluggish controls. A personal client ID has its own quota and takes two minutes to set up:

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard) and create an app
2. Set the redirect URI to `http://127.0.0.1:27228/callback`
3. Copy the **Client ID** and run setup with it:

   ```
   SPOTIFY_CLIENT_ID=your_client_id sidea auth
   ```

The client ID is saved to sidea's `config.json` (see `sidea config`) automatically, so you only need to set the environment variable once during setup.

> **Note:** new developer apps are in Spotify's *development mode*, which only allows the current API endpoints. sidea uses those (with legacy fallbacks for older apps), so playback, search, queue, liking and playlist editing all work. Spotify blocks two things for development-mode apps with no replacement: an artist's *top tracks* (artist pages show the discography instead), and the track list of playlists you don't own or collaborate on, including Spotify's own editorial playlists (sidea says so; Enter still plays the playlist, and `o` can open it in Spotify).

## Troubleshooting

Start with `sidea doctor`. It checks your config, Spotify app, login (and
whether Spotify accepts it), devices, sidea's player, the lyrics service and
your terminal, and prints the fix for anything wrong.

| Symptom | Cause / fix |
|---|---|
| *Spotify rate limit* / *Spotify is rate limiting* toasts | Spotify is throttling the app. sidea backs off automatically and honours Spotify's `Retry-After`. If it keeps happening, the shared client ID is busy — [use your own](#using-your-own-spotify-app). |
| *No active Spotify device* | Spotify must be open somewhere (desktop, phone, speaker), or let sidea play by itself: `D` → **play here with sidea**. sidea picks the only available device automatically or asks with `D`. |
| *Not available with this Spotify app* | Spotify blocks that endpoint for development-mode (personal) apps — currently only artists' top tracks. |
| A followed playlist shows no tracks | Spotify only lets development-mode (personal) apps list playlists you own or collaborate on. Press Enter to play it anyway, or `o` to open it in Spotify. |
| *Permission needed — run `sidea auth`* | Your saved token predates a feature that needs extra permissions (e.g. playlist editing). Re-run `sidea auth` once. |
| *Session expired* | Token revoked or refresh failed. Re-run `sidea auth`. |
| Nothing renders / garbled colours | Set a true-colour or 256-colour `TERM`, or `NO_COLOR=1` for monochrome. Minimum size is 40×10. |

Debug log: `SIDEA_LOG=/tmp/sidea.log sidea` (includes rate-limit `Retry-After` values).

## Environment Variables

| Variable            | Description                        |
|---------------------|------------------------------------|
| `SPOTIFY_CLIENT_ID` | Override the saved Spotify Client ID |
| `SIDEA_LOG`         | Path to debug log file (e.g. `/tmp/sidea.log`) |
| `SIDEA_ART`         | Override the `art` setting for one run (`blocks`, `kitty`, `auto`) |
| `SIDEA_NO_BROWSER`  | Don't launch a browser during `sidea auth`; just print the URL (SSH) |

## Releasing

Maintainers cut releases by following [RELEASING.md](RELEASING.md).

## Acknowledgements

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), and [Lip Gloss](https://github.com/charmbracelet/lipgloss) by [Charmbracelet](https://charm.sh). Demo recordings made with [VHS](https://github.com/charmbracelet/vhs). Huge thanks to the Charm team for making terminal UIs a joy to build.

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
