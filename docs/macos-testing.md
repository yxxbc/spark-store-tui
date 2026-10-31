# Testing the TUI on macOS

**Acceptance is manual and happens on the maintainer's Mac.** The tmux script in
this repository is only an optional smoke aid; it does not replace a human test
in a real terminal.

Package operations require Linux backends (`aptss` / Amber APM). On macOS you
can still test the full catalog, navigation, search, layouts and image preview.

## Prerequisites

```bash
brew install go tmux chafa   # chafa is optional but recommended
```

A terminal with a native image protocol gives the best preview experience:
Kitty, iTerm2, WezTerm or Ghostty. Other terminals fall back to chafa.

## Build and run

```bash
make go-build
./build/sparkstore
```

`sparkstore: package operations unavailable` is expected on macOS; browsing
still works.

## Optional smoke aid

```bash
scripts/test-tui-macos.sh
```

This starts a 120x40 tmux session and checks text-level state changes. It cannot
see graphical images, cannot judge spacing/legibility and is not the acceptance
test. Please complete the manual checklist below in your own terminal.

## Manual checklist

1. Home screen: the logo, detected host and "启动时不会安装软件" are visible.
2. The app opens directly in the browser and loads the catalog automatically;
   no `Enter` is needed.
3. Press `↓` / `↑` (or `j` / `k`): the selection moves; the details follow.
4. Press `]` then `[`: the category changes, the selection resets and the list
   reloads. Arrow keys must keep moving the app selection.
5. Press `/`, type `Code`, press `Enter`: the list filters; press `Esc`.
6. Select an app with screenshots. The right column shows a small chafa icon
   next to the app name and chafa symbol art for the first screenshot.
7. The right pane shows the application icon and the first screenshot as chafa
   art in the standard store layout; there is no screenshot-switching key.
8. Press `O`: the screen is cleared and the original screenshot is printed by
   the terminal. Kitty/iTerm2/WezTerm/Ghostty use their native protocol; other
   terminals fall back to chafa. Press Enter to return. Inside tmux the preview
   uses chafa by default; set `SPARKSTORE_TMUX_GRAPHICS=1` after enabling
   `allow-passthrough` to try native graphics.
9. Press `E`: the description expands and collapses.
10. Resize the terminal to 100x30 and 80x24: the layout must not scroll the
    header off-screen; short terminals show a compact details pane.
11. Press `D` / `I` / `P` / `U` / `X` on an app: on macOS the backend is
    unavailable, so expect a clear status message instead of a crash (for
    example `X` reports that there is no local package).
12. Press `q`: the process exits and restores the terminal.

If an image does not appear, check `echo "$TERM $TERM_PROGRAM"` and whether
`chafa` is installed. The native protocols are only used when the terminal
advertises them.
