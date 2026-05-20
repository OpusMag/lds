package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lds/fsinfo"
)

func TestFilterFiles(t *testing.T) {
	files := []fsinfo.FileInfo{
		{Name: "alpha.txt"},
		{Name: "Beta.md"},
		{Name: "gamma.go"},
		{Name: "AlphaBet"},
	}
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query returns all", "", []string{"alpha.txt", "Beta.md", "gamma.go", "AlphaBet"}},
		{"case-insensitive substring", "alpha", []string{"alpha.txt", "AlphaBet"}},
		{"upper query", "BETA", []string{"Beta.md"}},
		{"no match", "zzz", nil},
		{"single char", "g", []string{"gamma.go"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterFiles(files, tc.query)
			if len(got) != len(tc.want) {
				t.Fatalf("len=%d want %d (got %+v)", len(got), len(tc.want), got)
			}
			for i, g := range got {
				if g.Name != tc.want[i] {
					t.Errorf("index %d: got %q want %q", i, g.Name, tc.want[i])
				}
			}
		})
	}
}

func TestFindBestMatch(t *testing.T) {
	dirs := []fsinfo.FileInfo{{Name: "docs"}, {Name: "src"}}
	files := []fsinfo.FileInfo{{Name: "readme.md"}, {Name: "main.go"}}
	hidden := []fsinfo.FileInfo{{Name: ".hiddenfile"}}

	t.Run("matches first directory by prefix", func(t *testing.T) {
		got := FindBestMatch(dirs, files, hidden, "doc")
		if got == nil || got.Name != "docs" {
			t.Fatalf("got %+v want docs", got)
		}
	})
	t.Run("falls through to files", func(t *testing.T) {
		got := FindBestMatch(dirs, files, hidden, "main")
		if got == nil || got.Name != "main.go" {
			t.Fatalf("got %+v want main.go", got)
		}
	})
	t.Run("falls through to hidden", func(t *testing.T) {
		got := FindBestMatch(dirs, files, hidden, "hiddenfile")
		if got == nil || got.Name != ".hiddenfile" {
			t.Fatalf("got %+v want .hiddenfile", got)
		}
	})
	t.Run("no match returns nil", func(t *testing.T) {
		got := FindBestMatch(dirs, files, hidden, "nonsense")
		if got != nil {
			t.Fatalf("got %+v want nil", got)
		}
	})
	t.Run("case insensitive", func(t *testing.T) {
		got := FindBestMatch(dirs, files, hidden, "DOCS")
		if got == nil || got.Name != "docs" {
			t.Fatalf("got %+v want docs", got)
		}
	})
}

func TestGetFileType(t *testing.T) {
	dir := t.TempDir()
	regularPath := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(regularPath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	regInfo, err := os.Stat(regularPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := GetFileType(regInfo); got != "Regular File" {
		t.Errorf("regular file: got %q want Regular File", got)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := GetFileType(dirInfo); got != "Directory" {
		t.Errorf("dir: got %q want Directory", got)
	}

	symlinkPath := filepath.Join(dir, "link")
	if err := os.Symlink(regularPath, symlinkPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	linkInfo, err := os.Lstat(symlinkPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := GetFileType(linkInfo); got != "Symlink" {
		t.Errorf("symlink: got %q want Symlink", got)
	}
}

func TestGetLastModified(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		mod  time.Time
		want string
	}{
		{"hours", now.Add(-2 * time.Hour), "hours ago"},
		{"days", now.Add(-3 * 24 * time.Hour), "days ago"},
		{"months", now.Add(-60 * 24 * time.Hour), "months ago"},
		{"years", now.Add(-2 * 365 * 24 * time.Hour), "years ago"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := GetLastModified(tc.mod)
			if !strings.Contains(got, tc.want) {
				t.Errorf("got %q want substring %q", got, tc.want)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	t.Run("no tilde returns as-is", func(t *testing.T) {
		got, err := ExpandPath("/absolute/path")
		if err != nil {
			t.Fatal(err)
		}
		if got != "/absolute/path" {
			t.Errorf("got %q want /absolute/path", got)
		}
	})
	t.Run("empty returns empty", func(t *testing.T) {
		got, err := ExpandPath("")
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("got %q want empty", got)
		}
	})
	t.Run("relative returns as-is", func(t *testing.T) {
		got, err := ExpandPath("foo/bar")
		if err != nil {
			t.Fatal(err)
		}
		if got != "foo/bar" {
			t.Errorf("got %q want foo/bar", got)
		}
	})
	t.Run("tilde expands to non-empty absolute", func(t *testing.T) {
		got, err := ExpandPath("~/something")
		if err != nil {
			t.Fatal(err)
		}
		if got == "~/something" || got == "" {
			t.Errorf("tilde did not expand: %q", got)
		}
		if !strings.HasSuffix(got, "something") {
			t.Errorf("expected suffix 'something', got %q", got)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("expected absolute path, got %q", got)
		}
	})
	t.Run("bare tilde", func(t *testing.T) {
		got, err := ExpandPath("~")
		if err != nil {
			t.Fatal(err)
		}
		if got == "~" || got == "" {
			t.Errorf("bare ~ did not expand: %q", got)
		}
	})
}
