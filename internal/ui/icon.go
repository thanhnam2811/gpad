package ui

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed icon.png
var appIconPNG []byte

//go:embed icon.svg
var appIconSVG []byte

// GetAppIconPNG returns the Gopad++ PNG icon bytes.
func GetAppIconPNG() []byte {
	return appIconPNG
}

// GetAppIconSVG returns the Gopad++ SVG icon bytes.
func GetAppIconSVG() []byte {
	return appIconSVG
}

// EnsureDesktopEntry installs the .desktop file and icon for GNOME 46.
func EnsureDesktopEntry(binaryPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// 1. Write icon PNG and SVG
	iconDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "128x128", "apps")
	_ = os.MkdirAll(iconDir, 0o755)
	iconPath := filepath.Join(iconDir, "gpad.png")
	_ = os.WriteFile(iconPath, GetAppIconPNG(), 0o644)

	svgDir := filepath.Join(home, ".local", "share", "icons", "hicolor", "scalable", "apps")
	_ = os.MkdirAll(svgDir, 0o755)
	_ = os.WriteFile(filepath.Join(svgDir, "gpad.svg"), GetAppIconSVG(), 0o644)

	// 2. Write gpad.desktop
	desktopDir := filepath.Join(home, ".local", "share", "applications")
	_ = os.MkdirAll(desktopDir, 0o755)
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

	return os.WriteFile(desktopPath, []byte(desktopContent), 0o644)
}
