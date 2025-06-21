package ui

import (
	"fmt"
	"lds/config"

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

func displayText(screen tcell.Screen, startX, y int, text string, style tcell.Style, maxWidth int) {
	screenWidth, screenHeight := screen.Size()

	if y >= screenHeight || startX >= screenWidth {
		return
	}

	for i, r := range text {
		x := startX + i
		if x >= screenWidth || i >= maxWidth {
			break
		}
		screen.SetContent(x, y, r, nil, style)
	}
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	return string(runes[:maxLen])
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}

func DisplayFileInfo(screen tcell.Screen, x, y, maxWidth int, file config.FileInfo, labelStyle, valueStyle tcell.Style) {
	if screen == nil {
		return
	}

	screenWidth, screenHeight := screen.Size()

	if x >= screenWidth || y >= screenHeight || x < 0 || y < 0 {
		return
	}

	if maxWidth > screenWidth-x {
		maxWidth = screenWidth - x
	}

	infoItems := []struct {
		label string
		value string
	}{
		{"Name:", file.Name},
		{"Size:", formatFileSize(file.Size)},
		{"Type:", file.FileType},
		{"Permissions:", file.Permissions},
		{"Owner:", file.Owner},
		{"Last Modified:", file.LastAccessTime},
		{"Git Status:", file.GitRepoStatus},
	}

	currentY := y
	maxDisplayHeight := screenHeight - y - 1

	for i, item := range infoItems {
		if i >= maxDisplayHeight {
			break
		}

		labelWidth := len(item.label) + 1
		valueWidth := maxWidth - labelWidth

		if valueWidth <= 0 {
			continue
		}

		displayText(screen, x, currentY, item.label+" ", labelStyle, maxWidth)

		displayValue := item.value
		if len(displayValue) > valueWidth {
			displayValue = truncateString(displayValue, valueWidth-3) + "..."
		}

		displayText(screen, x+labelWidth, currentY, displayValue, valueStyle, valueWidth)
		currentY++
	}
}
