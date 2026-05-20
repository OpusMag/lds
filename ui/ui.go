package ui

import (
	"fmt"
	"strings"

	"lds/config"
	"lds/fileops"
	"lds/fsinfo"

	"github.com/gdamore/tcell/v2"
)

var Titles = []string{"Directories", "Files", "Search", "File Info"}

// Layout holds the per-frame layout dimensions computed from the screen size.
type Layout struct {
	Width, Height      int
	BoxWidth           int
	HalfBoxHeight      int
	IncreasedBoxHeight int
}

// ComputeLayout returns the Layout for the given screen size. Replaces the
// older global-mutating CalculateBoxDimensions.
func ComputeLayout(width, height int) Layout {
	boxWidth := width / 2
	boxHeight := height / 2
	half := boxHeight / 2
	return Layout{
		Width:              width,
		Height:             height,
		BoxWidth:           boxWidth,
		HalfBoxHeight:      half,
		IncreasedBoxHeight: boxHeight + half,
	}
}

// Styles is the set of resolved tcell styles for the current config. Built
// once per config load via NewStyles.
type Styles struct {
	Text      tcell.Style
	Border    tcell.Style
	Highlight tcell.Style
	Blinking  tcell.Style
	Label     tcell.Style
	Value     tcell.Style
	Focused   tcell.Style
}

// NewStyles resolves a Styles set from the config's colour palette.
func NewStyles(cfg *config.Config) Styles {
	return Styles{
		Text:      tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Text)),
		Border:    tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Border)),
		Highlight: tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Highlight)).Bold(true),
		Blinking:  tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Blinking)).Bold(true),
		Label:     tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Label)),
		Value:     tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Value)).Bold(true),
		Focused:   tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Focused)).Bold(true),
	}
}

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

func DrawBox(screen tcell.Screen, x, y, width, height int, files []fsinfo.FileInfo, selectedIndex int, scrollPosition int, textStyle, highlightStyle tcell.Style, isFocused bool) {
	maxLines := height - 2
	for i := scrollPosition; i < len(files) && i < scrollPosition+maxLines; i++ {
		file := files[i]
		style := textStyle
		if isFocused && i == selectedIndex {
			style = highlightStyle
		}
		lineY := y + (i - scrollPosition) + 1
		for j, r := range file.Name {
			if x+3+j >= x+width {
				break
			}
			screen.SetContent(x+3+j, lineY, r, nil, style)
		}
	}
}

func DrawTitles(screen tcell.Screen, layout Layout, style tcell.Style) {
	for i, title := range Titles {
		var tx, ty int
		switch i {
		case 0:
			tx, ty = 1, 0
		case 1:
			tx, ty = layout.BoxWidth+1, 0
		case 2:
			tx, ty = 1, layout.IncreasedBoxHeight
		case 3:
			tx, ty = layout.BoxWidth+1, layout.IncreasedBoxHeight
		}
		for j, r := range title {
			screen.SetContent(tx+j, ty, r, nil, style)
		}
	}
}

func DrawText(screen tcell.Screen, x, y int, text string) {
	for i, r := range text {
		screen.SetContent(x+i, y, r, nil, tcell.StyleDefault)
	}
}

// DrawFilePreview renders a preview of file into the pane at (x,y,w,h).
// The pane is split: the upper half (h/2) is assumed to contain metadata
// drawn by DisplayFileInfo; the lower remainder is used for the preview.
func DrawFilePreview(screen tcell.Screen, x, y, w, h int, file fsinfo.FileInfo, style tcell.Style) {
	split := h / 2
	previewY := y + split
	previewH := h - split
	if previewH <= 0 {
		return
	}
	contents, err := fileops.ReadFileContents(file.Name)
	if err != nil {
		displayText(screen, x+1, previewY, fmt.Sprintf("Error reading file: %v", err), style, w-2)
		return
	}
	lines := strings.Split(contents, "\n")
	contentX := x + 1
	contentWidth := w - 2
	for i, line := range lines {
		if i >= previewH {
			break
		}
		displayText(screen, contentX, previewY+i, line, style, contentWidth)
	}
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

func DrawASCIIArt(screen tcell.Screen, borderStyle tcell.Style) {
	width, height := screen.Size()
	asciiArt := `
___     _____   _____ 
| |    |  __ \ / ____| 
| |    | |  | || (___  
| |    | |  | |\___  \ 
| |___ | |__| |____) | 
|_____||_____/|_____/ `
	asciiArtLines := strings.Split(asciiArt, "\n")
	maxWidth := 0
	for _, line := range asciiArtLines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}
	asciiArtX := width - maxWidth - 2
	asciiArtY := height - len(asciiArtLines) - 2

	for i, line := range asciiArtLines {
		for j, r := range line {
			if asciiArtX+j < width && asciiArtY+i < height {
				screen.SetContent(asciiArtX+j, asciiArtY+i, r, nil, borderStyle)
			}
		}
	}
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

func DisplayFileInfo(screen tcell.Screen, x, y, maxWidth int, file fsinfo.FileInfo, labelStyle, valueStyle tcell.Style) {
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

func DrawStatusBar(screen tcell.Screen, width, height int, style tcell.Style) {
	msg := "Tab: next box • Shift+Tab: prev box • Alt+Up/Backspace: parent dir • Enter: open/go into • ↑/↓: navigate • Ctrl+U: clear search"
	y := height - 1
	for i, r := range msg {
		if 1+i >= width {
			break
		}
		screen.SetContent(1+i, y, r, nil, style)
	}
}
