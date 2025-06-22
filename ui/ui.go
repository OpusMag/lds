package ui

import (
	"fmt"
	"lds/config"
	"lds/fileops"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var (
	Titles             = []string{"Directories", "Files", "Search", "File Info"}
	IncreasedBoxHeight int
	HalfBoxHeight      int
)

func GetConfig() (*config.Config, error) {
	configPath, err := config.FindConfigFile()
	if err != nil {
		if configErr, ok := err.(*config.ConfigError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", configErr.Message)
			fmt.Fprintf(os.Stderr, "Searched in the following locations:\n")
			for _, path := range configErr.Paths {
				fmt.Fprintf(os.Stderr, "  - %s\n", path)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error finding config: %v\n", err)
		}
		return nil, err
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return nil, err
	}
	return cfg, nil
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

func CalculateBoxDimensions(width, height int) (int, int, int, int) {
	boxWidth := width / 2
	boxHeight := height / 2
	HalfBoxHeight = boxHeight / 2
	IncreasedBoxHeight = boxHeight + HalfBoxHeight
	return boxWidth, boxHeight, HalfBoxHeight, IncreasedBoxHeight
}

func DrawBox(screen tcell.Screen, x, y, width, height int, files []config.FileInfo, selectedIndex int, scrollPosition int, textStyle, highlightStyle tcell.Style, isFocused bool) {
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

func DrawTitles(screen tcell.Screen, x, y, width, height int, unused string, style tcell.Style) {
	boxWidth, _, _, increasedBoxHeight := CalculateBoxDimensions(width, height)
	titles := []string{"Directories", "Files", "Search", "File Info"}

	for i, title := range titles {
		var tx, ty int
		switch i {
		case 0:
			tx, ty = 1, 0
		case 1:
			tx, ty = boxWidth+1, 0
		case 2:
			tx, ty = 1, increasedBoxHeight
		case 3:
			tx, ty = boxWidth+1, increasedBoxHeight
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

func DrawFileContents(screen tcell.Screen, x, y int, file config.FileInfo, style tcell.Style) {
	fileContents, err := fileops.ReadFileContents(file.Name)
	if err != nil {
		displayText(screen, x, y, fmt.Sprintf("Error reading file: %v", err), style, 80)
		return
	}
	width, height := screen.Size()
	lines := strings.Split(fileContents, "\n")
	maxLines := height - y - 1
	for i, line := range lines {
		if i >= maxLines {
			break
		}
		displayText(screen, x, y+i, line, style, width-x-1)
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

func DrawASCIIArt(screen tcell.Screen) {
	cfg, err := GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}
	width, height := screen.Size()
	borderStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Border))
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
