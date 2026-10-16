package ui

import "github.com/egoist/mygo/ui"

const (
	svgWrapStart = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`
	svgWrapEnd   = `</svg>`

	pathNew      = `<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="15" y2="15"/>`
	pathOpen     = `<path d="m6 14 1.45-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.55 6a2 2 0 0 1-1.94 1.5H4a2 2 0 0 1-2-2V5c0-1.1.9-2 2-2h3.93a2 2 0 0 1 1.66.9l.82 1.2a2 2 0 0 0 1.66.9H18a2 2 0 0 1 2 2v2"/>`
	pathSave     = `<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/>`
	pathSaveAll  = `<path d="M16 18v3H4a2 2 0 0 1-2-2V7h3"/><path d="M20 15V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v11a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2z"/>`
	pathClose    = `<line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>`
	pathCut      = `<circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><line x1="20" y1="4" x2="8.12" y2="15.88"/><line x1="14.47" y1="14.48" x2="20" y2="20"/><line x1="8.12" y1="8.12" x2="12" y2="12"/>`
	pathCopy     = `<rect width="14" height="14" x="8" y="8" rx="2" ry="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>`
	pathPaste    = `<rect width="8" height="4" x="8" y="2" rx="1" ry="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>`
	pathUndo     = `<path d="M3 7v6h6"/><path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13"/>`
	pathRedo     = `<path d="M21 7v6h-6"/><path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3L21 13"/>`
	pathFind     = `<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>`
	pathReplace  = `<path d="m14 4 4 4-4 4"/><path d="M4 20v-4a4 4 0 0 1 4-4h10"/><path d="m10 20-4-4 4-4"/><path d="M20 4v4a4 4 0 0 1-4 4H6"/>`
	pathGoToLine = `<polyline points="10 9 15 4 20 9"/><path d="M4 20h7a4 4 0 0 0 4-4V4"/>`
	pathWordWrap = `<polyline points="16 16 19 19 16 22"/><path d="M3 6h18"/><path d="M3 12h15a3 3 0 1 1 0 6h-4"/><path d="M3 18h7"/>`
	pathZoomIn   = `<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="11" y1="8" x2="11" y2="14"/><line x1="8" y1="11" x2="14" y2="11"/>`
	pathZoomOut  = `<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="8" y1="11" x2="14" y2="11"/>`
)

// Pre-parsed static GPU vector icons - 0 allocations per frame.
var (
	IconNew      = ui.MustParseSVG([]byte(svgWrapStart + pathNew + svgWrapEnd))
	IconOpen     = ui.MustParseSVG([]byte(svgWrapStart + pathOpen + svgWrapEnd))
	IconSave     = ui.MustParseSVG([]byte(svgWrapStart + pathSave + svgWrapEnd))
	IconSaveAll  = ui.MustParseSVG([]byte(svgWrapStart + pathSaveAll + svgWrapEnd))
	IconClose    = ui.MustParseSVG([]byte(svgWrapStart + pathClose + svgWrapEnd))
	IconCut      = ui.MustParseSVG([]byte(svgWrapStart + pathCut + svgWrapEnd))
	IconCopy     = ui.MustParseSVG([]byte(svgWrapStart + pathCopy + svgWrapEnd))
	IconPaste    = ui.MustParseSVG([]byte(svgWrapStart + pathPaste + svgWrapEnd))
	IconUndo     = ui.MustParseSVG([]byte(svgWrapStart + pathUndo + svgWrapEnd))
	IconRedo     = ui.MustParseSVG([]byte(svgWrapStart + pathRedo + svgWrapEnd))
	IconFind     = ui.MustParseSVG([]byte(svgWrapStart + pathFind + svgWrapEnd))
	IconReplace  = ui.MustParseSVG([]byte(svgWrapStart + pathReplace + svgWrapEnd))
	IconGoToLine = ui.MustParseSVG([]byte(svgWrapStart + pathGoToLine + svgWrapEnd))
	IconWordWrap = ui.MustParseSVG([]byte(svgWrapStart + pathWordWrap + svgWrapEnd))
	IconZoomIn   = ui.MustParseSVG([]byte(svgWrapStart + pathZoomIn + svgWrapEnd))
	IconZoomOut  = ui.MustParseSVG([]byte(svgWrapStart + pathZoomOut + svgWrapEnd))
)
