package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestGnomeColorScheme(t *testing.T) {
	for _, tc := range []struct {
		value string
		dark  bool
	}{{"'prefer-dark'\n", true}, {"'prefer-light'", false}, {"'default'", false}, {"", false}} {
		if got := gnomeDarkPreference(tc.value); got != tc.dark {
			t.Errorf("%q: dark=%v", tc.value, got)
		}
	}
}

func TestLightAndDarkThemeApplied(t *testing.T) {
	for _, dark := range []bool{false, true} {
		application := newTestApp(t)
		gui := NewUI(application)
		gui.isDark = dark
		var actual bool
		tester := ui.NewTester(func(c *ui.Context) { gui.Render(c); actual = c.Theme().Dark }, 960, 680)
		tester.Frame()
		if actual != dark {
			t.Fatalf("wanted dark=%v got %v", dark, actual)
		}
	}
}

func TestDesktopEntryXDGScopeAndEscaping(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", data)
	executable := "/opt/My App/$cash%value/gpad"
	if err := EnsureDesktopEntry(executable); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(data, "applications", "gpad.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Exec=\"/opt/My App/\\\\$cash%%value/gpad\" %F\n") {
		t.Fatalf("wrong Exec: %s", content)
	}
	if validator, err := exec.LookPath("desktop-file-validate"); err == nil {
		if output, err := exec.Command(validator, filepath.Join(data, "applications", "gpad.desktop")).CombinedOutput(); err != nil {
			t.Fatalf("desktop validation: %v %s", err, output)
		}
	}
	if !strings.Contains(string(content), "Icon=gpad\n") {
		t.Fatal("icon ID changed")
	}
	if files, _ := os.ReadDir(home); len(files) != 0 {
		t.Fatal("ignored XDG_DATA_HOME")
	}
	for _, path := range []string{"relative", "/bad\npath", "/bad\rpath"} {
		if EnsureDesktopEntry(path) == nil {
			t.Fatalf("accepted %q", path)
		}
	}
}

func TestDesktopEntryErrorsAndRelativeXDGFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "relative-ignored")
	if err := EnsureDesktopEntry("/opt/gpad"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".local/share/applications/gpad.desktop")); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(t.TempDir(), "not-directory")
	if err := os.WriteFile(blocker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", blocker)
	if err := EnsureDesktopEntry("/opt/gpad"); err == nil {
		t.Fatal("swallowed installation error")
	}
}

func TestSaveBeforeCloseFailurePreservesDocument(t *testing.T) {
	application := newTestApp(t)
	doc := application.NewTab("draft", "sensitive draft")
	doc.SetDirty(true)
	gui := NewUI(application)
	gui.confirmClose.Open = true
	gui.confirmClose.Doc = doc
	index := application.ActiveTab
	if err := gui.saveConfirmedTab(index, t.TempDir()); err == nil {
		t.Fatal("save to directory succeeded")
	}
	if application.ActiveDocument() != doc || !doc.IsDirty() || !gui.confirmClose.Open {
		t.Fatal("failed save discarded document or dismissed confirmation")
	}
	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()
	if !tester.HasText("Save failed:") {
		t.Fatalf("save failure not displayed: %v", tester.Texts())
	}
	destination := filepath.Join(t.TempDir(), "saved.txt")
	if err := gui.saveConfirmedTab(index, destination); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != "sensitive draft" {
		t.Fatalf("save: %q %v", got, err)
	}
}

func TestSaveConfirmationRejectsShiftedTab(t *testing.T) {
	application := newTestApp(t)
	original := application.NewTab("original", "original content")
	originalIndex := application.ActiveTab
	other := application.NewTab("other", "other content")
	gui := NewUI(application)
	gui.confirmClose.Doc = original
	gui.confirmClose.Open = true
	// The former slot now names another document, as after an intervening close.
	application.Tabs[originalIndex] = other
	destination := filepath.Join(t.TempDir(), "must-not-exist.txt")
	if err := gui.saveConfirmedTab(originalIndex, destination); err == nil {
		t.Fatal("saved wrong document after tabs shifted")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatalf("unexpected write: %v", err)
	}
	if !gui.confirmClose.Open || application.Tabs[originalIndex] != other {
		t.Fatal("changed tabs after rejecting stale confirmation")
	}
}
