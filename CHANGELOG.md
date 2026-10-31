# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.9.1] - 2026-10-31

### Changed

- The app opens directly in the browser and loads the catalog on startup; the
  Enter-gated home screen was removed.
- Package actions are direct: `D` downloads (or installs an existing local
  package), `I` installs, `P` checks and upgrades, `U` uninstalls and `X` deletes
  the local package. Multi-step confirmation screens were removed.
- A completed download is installed immediately.

### Fixed

- Arrow keys now navigate the app list even when only one catalog source is
  configured; previously they silently reloaded the catalog and reset the view.
- Switching categories with `[` / `]` resets the selected app and keeps focus on
  the app list.
- The details pane now uses the standard store layout: the application icon
  (`icons`) and the first in-app screenshot (`img_urls`) are shown together,
  without a screenshot-switching key.
- In-TUI previews use chafa symbol art, because Bubble Tea v2's text renderer
  strips terminal graphics escape sequences. Press `O` to open the original
  screenshot outside the TUI with Kitty/iTerm2/WezTerm/Ghostty native protocols
  (chafa fallback).
- A two-column browser layout replaces the stacked panes, and short terminals get
  a compact details pane that cannot scroll the whole view.
- Catalog rows are deduplicated by package name and unsafe filenames are
  discarded.
- A catalog older than 10 minutes is refreshed before `D` / `P`, so downloads and
  upgrades do not use stale metadata; image caches revalidate after 24 hours.
- The unsupported-package-backend warning is no longer printed on non-Linux
  development hosts.

### Changed

- Use the Spark Store swirl with a small TUI badge as the current application
  icon; keep the orange terminal mark as the alternative.

## [0.9.0] - 2026-10-31

### Added

- `app`, `catalog`, `packages`, `platform`, `config` and `version` packages.
- `sparkstore --version`, with version, commit and build date injected at build
  time.
- `docs/data-interfaces.md`, JSON schemas, `docs/package-backends.md` and
  `SECURITY.md`.
- GitHub Actions CI, release and mirror workflows, a CNB pipeline and
  `scripts/sync-all.sh`.
- Desktop entry and hicolor icon for GUI application menus.
- Native terminal image protocols with chafa fallback.

### Changed

- The UI now depends only on `domain` and `preview`, and performs process
  handover through `ui.Terminal`.
- Metalink and aptss downloaders share one task lifecycle.
- Debian release assets use `0.9.0-1`; RPM uses `0.9.0-1`; the source archive is
  `r1`.
- The module path and GitHub URLs use `yxxbc/spark-store-tui`.

### Security

- Package commands are built as argument vectors and run through `sudo --`.
- Package names and filenames are validated; local package deletion is limited
  to the download directory.
- Metalink responses are bounded and reject non-HTTP(S) or credential-bearing
  URLs.
- SHA-256 is preferred for verification; SHA-1 is accepted only as an HTTPS
  transport-integrity check for Spark's current Metalink documents.
- Release assets include `SHA256SUMS`; the installer verifies them when present.
- Go 1.26.10 or newer is required for fixed standard-library TLS/x509 code.
