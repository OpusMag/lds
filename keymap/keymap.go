package keymap

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"unicode"

	"lds/config"

	"github.com/gdamore/tcell/v2"
)

// Action identifies a high-level user-facing operation bound to a key.
type Action int

const (
	ActionNone Action = iota
	ActionQuit
	ActionNextBox
	ActionPreviousBox
	ActionSelectUp
	ActionSelectDown
	ActionExecute
	ActionBackspace
	ActionRename
	ActionMove
	ActionDelete
	ActionCopy
	ActionGoBack
	ActionParentDir
)

// Binding describes a key combination.
type Binding struct {
	Key  tcell.Key
	Rune rune
	Mods tcell.ModMask
}

// Map associates Actions with their bound key combinations.
type Map map[Action]Binding

var namedKeys = map[string]tcell.Key{
	"tab":       tcell.KeyTab,
	"enter":     tcell.KeyEnter,
	"esc":       tcell.KeyEscape,
	"escape":    tcell.KeyEscape,
	"backspace": tcell.KeyBackspace,
	"up":        tcell.KeyUp,
	"down":      tcell.KeyDown,
	"left":      tcell.KeyLeft,
	"right":     tcell.KeyRight,
	"home":      tcell.KeyHome,
	"end":       tcell.KeyEnd,
	"pgup":      tcell.KeyPgUp,
	"pgdn":      tcell.KeyPgDn,
	"space":     tcell.Key(' '),
	"backtab":   tcell.KeyBacktab,
	"f1":        tcell.KeyF1,
	"f2":        tcell.KeyF2,
	"f3":        tcell.KeyF3,
	"f4":        tcell.KeyF4,
	"f5":        tcell.KeyF5,
	"f6":        tcell.KeyF6,
	"f7":        tcell.KeyF7,
	"f8":        tcell.KeyF8,
	"f9":        tcell.KeyF9,
	"f10":       tcell.KeyF10,
	"f11":       tcell.KeyF11,
	"f12":       tcell.KeyF12,
}

// Parse parses a key spec like "Ctrl+Alt+R" or "Tab" into a Binding.
func Parse(s string) (Binding, error) {
	if strings.TrimSpace(s) == "" {
		return Binding{}, fmt.Errorf("empty key spec")
	}
	tokens := strings.Split(s, "+")
	if len(tokens) == 0 {
		return Binding{}, fmt.Errorf("empty key spec")
	}
	var b Binding
	for i, tok := range tokens {
		t := strings.TrimSpace(tok)
		if t == "" {
			return Binding{}, fmt.Errorf("empty token in key spec %q", s)
		}
		if i < len(tokens)-1 {
			switch strings.ToLower(t) {
			case "ctrl":
				b.Mods |= tcell.ModCtrl
			case "alt":
				b.Mods |= tcell.ModAlt
			case "shift":
				b.Mods |= tcell.ModShift
			case "meta":
				b.Mods |= tcell.ModMeta
			default:
				return Binding{}, fmt.Errorf("unknown modifier %q", t)
			}
			continue
		}
		// final token is the key itself.
		if k, ok := namedKeys[strings.ToLower(t)]; ok {
			b.Key = k
			// Shift+Tab is canonically Backtab on tcell.
			if k == tcell.KeyTab && b.Mods&tcell.ModShift != 0 {
				b.Key = tcell.KeyBacktab
				b.Mods &^= tcell.ModShift
			}
			return b, nil
		}
		// Single character key.
		runes := []rune(t)
		if len(runes) != 1 {
			return Binding{}, fmt.Errorf("unknown key %q", t)
		}
		b.Key = tcell.KeyRune
		b.Rune = unicode.ToLower(runes[0])
		return b, nil
	}
	return Binding{}, fmt.Errorf("missing key in spec %q", s)
}

// Match resolves an EventKey to an Action via this Map.
func (m Map) Match(ev *tcell.EventKey) Action {
	for a, b := range m {
		if matches(b, ev) {
			return a
		}
	}
	return ActionNone
}

func matches(b Binding, ev *tcell.EventKey) bool {
	wanted := b.Mods & (tcell.ModCtrl | tcell.ModAlt | tcell.ModShift | tcell.ModMeta)
	got := ev.Modifiers() & (tcell.ModCtrl | tcell.ModAlt | tcell.ModShift | tcell.ModMeta)

	if b.Key == tcell.KeyRune {
		// Special case: Ctrl+<letter> may surface as KeyCtrlA..KeyCtrlZ.
		if b.Mods&tcell.ModCtrl != 0 && b.Rune >= 'a' && b.Rune <= 'z' {
			ctrlKey := tcell.KeyCtrlA + tcell.Key(b.Rune-'a')
			if ev.Key() == ctrlKey && b.Mods&^tcell.ModCtrl == 0 {
				return true
			}
		}
		if ev.Key() != tcell.KeyRune {
			return false
		}
		if unicode.ToLower(ev.Rune()) != b.Rune {
			return false
		}
		return wanted == got
	}

	// Treat KeyBackspace and KeyBackspace2 (DEL, 0x7f) as the same key:
	// most terminals emit Backspace2 when the user presses Backspace.
	if b.Key == tcell.KeyBackspace || b.Key == tcell.KeyBackspace2 {
		if ev.Key() != tcell.KeyBackspace && ev.Key() != tcell.KeyBackspace2 {
			return false
		}
	} else if ev.Key() != b.Key {
		return false
	}
	return wanted == got
}

type spec struct {
	action  Action
	field   string
	defKey  string
}

func defaults() []spec {
	return []spec{
		{ActionQuit, "Quit", "Ctrl+C"},
		{ActionNextBox, "NextBox", "Tab"},
		{ActionPreviousBox, "PreviousBox", "Shift+Tab"},
		{ActionSelectUp, "SelectUp", "Up"},
		{ActionSelectDown, "SelectDown", "Down"},
		{ActionExecute, "Execute", "Enter"},
		{ActionBackspace, "Backspace", "Backspace"},
		{ActionRename, "Rename", "Ctrl+Alt+R"},
		{ActionMove, "Move", "Ctrl+Alt+M"},
		{ActionDelete, "Delete", "Ctrl+Alt+D"},
		{ActionCopy, "Copy", "Ctrl+Alt+C"},
		{ActionGoBack, "GoBack", "Esc"},
		{ActionParentDir, "ParentDir", "Alt+Up"},
	}
}

// Build constructs a keymap from cfg.KeyBindings, falling back to defaults
// for any binding that fails to parse.
func Build(cfg *config.Config) (Map, error) {
	m := make(Map, len(defaults()))
	kb := reflect.ValueOf(cfg.KeyBindings)
	for _, s := range defaults() {
		var raw string
		if f := kb.FieldByName(s.field); f.IsValid() && f.Kind() == reflect.String {
			raw = f.String()
		}
		var b Binding
		var err error
		if strings.TrimSpace(raw) != "" {
			b, err = Parse(raw)
			if err != nil {
				log.Printf("keymap: invalid binding for %s (%q): %v; using default %q", s.field, raw, err, s.defKey)
				b, err = Parse(s.defKey)
			}
		} else {
			b, err = Parse(s.defKey)
		}
		if err != nil {
			return nil, fmt.Errorf("default binding %q for %s: %w", s.defKey, s.field, err)
		}
		m[s.action] = b
	}
	return m, nil
}
