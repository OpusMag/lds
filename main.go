package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

type FileInfo struct {
	Name         string
	Permissions  string
	Owner        string
	IsExecutable bool
}

func main() {
	// Initialize tcell screen
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating screen: %v\n", err)
		os.Exit(1)
	}
	err = screen.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing screen: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	// Read current directory contents
	files, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	var directories, regularFiles, hiddenFiles []FileInfo

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Get file permissions and owner
		var permissions, owner string
		var isExecutable bool

		if runtime.GOOS != "windows" {
			stat := info.Sys().(*syscall.Stat_t)
			uid := stat.Uid
			gid := stat.Gid
			usr, _ := user.LookupId(fmt.Sprint(uid))
			grp, _ := user.LookupGroupId(fmt.Sprint(gid))
			permissions = info.Mode().String()
			owner = fmt.Sprintf("%s:%s", usr.Username, grp.Name)
			isExecutable = info.Mode()&0111 != 0
		} else {
			permissions = "N/A"
			owner = "N/A"
			isExecutable = false
		}

		fileInfo := FileInfo{
			Name:         info.Name(),
			Permissions:  permissions,
			Owner:        owner,
			IsExecutable: isExecutable,
		}

		if info.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			if strings.HasPrefix(info.Name(), ".") {
				hiddenFiles = append(hiddenFiles, fileInfo)
			} else {
				regularFiles = append(regularFiles, fileInfo)
			}
		}
	}

	// Variables to track the current box and scroll positions
	currentBox := 2 // Start with the search box highlighted
	scrollPositions := []int{0, 0, 0, 0}
	selectedIndices := []int{0, 0, 0, 0}

	// Buffer for user input in the search box
	var userInput []rune

	// Blinking cursor state
	cursorVisible := true
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Add a variable to track the best search result match
	var bestMatch *FileInfo

	// Modify the main loop to update the best match based on user input
	for {
		select {
		case <-ticker.C:
			cursorVisible = !cursorVisible
		default:
			screen.Clear()

			// Get terminal dimensions
			width, height := screen.Size()

			// Define styles
			whiteStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			tealStyle := tcell.StyleDefault.Foreground(tcell.ColorTeal)
			highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorLightSkyBlue).Bold(true)
			commandStyle := tcell.StyleDefault.Foreground(tcell.ColorPurple)
			blinkingStyle := tcell.StyleDefault.Foreground(tcell.ColorLimeGreen).Bold(true)
			permissionsTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			permissionsValueStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
			ownerTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorFuchsia)
			ownerValueStyle := tcell.StyleDefault.Foreground(tcell.ColorTeal)
			executableTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
			executableValueStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue)

			// Calculate box dimensions
			boxWidth := width / 2
			boxHeight := height / 2
			halfBoxHeight := boxHeight / 2
			increasedBoxHeight := boxHeight + halfBoxHeight

			// Draw borders for the boxes
			drawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, tealStyle)                                    // Directories
			drawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, tealStyle)                                // Files
			drawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, tealStyle)     // Search
			drawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, tealStyle) // Dot Files

			// Display titles for the boxes
			titles := []string{"Directories", "Files", "Search", "Dot Files"}
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
					screen.SetContent(x+1+j, y, r, nil, whiteStyle)
				}
			}

			// Convert user input to string
			inputStr := string(userInput)

			// Filter file lists based on user input
			filteredDirectories := filterFiles(directories, inputStr)
			filteredFiles := filterFiles(regularFiles, inputStr)
			filteredHiddenFiles := filterFiles(hiddenFiles, inputStr)

			// Find the best match
			bestMatch = findBestMatch(filteredDirectories, filteredFiles, filteredHiddenFiles, inputStr)

			// Display file information in the boxes
			boxes := [][]FileInfo{filteredDirectories, filteredFiles, nil, filteredHiddenFiles}
			for i, box := range boxes {
				var x, y, maxHeight int
				switch i {
				case 0:
					x, y, maxHeight = 0, 0, increasedBoxHeight
				case 1:
					x, y, maxHeight = boxWidth, 0, increasedBoxHeight
				case 2:
					x, y, maxHeight = 0, increasedBoxHeight, halfBoxHeight
				case 3:
					x, y, maxHeight = boxWidth, increasedBoxHeight, halfBoxHeight
				}
				if box != nil {
					for j := scrollPositions[i]; j < len(box) && j < scrollPositions[i]+maxHeight-1; j++ {
						file := box[j]
						style := whiteStyle
						if j == selectedIndices[i] && currentBox == i {
							style = highlightStyle
						}
						// Highlight the best match in the search box
						if currentBox == 2 && bestMatch != nil && file.Name == bestMatch.Name {
							style = commandStyle
						}
						// Display file information with custom colors
						displayFileInfo(screen, x+5, y+j-scrollPositions[i]+1, boxWidth-1, file, style, permissionsTitleStyle, permissionsValueStyle, ownerTitleStyle, ownerValueStyle, executableTitleStyle, executableValueStyle)
					}
				}
			}

			// Define the ASCII art
			asciiArt := `
		 ___               _____  ______
			| |        /\    |  __ \ | ____|
			| |       /  \   | |  | || |__
			| |      / /\ \  | |  | || |__|
			| |___  / ____ \ | |__| || |
			|_____|/_/    \_\|_____/ |_|`

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
			asciiArtX := boxWidth + boxWidth - asciiArtWidth - 1 // Adjusted to place it on the right side
			asciiArtY := asciiBoxYStart - asciiArtHeight + 24    // Adjusted to move it further down

			// Render the ASCII art in the background
			for y, line := range asciiArtLines {
				for x, r := range line {
					screen.SetContent(asciiArtX+x, asciiArtY+y, r, nil, tealStyle)
				}
			}

			// Display user input in the search box
			for i, r := range userInput {
				screen.SetContent(1+i, increasedBoxHeight+1, r, nil, whiteStyle)
			}

			// Display blinking cursor in the search box if it is highlighted
			if currentBox == 2 && cursorVisible {
				screen.SetContent(1+len(userInput), increasedBoxHeight+1, '_', nil, blinkingStyle)
			}

			screen.Show()

			// Handle user input
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					return
				case tcell.KeyTab:
					currentBox = (currentBox + 1) % 4
				case tcell.KeyUp:
					if selectedIndices[currentBox] > 0 {
						selectedIndices[currentBox]--
						if selectedIndices[currentBox] < scrollPositions[currentBox] {
							scrollPositions[currentBox]--
						}
					}
				case tcell.KeyDown:
					// Define maxHeight based on the current box
					var maxHeight int
					switch currentBox {
					case 0, 1:
						maxHeight = increasedBoxHeight
					case 2, 3:
						maxHeight = halfBoxHeight
					}
					if selectedIndices[currentBox] < len(boxes[currentBox])-1 {
						selectedIndices[currentBox]++
						if selectedIndices[currentBox] >= scrollPositions[currentBox]+maxHeight-1 {
							scrollPositions[currentBox]++
						}
					}
				case tcell.KeyEnter:
					if currentBox == 2 && bestMatch != nil { // Search box
						showCommandPopup(screen, bestMatch.Name)
					} else {
						selectedFile := boxes[currentBox][selectedIndices[currentBox]]
						if currentBox == 0 { // Directory
							screen.Fini()
							changeDirectoryAndRerun(selectedFile.Name)
						} else { // File
							showCommandPopup(screen, selectedFile.Name)
						}
					}
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if currentBox == 2 && len(userInput) > 0 {
						userInput = userInput[:len(userInput)-1]
					}
				default:
					if ev.Rune() != 0 {
						if currentBox == 2 {
							userInput = append(userInput, ev.Rune())
						}
					}
				}
			case *tcell.EventMouse:
				x, y := ev.Position()
				if ev.Buttons() == tcell.Button1 {
					// Determine which box was clicked
					if y < increasedBoxHeight {
						if x < boxWidth {
							currentBox = 0 // Directories
						} else {
							currentBox = 1 // Files
						}
					} else if y < increasedBoxHeight+halfBoxHeight {
						if x < boxWidth {
							currentBox = 2 // Search
						} else {
							currentBox = 3 // Dot Files
						}
					}

					// Determine which file or directory was clicked
					if currentBox != 2 { // Not the search box
						var boxStartY int
						switch currentBox {
						case 0:
							boxStartY = 0
						case 1:
							boxStartY = 0
						case 2:
							boxStartY = increasedBoxHeight
						case 3:
							boxStartY = increasedBoxHeight
						}
						clickedIndex := y - boxStartY - 1 + scrollPositions[currentBox]
						if clickedIndex >= 0 && clickedIndex < len(boxes[currentBox]) {
							selectedIndices[currentBox] = clickedIndex
						}
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}
}

// Function to find the best match based on user input
func findBestMatch(directories, files, hiddenFiles []FileInfo, query string) *FileInfo {
	allFiles := append(directories, append(files, hiddenFiles...)...)
	var bestMatch *FileInfo
	for _, file := range allFiles {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			bestMatch = &file
			break
		}
	}
	return bestMatch
}

func drawBorder(screen tcell.Screen, x1, y1, x2, y2 int, style tcell.Style) {
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

func showCommandPopup(screen tcell.Screen, fileName string) {
	var commandInput []rune
	cursorVisible := true
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cursorVisible = !cursorVisible
		default:
			screen.Clear()

			// Get terminal dimensions
			width, height := screen.Size()

			// Define styles
			whiteStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
			//highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorLightSkyBlue).Bold(true)

			// Draw border for the popup
			popupWidth := 50
			popupHeight := 5
			x1 := (width - popupWidth) / 2
			y1 := (height - popupHeight) / 2
			x2 := x1 + popupWidth - 1
			y2 := y1 + popupHeight - 1
			drawBorder(screen, x1, y1, x2, y2, whiteStyle)

			// Display prompt
			prompt := "Enter command:"
			for i, r := range prompt {
				screen.SetContent(x1+2+i, y1+2, r, nil, whiteStyle)
			}

			// Display user input
			for i, r := range commandInput {
				screen.SetContent(x1+2+len(prompt)+1+i, y1+2, r, nil, whiteStyle)
			}

			// Display blinking cursor
			if cursorVisible {
				screen.SetContent(x1+2+len(prompt)+1+len(commandInput), y1+2, '_', nil, whiteStyle)
			}

			screen.Show()

			// Handle user input
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyCtrlC:
					return
				case tcell.KeyEnter:
					screen.Fini()
					runCommandOnFile(string(commandInput), fileName)
				case tcell.KeyBackspace, tcell.KeyBackspace2:
					if len(commandInput) > 0 {
						commandInput = commandInput[:len(commandInput)-1]
					}
				default:
					if ev.Rune() != 0 {
						commandInput = append(commandInput, ev.Rune())
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}
}

func changeDirectoryAndRerun(directory string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("cd %s && lafd", directory))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd %s && lafd", directory))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

func runCommandOnFile(command, fileName string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("%s %s", command, fileName))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("%s %s", command, fileName))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

func filterFiles(files []FileInfo, query string) []FileInfo {
	if query == "" {
		return files
	}
	var filtered []FileInfo
	for _, file := range files {
		if strings.Contains(strings.ToLower(file.Name), strings.ToLower(query)) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func displayFileInfo(screen tcell.Screen, x, y, maxWidth int, file FileInfo, defaultStyle, permissionsTitleStyle, permissionsValueStyle, ownerTitleStyle, ownerValueStyle, executableTitleStyle, executableValueStyle tcell.Style) {
	// Display file name
	for i, r := range file.Name {
		if i < maxWidth {
			screen.SetContent(x+i, y, r, nil, defaultStyle)
		}
	}

	// Display permissions
	permissionsTitle := "Permissions: "
	for i, r := range permissionsTitle {
		if x+20+i < maxWidth {
			screen.SetContent(x+20+i, y, r, nil, permissionsTitleStyle)
		}
	}
	for i, r := range file.Permissions {
		if x+32+i < maxWidth {
			screen.SetContent(x+32+i, y, r, nil, permissionsValueStyle)
		}
	}

	// Display owner
	ownerTitle := "Owner: "
	for i, r := range ownerTitle {
		if x+52+i < maxWidth {
			screen.SetContent(x+52+i, y, r, nil, ownerTitleStyle)
		}
	}
	for i, r := range file.Owner {
		if x+60+i < maxWidth {
			screen.SetContent(x+60+i, y, r, nil, ownerValueStyle)
		}
	}

	// Display executable status
	executableTitle := "Executable: "
	for i, r := range executableTitle {
		if x+80+i < maxWidth {
			screen.SetContent(x+80+i, y, r, nil, executableTitleStyle)
		}
	}
	executableStatus := strings.ToUpper(fmt.Sprint(file.IsExecutable))
	for i, r := range executableStatus {
		if x+92+i < maxWidth {
			screen.SetContent(x+92+i, y, r, nil, executableValueStyle)
		}
	}
}
