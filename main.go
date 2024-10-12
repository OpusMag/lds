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

	// Main loop
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
			permissionsTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)
			permissionsValueStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
			ownerTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorFuchsia)
			ownerValueStyle := tcell.StyleDefault.Foreground(tcell.ColorTeal)
			executableTitleStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
			executableValueStyle := tcell.StyleDefault.Foreground(tcell.ColorBlue)

			// Calculate box dimensions
			boxWidth := width / 2
			boxHeight := height / 2
			quarterBoxHeight := boxHeight / 4

			// Draw borders for the boxes
			drawBorder(screen, 0, 0, boxWidth-1, height-1, tealStyle)                           // Directories
			drawBorder(screen, boxWidth, 0, width-1, height-1, tealStyle)                       // Files
			drawBorder(screen, 0, height-quarterBoxHeight, boxWidth-1, height-1, tealStyle)     // Search
			drawBorder(screen, boxWidth, height-quarterBoxHeight, width-1, height-1, tealStyle) // Dot Files

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
					x, y = 0, height-quarterBoxHeight
				case 3:
					x, y = boxWidth, height-quarterBoxHeight
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

			// Display file information in the boxes
			boxes := [][]FileInfo{filteredDirectories, filteredFiles, nil, filteredHiddenFiles}
			for i, box := range boxes {
				var x, y, maxHeight int
				switch i {
				case 0:
					x, y, maxHeight = 0, 0, height-quarterBoxHeight
				case 1:
					x, y, maxHeight = boxWidth, 0, height-quarterBoxHeight
				case 2:
					x, y, maxHeight = 0, height-quarterBoxHeight, quarterBoxHeight
				case 3:
					x, y, maxHeight = boxWidth, height-quarterBoxHeight, quarterBoxHeight
				}
				if box != nil {
					for j := scrollPositions[i]; j < len(box) && j < scrollPositions[i]+maxHeight-1; j++ {
						file := box[j]
						style := whiteStyle
						if j == selectedIndices[i] && currentBox == i {
							style = highlightStyle
						}
						// Display file information with custom colors
						displayFileInfo(screen, x+5, y+j-scrollPositions[i]+1, boxWidth-1, file, style, permissionsTitleStyle, permissionsValueStyle, ownerTitleStyle, ownerValueStyle, executableTitleStyle, executableValueStyle)
					}
				}
			}

			// Display user input in the search box
			for i, r := range userInput {
				screen.SetContent(1+i, height-quarterBoxHeight+1, r, nil, whiteStyle)
			}

			// Display blinking cursor in the search box if it is highlighted
			if currentBox == 2 && cursorVisible {
				screen.SetContent(1+len(userInput), height-quarterBoxHeight+1, '_', nil, whiteStyle)
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
						maxHeight = height - quarterBoxHeight
					case 2, 3:
						maxHeight = quarterBoxHeight
					}
					if selectedIndices[currentBox] < len(boxes[currentBox])-1 {
						selectedIndices[currentBox]++
						if selectedIndices[currentBox] >= scrollPositions[currentBox]+maxHeight-1 {
							scrollPositions[currentBox]++
						}
					}
				case tcell.KeyEnter:
					if currentBox == 2 { // Search box
						if inputStr == ".." {
							screen.Fini()
							changeDirectoryAndRerun("..")
						}
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
					if len(userInput) > 0 {
						userInput = userInput[:len(userInput)-1]
					}
				default:
					if ev.Rune() != 0 {
						userInput = append(userInput, ev.Rune())
					}
				}
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}
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
	commands := []string{"cat", "head", "tail", "nano", "vim", "nvim"}
	selectedIndex := 0

	for {
		screen.Clear()

		// Get terminal dimensions
		width, height := screen.Size()

		// Define styles
		whiteStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		highlightStyle := tcell.StyleDefault.Foreground(tcell.ColorLightSkyBlue).Bold(true)

		// Draw border for the popup
		popupWidth := 30
		popupHeight := len(commands) + 2
		x1 := (width - popupWidth) / 2
		y1 := (height - popupHeight) / 2
		x2 := x1 + popupWidth - 1
		y2 := y1 + popupHeight - 1
		drawBorder(screen, x1, y1, x2, y2, whiteStyle)

		// Display commands in the popup
		for i, cmd := range commands {
			style := whiteStyle
			if i == selectedIndex {
				style = highlightStyle
			}
			for j, r := range cmd {
				screen.SetContent(x1+1+j, y1+1+i, r, nil, style)
			}
		}

		screen.Show()

		// Handle user input
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEscape, tcell.KeyCtrlC:
				return
			case tcell.KeyUp:
				if selectedIndex > 0 {
					selectedIndex--
				}
			case tcell.KeyDown:
				if selectedIndex < len(commands)-1 {
					selectedIndex++
				}
			case tcell.KeyEnter:
				screen.Fini()
				runCommandOnFile(commands[selectedIndex], fileName)
			}
		case *tcell.EventResize:
			screen.Sync()
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
