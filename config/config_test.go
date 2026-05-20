package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigLocations_HonorsLDSCONFIG(t *testing.T) {
	t.Setenv("LDS_CONFIG", "/tmp/some/explicit.json")
	t.Setenv("HOME", "/tmp/fake-home")
	locs := ConfigLocations()
	if len(locs) == 0 || locs[0] != "/tmp/some/explicit.json" {
		t.Fatalf("expected LDS_CONFIG first, got %v", locs)
	}
}

func TestConfigLocations_HomeBased(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix paths")
	}
	t.Setenv("LDS_CONFIG", "")
	t.Setenv("HOME", "/tmp/fake-home")
	locs := ConfigLocations()
	wantHas := []string{
		"/tmp/fake-home/.config/lds/config.json",
		"/tmp/fake-home/.lds/config.json",
		"config.json",
	}
	for _, w := range wantHas {
		found := false
		for _, l := range locs {
			if l == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing %q in %v", w, locs)
		}
	}
}

func TestConfigLocations_EmptyLDSCONFIGIsIgnored(t *testing.T) {
	t.Setenv("LDS_CONFIG", "   ")
	t.Setenv("HOME", "/tmp/fake-home")
	locs := ConfigLocations()
	for _, l := range locs {
		if strings.TrimSpace(l) == "" {
			t.Errorf("blank entry in locations: %v", locs)
		}
	}
}

func TestFindConfigFile_LDSCONFIGExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"language":"en"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LDS_CONFIG", path)
	t.Setenv("HOME", t.TempDir())

	got, err := FindConfigFile()
	if err != nil {
		t.Fatalf("FindConfigFile: %v", err)
	}
	if got != path {
		t.Errorf("got %q want %q", got, path)
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	t.Setenv("LDS_CONFIG", "/definitely/does/not/exist.json")
	emptyHome := t.TempDir()
	t.Setenv("HOME", emptyHome)

	// FindConfigFile may still pick "config.json" if cwd contains one.
	// Run in a clean tempdir to be safe.
	origDir, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(emptyHome); err != nil {
		t.Fatal(err)
	}

	_, err := FindConfigFile()
	if err == nil {
		t.Fatal("expected error when no config exists")
	}
	cerr, ok := err.(*ConfigError)
	if !ok {
		t.Fatalf("expected *ConfigError got %T", err)
	}
	if cerr.Message == "" {
		t.Error("ConfigError.Message empty")
	}
	if len(cerr.Paths) == 0 {
		t.Error("ConfigError.Paths empty")
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.json")
	data := `{
		"language": "en",
		"keyBindings": {"quit": "Ctrl+C", "parentDir": "Alt+Up"},
		"font": {"size": 12, "style": "bold"},
		"preferredEditor": "vim"
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("Language got %q want en", cfg.Language)
	}
	if cfg.KeyBindings.Quit != "Ctrl+C" {
		t.Errorf("Quit got %q", cfg.KeyBindings.Quit)
	}
	if cfg.KeyBindings.ParentDir != "Alt+Up" {
		t.Errorf("ParentDir got %q", cfg.KeyBindings.ParentDir)
	}
	if cfg.PreferredEditor != "vim" {
		t.Errorf("PreferredEditor got %q", cfg.PreferredEditor)
	}
	if cfg.Font.Size != 12 {
		t.Errorf("Font.Size got %d", cfg.Font.Size)
	}
}

func TestLoadConfig_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestLoadConfig_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadConfig(filepath.Join(dir, "nope.json"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestExpandPath_NoTilde(t *testing.T) {
	got, err := expandPath("/abs/path")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/abs/path" {
		t.Errorf("got %q", got)
	}
}

func TestExpandPath_Empty(t *testing.T) {
	got, err := expandPath("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q want empty", got)
	}
}

func TestExpandPath_Tilde(t *testing.T) {
	// os.UserHomeDir reads $HOME on unix.
	t.Setenv("HOME", "/tmp/fake-home")
	got, err := expandPath("~/foo")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && got != "/tmp/fake-home/foo" {
		t.Errorf("got %q want /tmp/fake-home/foo", got)
	}
}
