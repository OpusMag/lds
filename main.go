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
	// Find config
	configPaths := []string{
		"/etc/lds/config.json",
		"/usr/local/etc/lds/config.json",
		"/usr/local/lds/config.json",
		"/usr/lds/config.json",
		"/usr/local/bin/config.json",
		"~/.config/lds/config.json",
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

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logging.SetupLogging(cfg.Logging.File)

	// Channel to signal config reload
	reloadConfig := make(chan struct{})
	go events.WatchConfigFile(configPath, reloadConfig)

	// Initialize tcell screen
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

	// Blinking cursor state
	cursorVisible := true
	ticker := time.NewTicker(time.Duration(cfg.AutoSave.Interval) * time.Second)
	defer ticker.Stop()

	// Variables to track the current box and scroll positions
	currentBox := 2 // Start with the search box highlighted
	scrollPositions := []int{0, 0, 0, 0}
	selectedIndices := []int{0, 0, 0, 0}

	// Read initial directory contents and update the best match
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
				// Apply new config settings
			}
		case <-ticker.C:
			cursorVisible = !cursorVisible
		default:
			screen.Clear()

			// Get terminal dimensions
			width, height := screen.Size()
			boxWidth, boxHeight, halfBoxHeight, increasedBoxHeight := ui.CalculateBoxDimensions(width, height)

			// Define styles using config colors
			textStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Text))
			borderStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Border))
			highlightStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Highlight)).Bold(true)
			commandStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Command)).Bold(true)
			blinkingStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Blinking)).Bold(true)
			labelStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Label))
			valueStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Value)).Bold(true)
			focusedStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Focused)).Bold(true)

			// Draw borders for the boxes
			ui.DrawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, borderStyle)                                    // Directories
			ui.DrawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, borderStyle)                                // Files
			ui.DrawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, borderStyle)     // Search
			ui.DrawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, borderStyle) // File Info

			// Display titles for the boxes
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

			// Convert user input to string
			inputStr := string(userInput)

			// Filter file lists based on user input
			filteredDirectories := utils.FilterFiles(directories, inputStr)
			filteredFiles := utils.FilterFiles(append(regularFiles, hiddenFiles...), inputStr)

			// Find the best match
			bestMatch = utils.FindBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

			// Display file names in the directories and files boxes
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

			// Display detailed file information in the file info box
			if (currentBox == 0 || currentBox == 1) && len(boxes[currentBox]) > 0 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, selectedFile, labelStyle, valueStyle)
			} else if currentBox == 2 && bestMatch != nil {
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, *bestMatch, labelStyle, valueStyle)
			}

			// Display file contents in the directory box if a file is highlighted
			// TODO: Clear the content of the directory box before displaying the file contents
			if currentBox == 1 && len(boxes[currentBox]) > 0 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				fileContents, err := fileops.ReadFileContents(selectedFile.Name)
				if err == nil {
					lines := strings.Split(fileContents, "\n")
					for i, line := range lines {
						if i >= increasedBoxHeight-2 { // Adjust to fit within the box
							break
						}
						for j, r := range line {
							if j >= boxWidth-4 { // Adjust to fit within the box
								break
							}
							screen.SetContent(2+j, 1+i, r, nil, textStyle)
						}
					}
				}
			}

			// Define the ASCII art
			asciiArt := `
            ___     _____   _____ 
            | |    |  __ \ / ____| 
            | |    | |  | || (___  
            | |    | |  | |\___  \ 
            | |___ | |__| |____) | 
            |_____||_____/|_____/ `

			// Calculate the starting position for the ASCII art
			asciiArtLines := strings.Split(asciiArt, "\n")
			asciiArtHeight := len(asciiArtLines)
			asciiArtWidth := 0
			for _, line := range asciiArtLines {
				if len(line) > asciiArtWidth {
					asciiArtWidth = len(line)
				}
			}

			// Calculate the position for the ASCII art
			asciiHeight := 8 // One-fourth of the description window height
			asciiBoxYEnd := boxHeight
			asciiBoxYStart := asciiBoxYEnd - asciiHeight
			asciiArtX := boxWidth + boxWidth - asciiArtWidth + 0 // Adjusted to place it on the right side
			asciiArtY := asciiBoxYStart - asciiArtHeight + 25    // Adjusted to move it further down

			// Render the ASCII art in the background
			for y, line := range asciiArtLines {
				for x, r := range line {
					screen.SetContent(asciiArtX+x, asciiArtY+y, r, nil, borderStyle)
				}
			}

			// Display user input in the search box
			for i, r := range userInput {
				screen.SetContent(1+i, increasedBoxHeight+1, r, nil, textStyle)
			}

			// Display blinking cursor in the search box if it is highlighted
			if currentBox == 2 && cursorVisible {
				screen.SetContent(1+len(userInput), increasedBoxHeight+1, '_', nil, blinkingStyle)
			}

			// Highlight the selected box
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

			// Handle user input
			currentBox, userInput, selectedIndices, scrollPositions, bestMatch = events.HandleUserInput(screen, cfg, currentBox, userInput, boxes, selectedIndices, scrollPositions, bestMatch)
			if currentBox == -1 {
				return
			}
		}
	}
}
