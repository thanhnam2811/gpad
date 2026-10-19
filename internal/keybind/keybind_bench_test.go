package keybind

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func BenchmarkRegistry_Lookup(b *testing.B) {
	reg := NewRegistry()

	mods := ui.Cmd | ui.Shift
	key := ui.KeyS

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		action, ok := reg.Lookup(mods, key)
		if !ok || action != ActionFileSaveAs {
			b.Fatalf("Lookup failed")
		}
	}
}

func BenchmarkRegistry_ShortcutString(b *testing.B) {
	reg := NewRegistry()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		str := reg.ShortcutString(ActionFileSaveAs)
		if str == "" {
			b.Fatalf("ShortcutString empty")
		}
	}
}
