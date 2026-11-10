# Optional GitHub Actions checks

`github-actions.yml` contains the build, race-test, and vet workflow. It is
an inactive template here. To activate it, move it to
`.github/workflows/ci.yml` using a GitHub credential with workflow access.

The same checks can be run locally with `make check`.
