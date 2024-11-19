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

func OpenFileInEditor(editor, fileName string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("%s %s", editor, fileName))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, fileName))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
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
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		lineCount++
		if lineCount >= 200 {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
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
