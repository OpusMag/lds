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
	Name              string
	Permissions       string
	Owner             string
	IsExecutable      bool
	IsSymlink         bool
	SymlinkTarget     string
	MountPoint        string
	SELinuxContext    string
	GitRepoStatus     string
	HumanReadableDate string
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

	// Buffer for user input in the search box
	var userInput []rune

	// Blinking cursor state
	cursorVisible := true
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Variables to track the current box and scroll positions
	currentBox := 2 // Start with the search box highlighted
	scrollPositions := []int{0, 0, 0, 0}
	selectedIndices := []int{0, 0, 0, 0}

	// Read initial directory contents and update the best match
	query := string(userInput)
	directories, regularFiles, hiddenFiles, bestMatch := readDirectoryAndUpdateBestMatch(screen, query)

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
			labelStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			valueStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
			focusedStyle := tcell.StyleDefault.Foreground(tcell.ColorPaleTurquoise).Bold(true)

			// Calculate box dimensions
			boxWidth := width / 2
			boxHeight := height / 2
			halfBoxHeight := boxHeight / 2
			increasedBoxHeight := boxHeight + halfBoxHeight

			// Draw borders for the boxes
			drawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, tealStyle)                                    // Directories
			drawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, tealStyle)                                // Files
			drawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, tealStyle)     // Search
			drawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, tealStyle) // File Info

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
					screen.SetContent(x+1+j, y, r, nil, whiteStyle)
				}
			}

			// Convert user input to string
			inputStr := string(userInput)

			// Filter file lists based on user input
			filteredDirectories := filterFiles(directories, inputStr)
			filteredFiles := filterFiles(append(regularFiles, hiddenFiles...), inputStr)

			// Find the best match
			bestMatch = findBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

			// Display file names in the directories and files boxes
			boxes := [][]FileInfo{filteredDirectories, filteredFiles, nil}
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
						style := whiteStyle
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
			if currentBox == 0 || currentBox == 1 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				displayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, selectedFile, labelStyle, valueStyle)
			} else if currentBox == 2 && bestMatch != nil {
				displayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, *bestMatch, labelStyle, valueStyle)
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

			// Highlight the selected box
			switch currentBox {
			case 0:
				screen.SetContent(1, 1, '>', nil, focusedStyle)
				drawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, focusedStyle)
			case 1:
				screen.SetContent(1, increasedBoxHeight+1, '>', nil, focusedStyle)
				drawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, focusedStyle)
			case 2:
				screen.SetContent(1, increasedBoxHeight+halfBoxHeight+1, '>', nil, focusedStyle)
				drawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
			case 3:
				screen.SetContent(1, increasedBoxHeight+halfBoxHeight*2+1, '>', nil, focusedStyle)
				drawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
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
					currentBox = (currentBox + 1) % len(titles) // Ensure currentBox wraps correctly
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
						if selectedIndices[currentBox] >= scrollPositions[currentBox]+maxHeight-3 {
							scrollPositions[currentBox]++
						}
					}
				case tcell.KeyEnter:
					if currentBox == 2 && string(userInput) == ".." { // Check for .. command
						err := os.Chdir("..")
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error changing directory: %v\n", err)
						} else {
							// Clear user input
							userInput = []rune{}
							// Reset scroll positions and selected indices
							scrollPositions = []int{0, 0, 0, 0}
							selectedIndices = []int{0, 0, 0, 0}
							// Rerun the directory reading and updating logic
							directories, regularFiles, hiddenFiles, bestMatch = readDirectoryAndUpdateBestMatch(screen, "")
						}
					} else if currentBox == 2 && bestMatch != nil { // Search box
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
							currentBox = 3 // File Info
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

func getSymlinkStatus(file os.DirEntry) (bool, string) {
	if file.Type()&os.ModeSymlink != 0 {
		target, err := os.Readlink(file.Name())
		if err != nil {
			return true, "unknown"
		}
		return true, target
	}
	return false, ""
}

func getMountPoint(info os.FileInfo) string {
	// Implement logic to get mount point details
	// This example assumes a Unix-like system and uses the "findmnt" command
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "--target", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

func getSELinuxContext(info os.FileInfo) string {
	// Implement logic to get SELinux context
	if runtime.GOOS == "linux" {
		cmd := exec.Command("ls", "-Z", info.Name())
		output, err := cmd.Output()
		if err != nil {
			return "N/A"
		}
		parts := strings.Fields(string(output))
		if len(parts) > 3 {
			return parts[3] // SELinux context is usually the 4th field
		}
	}
	return "N/A"
}

func getGitRepoStatus(file os.DirEntry) string {
	// Check if the file is part of a Git repository
	cmd := exec.Command("git", "status", "--porcelain", file.Name())
	output, err := cmd.Output()
	if err != nil {
		return "Not a git repository"
	}
	if len(output) == 0 {
		return "Clean"
	}
	return "Modified"
}

func getHumanReadableDate(modTime time.Time) string {
	duration := time.Since(modTime)
	if duration.Hours() < 24 {
		return fmt.Sprintf("%d hours ago", int(duration.Hours()))
	} else if duration.Hours() < 24*30 {
		return fmt.Sprintf("%d days ago", int(duration.Hours()/24))
	} else if duration.Hours() < 24*365 {
		return fmt.Sprintf("%d months ago", int(duration.Hours()/(24*30)))
	} else {
		return fmt.Sprintf("%d years ago", int(duration.Hours()/(24*365)))
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

func readDirectoryAndUpdateBestMatch(screen tcell.Screen, query string) ([]FileInfo, []FileInfo, []FileInfo, *FileInfo) {
	files, err := os.ReadDir(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(1)
	}

	var directories, regularFiles []FileInfo

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Get file permissions and owner
		var permissions, owner string
		var isExecutable, isSymlink bool
		var symlinkTarget, mountPoint, selinuxContext, gitRepoStatus, humanReadableDate string

		if runtime.GOOS != "windows" {
			stat := info.Sys().(*syscall.Stat_t)
			uid := stat.Uid
			gid := stat.Gid
			usr, _ := user.LookupId(fmt.Sprint(uid))
			grp, _ := user.LookupGroupId(fmt.Sprint(gid))
			permissions = info.Mode().String()
			owner = fmt.Sprintf("%s:%s", usr.Username, grp.Name)
			isExecutable = info.Mode()&0111 != 0
			isSymlink, symlinkTarget = getSymlinkStatus(file)
			mountPoint = getMountPoint(info)
			selinuxContext = getSELinuxContext(info)
			gitRepoStatus = getGitRepoStatus(file)
			humanReadableDate = getHumanReadableDate(info.ModTime())
		} else {
			permissions = "N/A"
			owner = "N/A"
			isExecutable = false
			isSymlink = false
			symlinkTarget = "N/A"
			mountPoint = "N/A"
			selinuxContext = "N/A"
			gitRepoStatus = "N/A"
			humanReadableDate = "N/A"
		}

		fileInfo := FileInfo{
			Name:              info.Name(),
			Permissions:       permissions,
			Owner:             owner,
			IsExecutable:      isExecutable,
			IsSymlink:         isSymlink,
			SymlinkTarget:     symlinkTarget,
			MountPoint:        mountPoint,
			SELinuxContext:    selinuxContext,
			GitRepoStatus:     gitRepoStatus,
			HumanReadableDate: humanReadableDate,
		}

		if info.IsDir() {
			directories = append(directories, fileInfo)
		} else {
			regularFiles = append(regularFiles, fileInfo)
		}
	}

	// Filter file lists based on user input
	filteredDirectories := filterFiles(directories, query)
	filteredFiles := filterFiles(regularFiles, query)

	// Find the best match
	bestMatch := findBestMatch(filteredDirectories, filteredFiles, nil, query)

	return filteredDirectories, filteredFiles, nil, bestMatch
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
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("cd %s && lds", directory))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("cd %s && lds", directory))
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

func displayFileInfo(screen tcell.Screen, x, y, maxWidth int, file FileInfo, labelStyle, valueStyle tcell.Style) {
	// Define labels and values
	labels := []string{
		"Name: ", "Permissions: ", "Owner: ", "Executable: ", "Symlink: ", "Mount: ", "SELinux: ", "Git: ", "Date: ",
	}
	values := []string{
		file.Name, file.Permissions, file.Owner, strings.ToUpper(fmt.Sprint(file.IsExecutable)),
		fmt.Sprintf("-> %s", file.SymlinkTarget), file.MountPoint, file.SELinuxContext, file.GitRepoStatus, file.HumanReadableDate,
	}

	// Display labels and values
	for i, label := range labels {
		labelX := x
		valueX := x + len(label) + 1

		// Display label
		for j, r := range label {
			if labelX+j < maxWidth {
				screen.SetContent(labelX+j, y+i, r, nil, labelStyle)
			}
		}

		// Display value
		for j, r := range values[i] {
			if valueX+j < maxWidth {
				screen.SetContent(valueX+j, y+i, r, nil, valueStyle)
			}
		}
	}
}
