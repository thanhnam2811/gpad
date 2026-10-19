package keybind

import (
	"fmt"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// ActionID uniquely identifies an executable editor operation.
type ActionID string

const (
	ActionFileNew         ActionID = "file.new"
	ActionFileOpen        ActionID = "file.open"
	ActionFileSave        ActionID = "file.save"
	ActionFileSaveAs      ActionID = "file.saveAs"
	ActionFileCloseTab    ActionID = "file.closeTab"
	ActionFileCloseOthers ActionID = "file.closeOthers"
	ActionFileCloseRight  ActionID = "file.closeRight"
	ActionFileCloseAll    ActionID = "file.closeAll"
	ActionFileExit        ActionID = "file.exit"

	ActionEditUndo          ActionID = "edit.undo"
	ActionEditRedo          ActionID = "edit.redo"
	ActionEditCut           ActionID = "edit.cut"
	ActionEditCopy          ActionID = "edit.copy"
	ActionEditPaste         ActionID = "edit.paste"
	ActionEditSelectAll     ActionID = "edit.selectAll"
	ActionEditDupLine       ActionID = "edit.duplicateLine"
	ActionEditDelLine       ActionID = "edit.deleteLine"
	ActionEditToggleComment ActionID = "edit.toggleComment"

	ActionSearchFind          ActionID = "search.find"
	ActionSearchReplace       ActionID = "search.replace"
	ActionSearchFindNext      ActionID = "search.findNext"
	ActionSearchFindPrev      ActionID = "search.findPrev"
	ActionSearchGoToLine      ActionID = "search.goToLine"
	ActionSearchSelectMatches ActionID = "search.selectMatches"

	ActionViewToggleWrap     ActionID = "view.toggleWrap"
	ActionViewToggleLineNums ActionID = "view.toggleLineNumbers"
	ActionViewFullScreen     ActionID = "view.fullScreen"

	ActionDocConvertLF   ActionID = "doc.convertLF"
	ActionDocConvertCRLF ActionID = "doc.convertCRLF"
	ActionDocCopyPath    ActionID = "doc.copyPath"

	ActionHelpAbout ActionID = "help.about"
)

// Binding represents a keyboard shortcut combination.
type Binding struct {
	Mods ui.Modifiers
	Key  ui.Key
}

func packBinding(mods ui.Modifiers, key ui.Key) uint32 {
	return (uint32(mods) << 16) | uint32(key)
}

// String renders a high-density, human-readable shortcut string (e.g. "Ctrl+Shift+S", "F3").
func (b Binding) String() string {
	if b.Key == 0 {
		return ""
	}
	var parts []string
	if b.Mods&ui.Cmd != 0 || b.Mods&ui.Ctrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if b.Mods&ui.Alt != 0 {
		parts = append(parts, "Alt")
	}
	if b.Mods&ui.Shift != 0 {
		parts = append(parts, "Shift")
	}

	keyName := formatKey(b.Key)
	if keyName != "" {
		parts = append(parts, keyName)
	}

	return strings.Join(parts, "+")
}

func formatKey(k ui.Key) string {
	switch k {
	case ui.KeyA:
		return "A"
	case ui.KeyB:
		return "B"
	case ui.KeyC:
		return "C"
	case ui.KeyD:
		return "D"
	case ui.KeyE:
		return "E"
	case ui.KeyF:
		return "F"
	case ui.KeyG:
		return "G"
	case ui.KeyH:
		return "H"
	case ui.KeyI:
		return "I"
	case ui.KeyJ:
		return "J"
	case ui.KeyK:
		return "K"
	case ui.KeyL:
		return "L"
	case ui.KeyM:
		return "M"
	case ui.KeyN:
		return "N"
	case ui.KeyO:
		return "O"
	case ui.KeyP:
		return "P"
	case ui.KeyQ:
		return "Q"
	case ui.KeyR:
		return "R"
	case ui.KeyS:
		return "S"
	case ui.KeyT:
		return "T"
	case ui.KeyU:
		return "U"
	case ui.KeyV:
		return "V"
	case ui.KeyW:
		return "W"
	case ui.KeyX:
		return "X"
	case ui.KeyY:
		return "Y"
	case ui.KeyZ:
		return "Z"
	case ui.KeyF3:
		return "F3"
	case ui.KeyF11:
		return "F11"
	case ui.KeyEnter:
		return "Enter"
	case ui.KeyEscape:
		return "Esc"
	case ui.KeySlash:
		return "/"
	case ui.KeyEqual:
		return "+"
	case ui.KeyMinus:
		return "-"
	case ui.KeyTab:
		return "Tab"
	default:
		return ""
	}
}

// ActionDef contains metadata for an action.
type ActionDef struct {
	ID       ActionID
	Title    string
	Category string
	Default  Binding
	Current  Binding
}

// Registry manages the central repository of actions and keyboard shortcuts.
type Registry struct {
	mu      sync.RWMutex
	actions map[ActionID]*ActionDef
	lookup  map[uint32]ActionID
}

// NewRegistry initializes the registry with professional default keybindings.
func NewRegistry() *Registry {
	r := &Registry{
		actions: make(map[ActionID]*ActionDef, 32),
		lookup:  make(map[uint32]ActionID, 32),
	}

	defaults := []ActionDef{
		{ID: ActionFileNew, Title: "New Tab", Category: "File", Default: Binding{Mods: ui.Cmd, Key: ui.KeyN}},
		{ID: ActionFileOpen, Title: "Open File...", Category: "File", Default: Binding{Mods: ui.Cmd, Key: ui.KeyO}},
		{ID: ActionFileSave, Title: "Save", Category: "File", Default: Binding{Mods: ui.Cmd, Key: ui.KeyS}},
		{ID: ActionFileSaveAs, Title: "Save As...", Category: "File", Default: Binding{Mods: ui.Cmd | ui.Shift, Key: ui.KeyS}},
		{ID: ActionFileCloseTab, Title: "Close Tab", Category: "File", Default: Binding{Mods: ui.Cmd, Key: ui.KeyW}},
		{ID: ActionFileCloseOthers, Title: "Close Other Tabs", Category: "File"},
		{ID: ActionFileCloseRight, Title: "Close Tabs to the Right", Category: "File"},
		{ID: ActionFileCloseAll, Title: "Close All Tabs", Category: "File"},
		{ID: ActionFileExit, Title: "Exit", Category: "File", Default: Binding{Mods: ui.Cmd, Key: ui.KeyQ}},

		{ID: ActionEditUndo, Title: "Undo", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyZ}},
		{ID: ActionEditRedo, Title: "Redo", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyY}},
		{ID: ActionEditCut, Title: "Cut", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyX}},
		{ID: ActionEditCopy, Title: "Copy", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyC}},
		{ID: ActionEditPaste, Title: "Paste", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyV}},
		{ID: ActionEditSelectAll, Title: "Select All", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyA}},
		{ID: ActionEditDupLine, Title: "Duplicate Line", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeyD}},
		{ID: ActionEditDelLine, Title: "Delete Line", Category: "Edit", Default: Binding{Mods: ui.Cmd | ui.Shift, Key: ui.KeyK}},
		{ID: ActionEditToggleComment, Title: "Toggle Line Comment", Category: "Edit", Default: Binding{Mods: ui.Cmd, Key: ui.KeySlash}},

		{ID: ActionSearchFind, Title: "Find...", Category: "Search", Default: Binding{Mods: ui.Cmd, Key: ui.KeyF}},
		{ID: ActionSearchReplace, Title: "Replace...", Category: "Search", Default: Binding{Mods: ui.Cmd, Key: ui.KeyH}},
		{ID: ActionSearchFindNext, Title: "Find Next", Category: "Search", Default: Binding{Mods: 0, Key: ui.KeyF3}},
		{ID: ActionSearchFindPrev, Title: "Find Previous", Category: "Search", Default: Binding{Mods: ui.Shift, Key: ui.KeyF3}},
		{ID: ActionSearchGoToLine, Title: "Go to Line...", Category: "Search", Default: Binding{Mods: ui.Cmd, Key: ui.KeyG}},
		{ID: ActionSearchSelectMatches, Title: "Select All Matches", Category: "Search", Default: Binding{Mods: ui.Alt, Key: ui.KeyEnter}},

		{ID: ActionViewToggleWrap, Title: "Toggle Word Wrap", Category: "View", Default: Binding{Mods: ui.Alt, Key: ui.KeyZ}},
		{ID: ActionViewToggleLineNums, Title: "Toggle Line Numbers", Category: "View"},
		{ID: ActionViewFullScreen, Title: "Toggle Full Screen", Category: "View", Default: Binding{Mods: 0, Key: ui.KeyF11}},

		{ID: ActionDocConvertLF, Title: "Convert to LF (Unix)", Category: "Document"},
		{ID: ActionDocConvertCRLF, Title: "Convert to CRLF (Windows)", Category: "Document"},
		{ID: ActionDocCopyPath, Title: "Copy Document Path", Category: "Document"},

		{ID: ActionHelpAbout, Title: "About Gopad++", Category: "Help"},
	}

	for _, d := range defaults {
		d.Current = d.Default
		r.actions[d.ID] = &d
		if d.Current.Key != 0 {
			r.lookup[packBinding(d.Current.Mods, d.Current.Key)] = d.ID
		}
	}

	return r
}

// Lookup finds the ActionID mapped to the pressed key combination in O(1) time with 0 allocations.
func (r *Registry) Lookup(mods ui.Modifiers, key ui.Key) (ActionID, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.lookup[packBinding(mods, key)]
	return id, ok
}

// GetAction returns the definition and current shortcut for an action ID.
func (r *Registry) GetAction(id ActionID) (*ActionDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.actions[id]
	return a, ok
}

// ShortcutString returns the display text for an action's shortcut (e.g. "Ctrl+S").
func (r *Registry) ShortcutString(id ActionID) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if a, ok := r.actions[id]; ok {
		return a.Current.String()
	}
	return ""
}

// SetBinding re-maps an action to a new custom shortcut combination.
func (r *Registry) SetBinding(id ActionID, binding Binding) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	a, ok := r.actions[id]
	if !ok {
		return fmt.Errorf("action %q not found", id)
	}

	// Remove old reverse lookup
	if a.Current.Key != 0 {
		delete(r.lookup, packBinding(a.Current.Mods, a.Current.Key))
	}

	// Set new binding
	a.Current = binding
	if binding.Key != 0 {
		r.lookup[packBinding(binding.Mods, binding.Key)] = id
	}

	return nil
}
