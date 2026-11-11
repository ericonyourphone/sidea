# Side A changes

## 0.1.0 development

- Introduce a cassette terminal interface with a separate library, queue,
  progress bar, and clickable transport controls.
- Reuse the upstream Spotify client, authentication, media support, and eight
  themes with a separate Side A config directory.
- Include native Mac playback and a live dotted FFT spectrum.
- Add track search, saved-album and playlist browsing, device selection,
  library pagination, and theme persistence.
- Add an offline interactive demo and reproducible terminal snapshots.
- Retain upstream license and attribution; remove inherited publishing
  automation that refers to the original project's releases.

- Distinguish restricted playlists, failed loads, and genuinely empty
  collections. Preserve the restriction message and offer whole-playlist
  playback with Enter when Spotify will not expose individual tracks.

- Open Spotify search in a centered popup with artist, song, and album
  categories, keyboard/mouse navigation, and preserved background state.

- Match the cassette reference with a pixel case, label, twin reels,
  raised Play/Pause/Previous/Next buttons, and a shared player panel.
- Distinguish the playing song from keyboard selection and show the current
  song above upcoming queue tracks without changing queue skip offsets.
- Replace the upright dotted analyzer with solid, centered spectrum bars.
