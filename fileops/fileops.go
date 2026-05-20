package fileops

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func resolveEditor(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	
	if vis := os.Getenv("VISUAL"); strings.TrimSpace(vis) != "" {
		return vis, nil
	}
	if ed := os.Getenv("EDITOR"); strings.TrimSpace(ed) != "" {
		return ed, nil
	}
	candidates := []string{"nvim", "vim", "nano", "vi", "micro", "hx"}
	if runtime.GOOS == "windows" {
		candidates = append(candidates, "notepad")
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("no editor found; set $EDITOR or configure PreferredEditor in config")
}

func OpenFileInEditor(editor, fileName string) {
	info, statErr := os.Stat(fileName)
	if statErr != nil {
		fmt.Fprintf(os.Stderr, "Cannot open %q: %v\n", fileName, statErr)
		return
	}
	if info.IsDir() {
		fmt.Fprintf(os.Stderr, "Cannot open directory %q in editor\n", fileName)
		return
	}

	ed, err := resolveEditor(editor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Editor resolution error: %v\n", err)
		return
	}

	var bin string
	var args []string
	if runtime.GOOS == "windows" && fileExists(ed) {
		bin = ed
		args = []string{fileName}
	} else {
		parts := splitCommandLine(ed)
		if len(parts) == 0 {
			fmt.Fprintln(os.Stderr, "Editor resolution produced empty command")
			return
		}
		bin = parts[0]
		args = append(parts[1:], fileName)
	}

	cmd := exec.Command(bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running editor: %v\nTried: %q with args %q\n", err, bin, args)
	}
}

func ReadFileContents(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	lineCount := 0
	truncated := false
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		lineCount++
		if lineCount >= 200 {
			truncated = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if truncated {
		lines = append(lines, "[... file truncated ...]")
	}

	return strings.Join(lines, "\n"), nil
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func MoveFile(src, dst string) error {
	return os.Rename(src, dst)
}

func DeleteFile(fileName string) error {
	return os.Remove(fileName)
}

func RenameFile(oldName, newName string) error {
	return os.Rename(oldName, newName)
}

func splitCommandLine(s string) []string {
	var args []string
	var cur strings.Builder
	var inQuote rune
	escaped := false

	for _, r := range s {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && inQuote != '\'':
			escaped = true
		case r == '"' || r == '\'':
			if inQuote == 0 {
				inQuote = r
			} else if inQuote == r {
				inQuote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == ' ' || r == '\t':
			if inQuote != 0 {
				cur.WriteRune(r)
			} else if cur.Len() > 0 {
				args = append(args, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		args = append(args, cur.String())
	}
	return args
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
