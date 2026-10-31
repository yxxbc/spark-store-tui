<p align="center">
  <img src="icon.png" width="180" alt="Spark Store TUI logo">
</p>

<h1 align="center">Spark Store TUI</h1>

<p align="center">A native terminal UI for browsing and managing Spark Store packages on Linux.</p>

<p align="center">
  <a href="https://gitee.com/spark-store-project/spark-store-tui"><img src="https://img.shields.io/badge/Gitee-Mainline-C71D23?logo=gitee&logoColor=white" alt="Gitee mainline"></a>
  <a href="https://github.com/yxxbc/spark-store-tui/releases"><img src="https://img.shields.io/badge/GitHub-Releases%20%2F%20Mirror-181717?logo=github&logoColor=white" alt="GitHub releases and mirror"></a>
  <a href="https://cnb.cool/yxxbc/spark-store-tui"><img src="https://img.shields.io/badge/CNB-Mirror-1f6feb" alt="CNB mirror"></a>
  <a href="COPYING"><img src="https://img.shields.io/badge/License-GPL--3.0--only-4caf50" alt="GPL-3.0-only"></a>
  <img src="https://img.shields.io/badge/Go-%3E%3D1.26-00ADD8?logo=go&logoColor=white" alt="Go 1.26 or newer">
</p>

> **Repository roles:** Gitee is the canonical mainline. GitHub builds and
> publishes packages, provides the Go module path and serves as the overseas
> mirror. CNB is a mirror. Release assets are identical on all three.

Current release: **0.9.1**.

## Features

- Browse Spark Store categories, search apps and read long descriptions.
- Debian family: download, verify, install, update and remove through the
  official `aptss` backend.
- Arch, Fedora/RHEL and openSUSE: install through Amber APM.
- Resumable downloads; interrupted tasks are recovered after a restart.
- Check and upgrade only the selected application with `P`.
- In-app screenshots (`img_urls`) with `S`; the metadata icon (`icons`) is used
  as a fallback and as the application icon.
- Original-image preview on Kitty, iTerm2, WezTerm and Ghostty; chafa is the
  fallback.
- A desktop entry and icon are installed so the app appears in GUI menus.
- Fully in-process Bubble Tea UI: no `fzf`, `rg` or shell-based selection.

## Install

### One command (mainland China)

```bash
bash <(curl -fsSL https://gitee.com/spark-store-project/spark-store-tui/raw/main/scripts/install-sparkstore.sh) --mirror gitee
```

The installer detects the distribution and architecture, bootstraps the
official `aptss` package on Debian systems, verifies `SHA256SUMS` when
available and falls back to a source build only after asking. Non-interactive
installs default to Gitee, and source builds use `https://goproxy.cn,direct`
unless `GOPROXY` is set.

### Direct downloads

| Platform | Download |
|---|---|
| Debian / Ubuntu / UOS / Kylin | [Gitee Release](https://gitee.com/spark-store-project/spark-store-tui/releases) · [GitHub Releases](https://github.com/yxxbc/spark-store-tui/releases) |
| Fedora / RHEL / openSUSE | `spark-store-tui-0.9.1-1.<arch>.rpm` |
| Arch Linux | `yay -S spark-store-tui` |
| Source | `spark-store-tui-source-0.9.1-r1.tar.gz` |

Debian example:

```bash
curl -LO https://gitee.com/spark-store-project/spark-store-tui/releases/download/v0.9.1/spark-store-tui_0.9.1-1_amd64.deb
sudo apt install ./spark-store-tui_0.9.1-1_amd64.deb
sudo aptss ssupdate
```

Use `arm64` for aarch64 and `loong64` for LoongArch. On RPM systems:

```bash
sudo dnf install ./spark-store-tui-0.9.1-1.x86_64.rpm
# openSUSE:
sudo zypper install ./spark-store-tui-0.9.1-1.x86_64.rpm
```

### GUI launch

The Debian, RPM and AUR packages install `sparkstore.desktop` and a hicolor
icon. After installation, **Spark Store TUI / 星火终端助手** appears in the
application menu or software center and opens in a terminal (`Terminal=true`).

## Usage

```bash
sparkstore
sparkstore --version
sparkstore --bootstrap-images   # optionally install chafa
```

Package aliases: `SparkStore`, `SPARKSTORE` and `spark-store-tui`.

| Key | Action |
|---|---|
| `↑` / `↓` or `j` / `k` | Move between applications |
| `/` | Search the current category |
| `[` / `]` | Previous / next category |
| `D` | Download or resume |
| `I` / `Enter` | Install the downloaded package |
| `P` | Check and upgrade the selected application |
| `U` | Uninstall |
| `E` | Expand or collapse the description |
| `S` | Cycle in-app screenshots |
| `R` | Reload the catalog |
| `Esc` | Back to the home screen |
| `q` | Quit |

## Package management

The TUI only reads public Spark metadata for browsing. Actual package work is
delegated to the distribution's backend:

- **Debian family:** `aptss` handles repository selection, mirror download,
  digest verification, resume, install, update and removal.
- **Arch / RPM / openSUSE:** Amber APM (`apm`) handles install/remove/update;
  the TUI does not unpack Chrome/Electron packages.

See [Package management and download paths](docs/package-backends.md) for the
exact commands and metadata fields.

## Security

- Commands use argument vectors and `sudo --`; no shell interpolation.
- Package names and filenames are validated; deletion is limited to the download
  directory.
- Metalink responses are bounded and reject unsafe URLs.
- SHA-256 is preferred; SHA-1 is only a fallback for Spark's current Metalink
  documents over HTTPS.
- Release assets ship with `SHA256SUMS`.

Read [SECURITY.md](SECURITY.md) for the full threat model and reporting process.

## Documentation

- [Architecture](docs/architecture.md)
- [Data interfaces and JSON schemas](docs/data-interfaces.md)
- [Package management and download paths](docs/package-backends.md)
- [Packaging and release process](packaging/README.md)
- [Build from source](README.build.md)
- [macOS interaction testing](docs/macos-testing.md)
- [Changelog](CHANGELOG.md)

## Development

```bash
go test ./...
go vet ./...
make check
make go-build          # native binary at build/sparkstore
./build/sparkstore --version
```

Full release artifacts (`deb`, `rpm`, source, AUR and `SHA256SUMS`) are built on
Linux with:

```bash
make dist
```

## Release artifacts

| Channel | Artifact |
|---|---|
| Debian | `spark-store-tui_0.9.1-1_{amd64,arm64,loong64}.deb` |
| RPM | `spark-store-tui-0.9.1-1.{x86_64,aarch64,loongarch64}.rpm` |
| Source | `spark-store-tui-source-0.9.1-r1.tar.gz` |
| Checksums | `SHA256SUMS` |
| AUR | `spark-store-tui` (built locally) |

## License

GPL-3.0-only. See [COPYING](COPYING).
