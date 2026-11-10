# Security notes

Side A is an early local development build derived from Waxon. It has no
published release or separate security support program yet.

Keep Spotify tokens, player credentials, debug logs, and personal settings
out of the repository. They live in the user's config directory. Do not
include them when reporting a bug. OAuth tokens should not be pasted into
issue descriptions or terminal screenshots.

The existing GPL license and upstream attribution are retained. See
UPSTREAM.md for the source base. No release attestation or packaged-build
claims are made for this development version.
