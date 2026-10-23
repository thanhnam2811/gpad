package ui

import (
	_ "embed"
	"fmt"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"strings"
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

// aboutIcon is parsed once, never during rendering.
var aboutIcon = ui.MustParseSVG(appIconSVG)

// EnsureDesktopEntry explicitly installs the per-user launcher and icon assets.
func EnsureDesktopEntry(binaryPath string) error {
	if !filepath.IsAbs(binaryPath) || strings.ContainsAny(binaryPath, "\n\r\x00") {
		return fmt.Errorf("invalid absolute executable path")
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(dataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	for _, asset := range []struct {
		path string
		data []byte
	}{
		{filepath.Join("icons", "hicolor", "128x128", "apps", "gpad.png"), appIconPNG},
		{filepath.Join("icons", "hicolor", "scalable", "apps", "gpad.svg"), appIconSVG},
		{filepath.Join("applications", "gpad.desktop"), []byte("[Desktop Entry]\nType=Application\nName=Gopad++\nGenericName=Text Editor\nComment=Compact native text editor\nExec=" + desktopExec(binaryPath) + " %F\nIcon=gpad\nTerminal=false\nCategories=Utility;TextEditor;\nMimeType=text/plain;\nStartupWMClass=gpad\n")},
	} {
		target := filepath.Join(dataHome, asset.path)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("create launcher directory: %w", err)
		}
		if err := os.WriteFile(target, asset.data, 0644); err != nil {
			return fmt.Errorf("install %s: %w", asset.path, err)
		}
	}
	return nil
}

func desktopExec(path string) string {
	// First quote the Exec argument, then escape the desktop entry string layer.
	quoted := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`, "%", "%%").Replace(path)
	return `"` + strings.ReplaceAll(quoted, `\`, `\\`) + `"`
}
