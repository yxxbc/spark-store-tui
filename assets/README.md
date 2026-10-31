# Icon assets

| File | Purpose |
|---|---|
| `icon.svg` | Current application icon: orange terminal prompt and spark. Used for the app, package icon and README logo. |
| `icon-official.svg` | Official-collection variant, derived from the Spark Store swirl artwork (`icon-official-source.png`) with a small TUI badge. Reserved for the official homepage after inclusion. |
| `icon-official-source.png` | Reference artwork supplied by the maintainer. |
| `icon-official-512.png` | Rendered official variant. |

Regenerate the PNG files with:

```bash
scripts/build-icons.sh
```

The package icon is generated from `icon.svg` at 256x256. The current icon is
the only one referenced by packages; the official variant is intentionally kept
outside the packaging flow until the project is listed officially.
