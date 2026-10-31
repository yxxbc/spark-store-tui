# Security

`sparkstore` runs as an unprivileged user. Package operations are delegated to
`aptss` (Debian family) or Amber APM (`apm`) through `sudo`; the TUI itself
never runs as a privileged process and never installs a setuid helper.

## Privilege boundary

- No operation is executed without an explicit key press from the user.
  Destructive keys (`U`, `X`) act immediately by design; the TUI shows the
  result in its status line.
- Commands are built as argument vectors (`exec.Command`) and never passed to a
  shell. Package names and filenames are validated against a character
  whitelist before they can reach a package manager.
- Non-root sessions run `sudo -- <backend> ...`. `--` prevents a backend path
  or package name from being parsed as a `sudo` option.
- The installer only uses `sudo` for package-manager calls; it never pipes
  remote content into a shell.
- WSL sessions strip Windows paths from `PATH` before invoking Amber APM.

## Download and metadata trust

- Catalog data is parsed as JSON with bounded responses; no external process is
  started for browsing or search.
- Metalink documents are limited to 1 MiB. Package filenames must be a single
  path segment; path traversal and control characters are rejected.
- Only `http(s)` resource URLs with a host and no embedded credentials are
  accepted. Plain HTTP is refused unless `SPARKSTORE_ALLOW_HTTP=1` is set.
- Spark currently publishes MD5/SHA-1 in Metalink. The client verifies SHA-256
  when present, otherwise SHA-1 over HTTPS as a transport-integrity check.
  SHA-1 is not treated as proof of authenticity. Debian downloads are verified
  by `aptss` against the repository digest.
- Downloads are written to a `.part` file, verified, then atomically renamed.
- Remote images are limited to 12 MiB, require an `image/*` content type, and
  are cached with `0700` directories / `0600` files.

## Local files

- `DeletePackage` only removes files inside the configured download directory.
- Task state is written with `0600` and replaced atomically.

## Secrets and release artifacts

- CI tokens are read from environment variables or the private
  `~/.config/repo-all-sync/config.json`; they are never printed or committed.
- Release assets include `SHA256SUMS`. Verify it before installing:

  ```bash
  sha256sum -c SHA256SUMS
  ```

- AUR files are generated from the same source archive and digest that the
  release publishes.

## Reporting a vulnerability

Open a private report to the maintainers instead of a public issue. Include the
affected version (`sparkstore --version`), distribution, reproduction steps and
the expected impact. Do not include secrets, tokens or personal data.

## Supported versions

Security fixes are issued for the latest minor release (currently 0.9.x).
