package main

import (
	"fmt"
	"lds/config"
	"lds/events"
	"lds/logging"
	"lds/ui"
	"lds/utils"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

var wg sync.WaitGroup

func main() {
	// Load configuration
	cfg, err := ui.GetConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		return
	}
	configPath, err := config.FindConfigFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding config: %v\n", err)
		return
	}
	log.Printf("Config file found at: %s", configPath)
	logging.SetupLogging(cfg.Logging.File)

	// Watch for config changes
	reloadConfig := make(chan struct{})
	go events.WatchConfigFile(configPath, reloadConfig)

	// Initialize the terminal screen
	screen, err := tcell.NewScreen()
	if err != nil {
		logging.LogErrorAndExit("Error creating screen", err)
	}
	err = screen.Init()
	if err != nil {
		logging.LogErrorAndExit("Error initializing screen", err)
	}
	defer screen.Fini()

	// State variables
	var userInput []rune
	cursorVisible := true
	ticker := time.NewTicker(time.Duration(cfg.AutoSave.Interval) * time.Second)
	defer ticker.Stop()

	// UI state: currentBox index, scroll and selection for each box
	currentBox := 2 // 2 = search box by default
	scrollPositions := []int{0, 0, 0, 0}
	selectedIndices := []int{0, 0, 0, 0}

	// Initial file lists from the starting directory
	query := string(userInput)
	directories, regularFiles, hiddenFiles, bestMatch := utils.ReadDirectoryAndUpdateBestMatch(screen, query)

	for {
		select {
		case <-reloadConfig:
			cfg, err = config.LoadConfig(configPath)
			if err != nil {
				log.Println("Error reloading config:", err)
			} else {
				log.Println("Config reloaded")
			}
		case <-ticker.C:
			cursorVisible = !cursorVisible
		default:
			screen.Clear()
			width, height := screen.Size()
			boxWidth, _, halfBoxHeight, increasedBoxHeight := ui.CalculateBoxDimensions(width, height)

			// Setup styles from config
			textStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Text))
			borderStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Border))
			highlightStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Highlight)).Bold(true)
			blinkingStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Blinking)).Bold(true)
			labelStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Label))
			valueStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Value)).Bold(true)
			focusedStyle := tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Colors.Focused)).Bold(true)

			// Draw borders for each UI region
			ui.DrawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, borderStyle)                                    // Directories
			ui.DrawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, borderStyle)                                // Files
			ui.DrawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, borderStyle)     // Search
			ui.DrawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, borderStyle) // File Info

			// Draw titles (this function now draws all four titles as in old_ui.go)
			ui.DrawTitles(screen, 0, 0, width, height, "", textStyle)

			// Filter file lists based on user input
			inputStr := string(userInput)
			filteredDirectories := utils.FilterFiles(directories, inputStr)
			filteredFiles := utils.FilterFiles(append(regularFiles, hiddenFiles...), inputStr)
			bestMatch = utils.FindBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

			// Draw file lists in Directories and Files boxes
			ui.DrawBox(screen, 0, 0, boxWidth, increasedBoxHeight, filteredDirectories, selectedIndices[0], scrollPositions[0], textStyle, highlightStyle, currentBox == 0)
			ui.DrawBox(screen, boxWidth, 0, width-boxWidth, increasedBoxHeight, filteredFiles, selectedIndices[1], scrollPositions[1], textStyle, highlightStyle, currentBox == 1)

			// Draw File Contents or File Info depending on selection:
			// When a file is highlighted in the Files box, draw its contents in the Directories box area.
			if currentBox == 1 && len(filteredFiles) > 0 {
				selectedFile := filteredFiles[selectedIndices[1]]
				ui.DrawFileContents(screen, 0, 0, selectedFile, textStyle)
			} else if currentBox == 0 && len(filteredDirectories) > 0 {
				// For a directory selection, keep showing file info as before.
				selectedFile := filteredDirectories[selectedIndices[0]]
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, selectedFile, labelStyle, valueStyle)
			} else if currentBox == 2 && bestMatch != nil {
				ui.DisplayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, *bestMatch, labelStyle, valueStyle)
			}

			ui.DrawASCIIArt(screen)

			for i, r := range userInput {
				screen.SetContent(1+i, increasedBoxHeight+1, r, nil, textStyle)
			}
			if currentBox == 2 && cursorVisible {
				screen.SetContent(1+len(userInput), increasedBoxHeight+1, '_', nil, blinkingStyle)
			}

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
			currentBox, userInput, selectedIndices, scrollPositions, bestMatch =
				events.HandleUserInput(screen, cfg, currentBox, userInput, [][]config.FileInfo{filteredDirectories, filteredFiles, nil}, selectedIndices, scrollPositions, bestMatch)
			if currentBox == -1 {
				return
			}
		}
	}
}
