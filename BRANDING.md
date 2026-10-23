# Gopad++ visual identity

Use **Gopad++** in prose and window titles; **gpad** is the binary, module and desktop-entry
basename. Keep the two plus signs. The repository is `thanhnam2811/gpad`; the AppStream ID is
`io.github.thanhnam2811.gpad`. No affiliation with Notepad++ or the Go project is implied.

## Mark and palette

The existing G/++ rounded-tile identity is retained. Glow, scanlines and hairline decorations
were removed so the mark remains legible at launcher and dialog sizes. A short lower ledge
anchors the tile in the GNOME app grid; the cyan G and mint plus signs distinguish it on both
light and dark surfaces. No font is needed to render the icon.

| Role | Color |
|---|---|
| G / primary accent | `#38c9e8` |
| Plus signs / secondary accent | `#72e0b1` |
| Tile upper surface | `#213140` |
| Tile main surface | `#192633` |
| Tile lower ledge | `#0b121b` |
| Social preview background | `#111820` |
| Light text | `#f3f6f8` |
| Muted text | `#b8c9d6` |

UI controls continue using mygo's light/dark theme colors. The brand palette belongs to the
mark and project artwork; it does not replace semantic editor colors or syntax highlighting.
Keep the compact UI dimensions: 22 px menu, 24 px toolbar, 28 px tabs and 20 px status bar.
Avoid adding decorative padding around the editor to match a marketing image.

## Assets and sources

| Asset | Purpose |
|---|---|
| [assets/icon.svg](assets/icon.svg) | Canonical editable full-color mark, 128-unit viewBox |
| [assets/icon-symbolic.svg](assets/icon-symbolic.svg) | Monochrome desktop variant; themes may recolor foreground |
| `assets/icons/{16,24,32,48,64,128,256,512}.png` | Native raster sizes, transparent background |
| [assets/icon.png](assets/icon.png) | 512 px compatibility/project image |
| `internal/ui/icon.svg`, `internal/ui/icon.png` | Generated embedded copies; PNG is 128 px |
| [Social SVG](assets/branding/social-preview.svg) / [PNG](assets/branding/social-preview.png) | Editable 1280×640 GitHub social preview |
| [Icon review sheet](assets/branding/icon-preview.png) | Actual 16–128 px renders on light/dark backgrounds, not a UI screenshot |
| `assets/screenshots/` | Real native-window captures using synthetic documents |

For HiDPI, select the raster matching physical pixels (for example 64 px for a 32 px slot
at 2×), or use the scalable SVG. Do not upscale a 16 px PNG for a large launcher. Keep the
transparent clearspace built into the SVG. Do not add a glow, change the G to text, recolor
the full-color mark per theme, or stretch the square into a rectangle.

Rebuild committed raster and embedded assets on Ubuntu with librsvg/Cairo:

```sh
sudo apt-get install python3-gi python3-cairo python3-gi-cairo gir1.2-rsvg-2.0
python3 scripts/render-branding.py
python3 scripts/check-assets.py
```

The social preview uses DejaVu Sans via the system font renderer; its text remains editable
in SVG. Generation versions affect raster antialiasing, so use the committed PNGs for releases.
Review small icons at native size and 2×, and check both backgrounds before updating them.
The runtime window, About dialog, launcher and README must use the canonical mark.

## Screenshots and social preview

Screenshots must come from a real app window or be explicitly labeled headless renderings.
Do not composite fictional controls into them. Use synthetic files and the isolated GUI helper
in README; never capture a real session, clipboard, personal paths or active desktop.
`RELEASE_READINESS.md` records the capture environment and test limits. Older images and the
older demo in `assets/` are historical material and are not evidence for the current build.

The social preview is deliberately artwork, not a screenshot or performance assertion.
Its PNG is ready for a maintainer to upload in GitHub repository settings; this task does
not change remote settings or publish it.

## Attribution and license

The Gopad++ icon is a refinement of this project's existing G/++ artwork. Original project
artwork, generated images and screenshot captures are covered by the repository's
[MIT license](LICENSE). No stock image, generated bitmap or external logo was introduced.
The symbolic mark and social preview are original project derivatives under the same license.

The existing toolbar line icons include Feather and adapted Lucide geometry. The File Plus,
Save and Scissors shapes match official Feather SVG paths; Folder Open and Copy follow Lucide
variants. Their MIT/ISC notices and verified upstream snapshots are recorded in
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and bundled with releases. The baseline does
not identify the original import versions, so the notices retain both upstream attributions
without inventing provenance. The G/++ project mark has no external icon dependency.
