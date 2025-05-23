package ui

import (
	"fmt"
	"lds/config"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var (
	Titles             = []string{"Directories", "Files", "Search", "File Info"}
	IncreasedBoxHeight int
	HalfBoxHeight      int
)

func DrawBorder(screen tcell.Screen, x1, y1, x2, y2 int, style tcell.Style) {
	for x := x1; x <= x2; x++ {
		screen.SetContent(x, y1, tcell.RuneHLine, nil, style)
		screen.SetContent(x, y2, tcell.RuneHLine, nil, style)
	}
	for y := y1; y <= y2; y++ {
		screen.SetContent(x1, y, tcell.RuneVLine, nil, style)
		screen.SetContent(x2, y, tcell.RuneVLine, nil, style)
	}
	screen.SetContent(x1, y1, tcell.RuneULCorner, nil, style)
	screen.SetContent(x2, y1, tcell.RuneURCorner, nil, style)
	screen.SetContent(x1, y2, tcell.RuneLLCorner, nil, style)
	screen.SetContent(x2, y2, tcell.RuneLRCorner, nil, style)
}

func CalculateBoxDimensions(width, height int) (int, int, int, int) {
	boxWidth := width / 2
	boxHeight := height / 2
	HalfBoxHeight = boxHeight / 2
	IncreasedBoxHeight = boxHeight + HalfBoxHeight
	return boxWidth, boxHeight, HalfBoxHeight, IncreasedBoxHeight
}

func DrawText(screen tcell.Screen, x, y int, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, tcell.StyleDefault)
	}
}

func DrawTitle(screen tcell.Screen, title string) {
	width, _ := screen.Size()
	titleX := width/2 - len(title)/2
	DrawText(screen, titleX, 0, title)
}

func DrawPrompt(screen tcell.Screen, prompt string) {
	width, height := screen.Size()
	boxWidth := width / 2
	boxHeight := height / 4
	x1 := (width - boxWidth) / 2
	y1 := (height - boxHeight) / 2
	x2 := x1 + boxWidth - 1
	y2 := y1 + boxHeight - 1

	DrawBorder(screen, x1, y1, x2, y2, tcell.StyleDefault.Foreground(tcell.ColorWhite))

	DrawText(screen, x1+2, y1+2, prompt)

	screen.Show()
}

func DisplayFileInfo(screen tcell.Screen, x, y, maxWidth int, file config.FileInfo, labelStyle, valueStyle tcell.Style) {

	labelsColumn1 := []string{
		"Name: ", "Owner: ", "User permissions: ", "Group permissions: ", "Others permissions: ", "File Type: ", "Size: ", "Last Access: ", "Creation Time: ",
	}
	valuesColumn1 := []string{
		file.Name, file.Owner, file.Permissions[1:4], file.Permissions[4:7], file.Permissions[7:10], file.FileType, fmt.Sprintf("%d bytes", file.Size), file.LastAccessTime, file.CreationTime,
	}

	labelsColumn2 := []string{
		"Executable: ", "Git: ", "Mount: ", "Hard Links: ", "Inode: ", "Symlink: ", "Symlink Target: ",
	}
	valuesColumn2 := []string{
		strings.ToUpper(fmt.Sprint(file.IsExecutable)), file.GitRepoStatus, file.MountPoint, fmt.Sprintf("%d", file.HardLinksCount), fmt.Sprintf("%d", file.Inode), fmt.Sprintf("%t", file.IsSymlink), file.SymlinkTarget,
	}

	width, height := screen.Size()
	boxWidth := width / 2
	boxHeight := height / 2

	// Displays labels and values in two columns because there wasn't space in one
	columnWidth := boxWidth / 2

	for i, label := range labelsColumn1 {
		labelX := x
		valueX := x + len(label) + 1
		row := y + i

		// Wraps text so it doesn't spill beyond box height
		if row >= y+boxHeight {
			break
		}

		for j, r := range label {
			if labelX+j < maxWidth {
				screen.SetContent(labelX+j, row, r, nil, labelStyle)
			}
		}

		value := valuesColumn1[i]
		for j, r := range value {
			if valueX+j < maxWidth {
				screen.SetContent(valueX+j, row, r, nil, valueStyle)
			}
		}
	}

	for i, label := range labelsColumn2 {
		labelX := x + columnWidth
		valueX := labelX + len(label) + 1
		row := y + i

		// Wraps text so it doesn't spill beyond box height
		if row >= y+boxHeight {
			break
		}

		for j, r := range label {
			if labelX+j < maxWidth {
				screen.SetContent(labelX+j, row, r, nil, labelStyle)
			}
		}

		value := valuesColumn2[i]
		for j, r := range value {
			if valueX+j < maxWidth {
				screen.SetContent(valueX+j, row, r, nil, valueStyle)
			}
		}
	}
}
