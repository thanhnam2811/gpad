package ui

import (
	"github.com/egoist/mygo/ui"
)

// renderMenuItem renders a single high-density menu item with an optional shortcut.
func renderMenuItem(c *ui.Context, t *ui.Theme, key, label, shortcut string, disabled bool, onClick func()) {
	btn := ui.ButtonBase(c.Key("mi-"+key)).FillWidth().Height(22).Padding(2, 8).Radius(3).FocusRing(false)
	if disabled {
		btn.Disabled(true)
	} else if btn.Hovered() {
		btn.Background(t.SurfaceHover)
	}

	btn.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			textColor := t.Text
			if disabled {
				textColor = t.TextMuted
			}
			ui.Text(c, label).FontSize(11).TextColor(textColor)

			if shortcut != "" {
				ui.Text(c, shortcut).FontSize(10).TextColor(t.TextMuted)
			}
		})
	})

	if !disabled && btn.Clicked() {
		onClick()
	}
}

// renderMenuSeparator renders a 1px border separator in dropdown menus.
func renderMenuSeparator(c *ui.Context, t *ui.Theme) {
	ui.Box(c).FillWidth().Height(1).Background(t.Border).Margin(2, 0)
}
