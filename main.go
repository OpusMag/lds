package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lds/config"
	"lds/events"
	"lds/fileops"
	"lds/logging"
	"lds/ui"
	"lds/utils"

	"github.com/gdamore/tcell/v2"
)

var wg sync.WaitGroup

func main() {

	configPaths := []string{
		"/etc/lds/config.json",
		"/usr/local/etc/lds/config.json",
		"/usr/local/lds/config.json",
		"/usr/lds/config.json",
		"/usr/local/bin/config.json",
		"~/.config/lds/config.json",
		"~/Downloads/lds/config.json",
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		configPaths = append(configPaths, filepath.Join(homeDir, ".lds", "config.json"))
	}
	configPaths = append(configPaths, "config.json")

	var configPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		} else {
			fmt.Printf("Config file not found at: %s\n", path)
		}
	}

	if configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: config file not found in any of the standard locations\n")
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	logging.SetupLogging(cfg.Logging.File)

	reloadConfig := make(chan struct{})
	go events.WatchConfigFile(configPath, reloadConfig)

	screen, err := tcell.NewScreen()
	if err != nil {
		logging.LogErrorAndExit("Error creating screen", err)
	}
	err = screen.Init()
	if err != nil {
		logging.LogErrorAndExit("Error initializing screen", err)
	}
	defer screen.Fini()

	// Buffer for user input in the search box
	var userInput []rune

	cursorVisible := true
	ticker := time.NewTicker(time.Duration(cfg.AutoSave.Interval) * time.Second)
	defer ticker.Stop()

	currentBox := 2 // Start with the search box highlighted so the user can search immediately
	scrollPositions := []int{0, 0, 0, 0}
	selectedIndices := []int{0, 0, 0, 0}

	query := string(userInput)
	directories, regularFiles, hiddenFiles, bestMatch := utils.ReadDirectoryAndUpdateBestMatch(screen, query)

	// Main loop
	for {
		select {
		case <-reloadConfig:
			cfg, err = config.LoadConfig(configPath)
			if err != nil {
				log.Println("Error reloading config:", err)
			} else {
				log.Println("Config loaded")
			}
		case <-ticker.C:
			cursorVisible = !cursorVisible
		default:
			screen.Clear()

			// Uses the terminal dimensions to dynamically calculate the box dimensions so they scale with window size
			width, height := screen.Size()
			boxWidth, boxHeight, halfBoxHeight, increasedBoxHeight := ui.CalculateBoxDimensions(width, height)

			// Don't change these here, change the colors in the config file
			textStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Text))
			borderStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Border))
			highlightStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Highlight)).Bold(true)
			commandStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Command)).Bold(true)
			blinkingStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Blinking)).Bold(true)
			labelStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Label))
			valueStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Value)).Bold(true)
			focusedStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Focused)).Bold(true)

			ui.DrawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, borderStyle)                                    // Directories
			ui.DrawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, borderStyle)                                // Files
			ui.DrawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, borderStyle)     // Search
			ui.DrawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, borderStyle) // File Info

			titles := []string{"Directories", "Files", "Search", "File Info"}
			for i, title := range titles {
				var x, y int
				switch i {
				case 0:
					x, y = 0, 0
				case 1:
					x, y = boxWidth, 0
				case 2:
					x, y = 0, increasedBoxHeight
				case 3:
					x, y = boxWidth, increasedBoxHeight
				}
				for j, r := range title {
					screen.SetContent(x+1+j, y, r, nil, textStyle)
				}
			}

			inputStr := string(userInput)

			filteredDirectories := utils.FilterFiles(directories, inputStr)
			filteredFiles := utils.FilterFiles(append(regularFiles, hiddenFiles...), inputStr)

			bestMatch = utils.FindBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

			boxes := [][]config.FileInfo{filteredDirectories, filteredFiles, nil}
			for i, box := range boxes {
				var x, y, boxHeight int
				switch i {
				case 0:
					x, y, boxHeight = 0, 0, increasedBoxHeight
				case 1:
					x, y, boxHeight = boxWidth, 0, increasedBoxHeight
				case 2:
					x, y, boxHeight = 0, increasedBoxHeight, halfBoxHeight
				}
				if box != nil {
					for j := scrollPositions[i]; j < len(box) && j < scrollPositions[i]+boxHeight-3; j++ {
						file := box[j]
						style := textStyle
						if j == selectedIndices[i] && currentBox == i {
							style = highlightStyle
						}
						if currentBox == 2 && bestMatch != nil && file.Name == bestMatch.Name {
							style = commandStyle
						}
						for k, r := range file.Name {
							if x+3+k < width {
								screen.SetContent(x+3+k, y+j-scrollPositions[i]+1, r, nil, style)
							}
						}
					}
				}
			}

			if (currentBox == 0 || currentBox == 1) && len(boxes[currentBox]) > 0 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, selectedFile, labelStyle, valueStyle)
			} else if currentBox == 2 && bestMatch != nil {
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, *bestMatch, labelStyle, valueStyle)
			}

			// TODO: Clear the content of the directory box before displaying the file contents
			// For now, the file contents are displayed on top of the directories listed in the directory box
			if currentBox == 1 && len(boxes[currentBox]) > 0 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				fileContents, err := fileops.ReadFileContents(selectedFile.Name)
				if err == nil {
					lines := strings.Split(fileContents, "\n")
					for i, line := range lines {
						if i >= increasedBoxHeight-2 {
							break
						}
						for j, r := range line {
							if j >= boxWidth-4 {
								break
							}
							screen.SetContent(2+j, 1+i, r, nil, textStyle)
						}
					}
				}
			}

			asciiArt := `
            ___     _____   _____ 
            | |    |  __ \ / ____| 
            | |    | |  | || (___  
            | |    | |  | |\___  \ 
            | |___ | |__| |____) | 
            |_____||_____/|_____/ `

			asciiArtLines := strings.Split(asciiArt, "\n")
			asciiArtHeight := len(asciiArtLines)
			asciiArtWidth := 0
			for _, line := range asciiArtLines {
				if len(line) > asciiArtWidth {
					asciiArtWidth = len(line)
				}
			}

			asciiHeight := 8
			asciiBoxYEnd := boxHeight
			asciiBoxYStart := asciiBoxYEnd - asciiHeight
			asciiArtX := boxWidth + boxWidth - asciiArtWidth + 0
			asciiArtY := asciiBoxYStart - asciiArtHeight + 25

			// Displays the ASCII art in the background so it doesn't overlap with other content
			for y, line := range asciiArtLines {
				for x, r := range line {
					screen.SetContent(asciiArtX+x, asciiArtY+y, r, nil, borderStyle)
				}
			}

			// Ensures user input is displayed in the search box
			for i, r := range userInput {
				screen.SetContent(1+i, increasedBoxHeight+1, r, nil, textStyle)
			}

			if currentBox == 2 && cursorVisible {
				screen.SetContent(1+len(userInput), increasedBoxHeight+1, '_', nil, blinkingStyle)
			}

			// Highlight the selected box to make it clear which box is currently selected
			switch currentBox {
			case 0:
				ui.DrawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, focusedStyle)
			case 1:
				ui.DrawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, focusedStyle)
			case 2:
				ui.DrawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
			case 3:
				ui.DrawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
			}

			screen.Show()

			currentBox, userInput, selectedIndices, scrollPositions, bestMatch = events.HandleUserInput(screen, cfg, currentBox, userInput, boxes, selectedIndices, scrollPositions, bestMatch)
			if currentBox == -1 {
				return
			}
		}
	}
}
