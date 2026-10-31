# AGENTS.md

Guidance for AI agents and contributors working on Spark Store TUI.

## Repository roles

- **Gitee** (`spark-store-project/spark-store-tui`): canonical mainline and the
  user-facing home for mainland China.
- **GitHub** (`yxxbc/spark-store-tui`): builds and publishes packages, owns the
  Go module path, and serves as the overseas mirror.
- **CNB** (`yxxbc/spark-store-tui`): mirror and CNB-native pipeline.

Do not change these roles, remotes, or release targets without an explicit
maintainer decision.

## Project shape

- Go 1.26.10 or newer, Bubble Tea v2.
- Entry point: `cmd/sparkstore`.
- `internal/app` orchestrates catalog, download and package operations.
- `internal/ui` may import only `internal/domain` and `internal/preview`. It
  must not import `os/exec`, `internal/packages`, `internal/catalog`,
  `internal/platform`, `internal/download` or `internal/state`.
- Process handover belongs in `cmd/sparkstore/terminal.go`.

## Commands

```bash
gofmt -l cmd internal
go vet ./...
go test -race ./...
make check
make go-build                  # build/sparkstore for the current host
./build/sparkstore --version
make release-check             # version/path consistency

scripts/build-icons.sh         # render assets/icon*.svg -> PNGs
scripts/test-tui-macos.sh      # optional tmux smoke aid (not acceptance)

# Linux only (dpkg-deb + rpmbuild):
make dist
scripts/build-dist.sh --skip-deb --skip-rpm
```

## Testing expectations

- Unit tests cover UI state transitions (navigation, categories, screenshots,
  preview placement) and package backend routing.
- Release acceptance is a manual test by the maintainer on a real macOS
  terminal. The tmux script is only a smoke aid; do not treat it as acceptance.
- Before publishing, inspect layouts at 120x40, 100x30 and 80x24.
- Images require a native terminal protocol (Kitty, iTerm2, WezTerm, Ghostty)
  or `chafa`; verify both paths when touching preview code.
- Never publish a release without the maintainer asking for it.

## Security rules

- Build commands as argument vectors; never pass metadata through a shell.
- Non-root operations use `sudo --`.
- Validate package names and filenames before use.
- Local package deletion is restricted to the configured download directory.
- Plain HTTP is opt-in; SHA-256 is preferred over SHA-1.
- Never commit tokens, credentials or secrets.

## Versioning and release

- `VERSION` is the only upstream version. Packaging revisions live in
  `packaging/revisions.env`.
- `CHANGELOG.md` is the release-notes source; `scripts/release-notes.sh` extracts
  the section for the current `VERSION`.
- GitHub Actions builds and publishes artifacts; `repo-all-sync` mirrors source
  and releases to Gitee/CNB.
- Keep artifact names stable: `spark-store-tui_<ver>-<rev>_<arch>.deb`,
  `spark-store-tui-<ver>-<rev>.<arch>.rpm`,
  `spark-store-tui-source-<ver>-r<rev>.tar.gz`.

## Style

- Run `gofmt`.
- Keep user-visible TUI strings in Chinese; keep README and code comments in
  English.
- Define interfaces at the consumer, keep them narrow.
- Prefer standard library and existing dependencies over new ones.

## Documentation

- `README.md`: user-facing English entry point.
- `docs/architecture.md`: layers and boundaries.
- `docs/data-interfaces.md`: metadata and Go contracts.
- `docs/package-backends.md`: aptss/APM commands and metadata research.
- `docs/macos-testing.md`: manual interaction test.
- `SECURITY.md`: threat model and reporting.
