package keybind

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestRegistry_Lookup(t *testing.T) {
	reg := NewRegistry()

	// Test Ctrl+N -> ActionFileNew
	action, ok := reg.Lookup(ui.Cmd, ui.KeyN)
	if !ok || action != ActionFileNew {
		t.Fatalf("expected ActionFileNew, got %v (ok=%v)", action, ok)
	}

	// Test Ctrl+Shift+S -> ActionFileSaveAs
	action, ok = reg.Lookup(ui.Cmd|ui.Shift, ui.KeyS)
	if !ok || action != ActionFileSaveAs {
		t.Fatalf("expected ActionFileSaveAs, got %v (ok=%v)", action, ok)
	}

	// Test Zoom and Wrap shortcuts
	action, ok = reg.Lookup(ui.Cmd, ui.KeyEqual)
	if !ok || action != ActionViewZoomIn {
		t.Fatalf("expected ActionViewZoomIn, got %v", action)
	}
	action, ok = reg.Lookup(ui.Cmd, ui.KeyMinus)
	if !ok || action != ActionViewZoomOut {
		t.Fatalf("expected ActionViewZoomOut, got %v", action)
	}
	action, ok = reg.Lookup(ui.Cmd, ui.Key0)
	if !ok || action != ActionViewZoomReset {
		t.Fatalf("expected ActionViewZoomReset, got %v", action)
	}
	action, ok = reg.Lookup(ui.Alt, ui.KeyZ)
	if !ok || action != ActionViewToggleWrap {
		t.Fatalf("expected ActionViewToggleWrap, got %v", action)
	}

	// Test non-existent shortcut
	_, ok = reg.Lookup(ui.Alt, ui.KeyQ)
	if ok {
		t.Fatalf("expected no action for Alt+Q")
	}
}

func TestRegistry_ShortcutString(t *testing.T) {
	reg := NewRegistry()

	str := reg.ShortcutString(ActionFileNew)
	if str != "Ctrl+N" {
		t.Fatalf("expected 'Ctrl+N', got %q", str)
	}

	str = reg.ShortcutString(ActionFileSaveAs)
	if str != "Ctrl+Shift+S" {
		t.Fatalf("expected 'Ctrl+Shift+S', got %q", str)
	}

	if str := reg.ShortcutString(ActionViewZoomIn); str != "Ctrl++" {
		t.Fatalf("expected 'Ctrl++', got %q", str)
	}
	if str := reg.ShortcutString(ActionViewZoomOut); str != "Ctrl+-" {
		t.Fatalf("expected 'Ctrl+-', got %q", str)
	}
	if str := reg.ShortcutString(ActionViewZoomReset); str != "Ctrl+0" {
		t.Fatalf("expected 'Ctrl+0', got %q", str)
	}
	if str := reg.ShortcutString(ActionViewToggleWrap); str != "Alt+Z" {
		t.Fatalf("expected 'Alt+Z', got %q", str)
	}
}

func TestRegistry_SetBinding(t *testing.T) {
	reg := NewRegistry()

	// Rebind ActionFileNew to Ctrl+T
	newBinding := Binding{Mods: ui.Cmd, Key: ui.KeyT}
	err := reg.SetBinding(ActionFileNew, newBinding)
	if err != nil {
		t.Fatalf("SetBinding failed: %v", err)
	}

	// Old binding should be gone
	_, ok := reg.Lookup(ui.Cmd, ui.KeyN)
	if ok {
		t.Fatalf("old binding Ctrl+N should not resolve")
	}

	// New binding should resolve
	action, ok := reg.Lookup(ui.Cmd, ui.KeyT)
	if !ok || action != ActionFileNew {
		t.Fatalf("expected ActionFileNew for Ctrl+T, got %v (ok=%v)", action, ok)
	}
}
