package fileops

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty", "", nil},
		{"single", "vim", []string{"vim"}},
		{"two tokens", "vim -p", []string{"vim", "-p"}},
		{"multiple spaces", "vim    -p   file", []string{"vim", "-p", "file"}},
		{"tabs", "vim\t-p\tfile", []string{"vim", "-p", "file"}},
		{"double quotes", `code "my file.txt"`, []string{"code", "my file.txt"}},
		{"single quotes", `code 'my file.txt'`, []string{"code", "my file.txt"}},
		{"escaped space", `vim my\ file`, []string{"vim", "my file"}},
		{"escaped quote in double", `vim "a\"b"`, []string{"vim", `a"b`}},
		{"mixed", `vim -c "set nu" file.txt`, []string{"vim", "-c", "set nu", "file.txt"}},
		{"single quote no escape", `vim '\n'`, []string{"vim", `\n`}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCommandLine(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("hello copy")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("content: got %q want %q", got, content)
	}
	// src should still exist
	if _, err := os.Stat(src); err != nil {
		t.Errorf("src missing after copy: %v", err)
	}
}

func TestCopyFile_SrcMissing(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nope"), filepath.Join(dir, "out"))
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestMoveFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(src, []byte("move me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MoveFile(src, dst); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected src to be gone, got err=%v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "move me" {
		t.Errorf("got %q want %q", got, "move me")
	}
}

func TestRenameFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "old.txt")
	dst := filepath.Join(dir, "new.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RenameFile(src, dst); err != nil {
		t.Fatalf("RenameFile: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("old name still exists: err=%v", err)
	}
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("new name missing: %v", err)
	}
}

func TestDeleteFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "del.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := DeleteFile(f); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("file still exists: err=%v", err)
	}
}

func TestDeleteFile_Missing(t *testing.T) {
	dir := t.TempDir()
	err := DeleteFile(filepath.Join(dir, "nope"))
	if err == nil {
		t.Fatal("expected error deleting missing file")
	}
}

func TestReadFileContents(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "r.txt")
	if err := os.WriteFile(f, []byte("line1\nline2\nline3"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileContents(f)
	if err != nil {
		t.Fatal(err)
	}
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestReadFileContents_LineCap(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "big.txt")
	var content []byte
	for i := 0; i < 300; i++ {
		content = append(content, []byte("x\n")...)
	}
	if err := os.WriteFile(f, content, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFileContents(f)
	if err != nil {
		t.Fatal(err)
	}
	// Should be capped at 200 lines.
	lines := 1
	for _, r := range got {
		if r == '\n' {
			lines++
		}
	}
	if lines > 200 {
		t.Errorf("expected <=200 lines, got %d", lines)
	}
}

func TestReadFileContents_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadFileContents(filepath.Join(dir, "nope"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
