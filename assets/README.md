# Icon assets

| File | Purpose |
|---|---|
| `icon-official.svg` | Current application icon: the Spark Store swirl with a small TUI badge. Used for the app, package icon and README logo. |
| `icon-official-source.png` | Reference artwork supplied by the maintainer. |
| `icon-official-512.png` | Rendered current icon. |
| `icon.svg` | Alternative orange terminal icon, kept as a secondary mark. |
| `icon-alt-512.png` | Rendered alternative. |

Regenerate the PNG files with:

```bash
scripts/build-icons.sh
```

The package icon is generated from `icon-official.svg` at 256x256.
