package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

var cachedIconPNG []byte

// GetAppIconPNG returns the Gopad++ 64x64 PNG icon bytes.
func GetAppIconPNG() []byte {
	if len(cachedIconPNG) > 0 {
		return cachedIconPNG
	}

	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Background: Sleek rounded dark slate #161b22
	bg := color.RGBA{R: 22, G: 27, B: 34, A: 255}
	borderCol := color.RGBA{R: 48, G: 54, B: 61, A: 255}
	accentCyan := color.RGBA{R: 0, G: 173, B: 216, A: 255}   // Go Cyan
	accentGreen := color.RGBA{R: 78, G: 186, B: 111, A: 255} // Notepad++ Green
	white := color.RGBA{R: 240, G: 246, B: 252, A: 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// Rounded square mask (radius 12)
			dx := x
			if dx >= size/2 {
				dx = size - 1 - x
			}
			dy := y
			if dy >= size/2 {
				dy = size - 1 - y
			}

			if dx < 12 && dy < 12 {
				distSq := (12-dx)*(12-dx) + (12-dy)*(12-dy)
				if distSq > 12*12 {
					img.Set(x, y, color.Transparent)
					continue
				}
			}

			// Border
			if x == 0 || x == size-1 || y == 0 || y == size-1 {
				img.Set(x, y, borderCol)
				continue
			}

			img.Set(x, y, bg)
		}
	}

	// Draw Notepad page outline: (x: 14 to 50, y: 12 to 52)
	for y := 12; y <= 52; y++ {
		for x := 14; x <= 50; x++ {
			// Right top dog-ear fold
			if x > 40 && y < 22 && (x-40)+(y-12) > 10 {
				continue
			}
			img.Set(x, y, color.RGBA{R: 30, G: 36, B: 46, A: 255})
		}
	}

	// Code editor lines (horizontal stripes)
	lineColors := []color.RGBA{accentCyan, accentGreen, white, accentCyan, accentGreen}
	lineY := []int{22, 28, 34, 40, 46}
	lineW := []int{18, 26, 22, 28, 16}

	for i, ly := range lineY {
		c := lineColors[i%len(lineColors)]
		lw := lineW[i%len(lineW)]
		for y := ly; y < ly+2; y++ {
			for x := 20; x < 20+lw; x++ {
				img.Set(x, y, c)
			}
		}
	}

	// "++" badge in bottom right (x: 40 to 52, y: 38 to 50)
	badgeBg := accentGreen
	for y := 38; y <= 50; y++ {
		for x := 38; x <= 50; x++ {
			img.Set(x, y, badgeBg)
		}
	}
	// First '+'
	for y := 41; y <= 47; y++ {
		img.Set(41, y, white)
	}
	for x := 39; x <= 43; x++ {
		img.Set(x, 44, white)
	}
	// Second '+'
	for y := 41; y <= 47; y++ {
		img.Set(47, y, white)
	}
	for x := 45; x <= 49; x++ {
		img.Set(x, 44, white)
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	cachedIconPNG = buf.Bytes()
	return cachedIconPNG
}

// EnsureDesktopEntry installs the .desktop file and icon for GNOME 46.
func EnsureDesktopEntry(binaryPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 1. Write icon PNG
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "64x64", "apps")
	_ = os.MkdirAll(iconDir, 0755)
	iconPath := filepath.Join(iconDir, "gpad.png")
	if err := os.WriteFile(iconPath, GetAppIconPNG(), 0644); err != nil {
		return err
	}

	// 2. Write gpad.desktop
	desktopDir := filepath.Join(home, ".local", "share", "applications")
	_ = os.MkdirAll(desktopDir, 0755)
	desktopPath := filepath.Join(desktopDir, "gpad.desktop")

	desktopContent := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Gopad++\n" +
		"Comment=High-Density Native Text Editor\n" +
		"Exec=" + binaryPath + " %F\n" +
		"Icon=" + iconPath + "\n" +
		"Terminal=false\n" +
		"Categories=Development;TextEditor;Utility;\n" +
		"MimeType=text/plain;text/x-go;text/x-c;text/x-c++;text/markdown;\n" +
		"StartupNotify=true\n"

	return os.WriteFile(desktopPath, []byte(desktopContent), 0644)
}
