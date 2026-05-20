package keymap

import (
	"testing"

	"lds/config"

	"github.com/gdamore/tcell/v2"
)

func TestParse_Valid(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantKey  tcell.Key
		wantRune rune
		wantMods tcell.ModMask
	}{
		{"tab", "Tab", tcell.KeyTab, 0, 0},
		{"enter", "Enter", tcell.KeyEnter, 0, 0},
		{"esc", "Esc", tcell.KeyEscape, 0, 0},
		{"escape long", "Escape", tcell.KeyEscape, 0, 0},
		{"up", "Up", tcell.KeyUp, 0, 0},
		{"backspace", "Backspace", tcell.KeyBackspace, 0, 0},
		{"f1", "F1", tcell.KeyF1, 0, 0},
		{"ctrl+c", "Ctrl+C", tcell.KeyRune, 'c', tcell.ModCtrl},
		{"single rune lowercased", "R", tcell.KeyRune, 'r', 0},
		{"ctrl+alt+r", "Ctrl+Alt+R", tcell.KeyRune, 'r', tcell.ModCtrl | tcell.ModAlt},
		{"shift+tab → backtab", "Shift+Tab", tcell.KeyBacktab, 0, 0},
		{"backtab named", "Backtab", tcell.KeyBacktab, 0, 0},
		{"alt+up", "Alt+Up", tcell.KeyUp, 0, tcell.ModAlt},
		{"case-insensitive mods", "cTrL+aLt+R", tcell.KeyRune, 'r', tcell.ModCtrl | tcell.ModAlt},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := Parse(tc.in)
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.in, err)
			}
			if b.Key != tc.wantKey {
				t.Errorf("Key: got %v want %v", b.Key, tc.wantKey)
			}
			if b.Rune != tc.wantRune {
				t.Errorf("Rune: got %q want %q", b.Rune, tc.wantRune)
			}
			if b.Mods != tc.wantMods {
				t.Errorf("Mods: got %v want %v", b.Mods, tc.wantMods)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"Ctrl+",
		"+R",
		"Foo+R",
		"Ctrl+Ctrl+abc", // last token "abc" — multi-char unknown key
	}
	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			_, err := Parse(in)
			if err == nil {
				t.Errorf("Parse(%q) expected error", in)
			}
		})
	}
}

func TestMatch_CtrlLetterDualForm(t *testing.T) {
	b, err := Parse("Ctrl+R")
	if err != nil {
		t.Fatal(err)
	}
	m := Map{ActionRename: b}

	// Form (a): tcell.KeyCtrlR
	ev1 := tcell.NewEventKey(tcell.KeyCtrlR, 0, tcell.ModCtrl)
	if got := m.Match(ev1); got != ActionRename {
		t.Errorf("KeyCtrlR: got %v want ActionRename", got)
	}

	// Form (b): KeyRune with rune='r' + ModCtrl
	ev2 := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModCtrl)
	if got := m.Match(ev2); got != ActionRename {
		t.Errorf("KeyRune+ModCtrl: got %v want ActionRename", got)
	}

	// No mods → no match
	ev3 := tcell.NewEventKey(tcell.KeyRune, 'r', 0)
	if got := m.Match(ev3); got != ActionNone {
		t.Errorf("plain r: got %v want ActionNone", got)
	}
}

func TestMatch_CtrlAltLetter(t *testing.T) {
	b, _ := Parse("Ctrl+Alt+R")
	m := Map{ActionRename: b}
	// Terminals emit KeyRune with both mods.
	ev := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModCtrl|tcell.ModAlt)
	if got := m.Match(ev); got != ActionRename {
		t.Errorf("Ctrl+Alt+R: got %v want ActionRename", got)
	}
	// Only Ctrl → should not match
	ev2 := tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModCtrl)
	if got := m.Match(ev2); got == ActionRename {
		t.Errorf("Ctrl-only should not match Ctrl+Alt+R")
	}
}

func TestMatch_BackspaceEquivalence(t *testing.T) {
	b, _ := Parse("Backspace")
	m := Map{ActionBackspace: b}

	ev1 := tcell.NewEventKey(tcell.KeyBackspace, 0, 0)
	if got := m.Match(ev1); got != ActionBackspace {
		t.Errorf("KeyBackspace: got %v want ActionBackspace", got)
	}
	ev2 := tcell.NewEventKey(tcell.KeyBackspace2, 0, 0)
	if got := m.Match(ev2); got != ActionBackspace {
		t.Errorf("KeyBackspace2: got %v want ActionBackspace", got)
	}
}

func TestMatch_NamedKey(t *testing.T) {
	b, _ := Parse("Tab")
	m := Map{ActionNextBox: b}
	ev := tcell.NewEventKey(tcell.KeyTab, 0, 0)
	if got := m.Match(ev); got != ActionNextBox {
		t.Errorf("Tab: got %v want ActionNextBox", got)
	}
	// Shift+Tab should NOT match plain Tab binding (different modifier set).
	ev2 := tcell.NewEventKey(tcell.KeyBacktab, 0, 0)
	if got := m.Match(ev2); got == ActionNextBox {
		t.Errorf("Backtab matched plain Tab binding")
	}
}

func TestMatch_ShiftTabBacktab(t *testing.T) {
	b, _ := Parse("Shift+Tab")
	m := Map{ActionPreviousBox: b}
	ev := tcell.NewEventKey(tcell.KeyBacktab, 0, 0)
	if got := m.Match(ev); got != ActionPreviousBox {
		t.Errorf("Backtab: got %v want ActionPreviousBox", got)
	}
}

func TestMatch_AltUp(t *testing.T) {
	b, _ := Parse("Alt+Up")
	m := Map{ActionParentDir: b}
	ev := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModAlt)
	if got := m.Match(ev); got != ActionParentDir {
		t.Errorf("Alt+Up: got %v want ActionParentDir", got)
	}
	// Plain Up should not match
	ev2 := tcell.NewEventKey(tcell.KeyUp, 0, 0)
	if got := m.Match(ev2); got == ActionParentDir {
		t.Errorf("plain Up matched Alt+Up binding")
	}
}

func TestMatch_NoMatch(t *testing.T) {
	b, _ := Parse("Ctrl+C")
	m := Map{ActionQuit: b}
	ev := tcell.NewEventKey(tcell.KeyRune, 'x', 0)
	if got := m.Match(ev); got != ActionNone {
		t.Errorf("unrelated key: got %v want ActionNone", got)
	}
}

func TestBuild_Defaults(t *testing.T) {
	cfg := &config.Config{}
	m, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Should have all the canonical actions populated from defaults.
	wantActions := []Action{
		ActionQuit, ActionNextBox, ActionPreviousBox, ActionSelectUp,
		ActionSelectDown, ActionExecute, ActionBackspace, ActionRename,
		ActionMove, ActionDelete, ActionCopy, ActionGoBack, ActionParentDir,
	}
	for _, a := range wantActions {
		if _, ok := m[a]; !ok {
			t.Errorf("action %v missing", a)
		}
	}
	// Quit default = Ctrl+C
	ev := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
	if got := m.Match(ev); got != ActionQuit {
		t.Errorf("default Ctrl+C: got %v want ActionQuit", got)
	}
}

func TestBuild_PartialOverride(t *testing.T) {
	cfg := &config.Config{}
	cfg.KeyBindings.Quit = "Ctrl+Q"
	m, err := Build(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ev := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModCtrl)
	if got := m.Match(ev); got != ActionQuit {
		t.Errorf("custom Ctrl+Q: got %v want ActionQuit", got)
	}
}

func TestBuild_InvalidFallsBackToDefault(t *testing.T) {
	cfg := &config.Config{}
	cfg.KeyBindings.Quit = "Bogus+Garbage"
	m, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build should not error on invalid binding: %v", err)
	}
	// Should fall back to default Ctrl+C.
	ev := tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModCtrl)
	if got := m.Match(ev); got != ActionQuit {
		t.Errorf("fallback default: got %v want ActionQuit", got)
	}
}
