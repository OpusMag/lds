package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Config struct {
	Colors struct {
		Text      string `json:"text"`
		Border    string `json:"border"`
		Highlight string `json:"highlight"`
		Command   string `json:"command"`
		Blinking  string `json:"blinking"`
		Label     string `json:"label"`
		Value     string `json:"value"`
		Focused   string `json:"focused"`
	} `json:"colors"`
	Navigation struct {
		Up    string `json:"up"`
		Down  string `json:"down"`
		Left  string `json:"left"`
		Right string `json:"right"`
	} `json:"navigation"`
	KeyBindings struct {
		Quit        string `json:"quit"`
		NextBox     string `json:"nextBox"`
		PreviousBox string `json:"previousBox"`
		SelectUp    string `json:"selectUp"`
		SelectDown  string `json:"selectDown"`
		Execute     string `json:"execute"`
		Backspace   string `json:"backspace"`
	} `json:"keyBindings"`
	Theme  string `json:"theme"`
	Themes struct {
		Dark struct {
			Background string `json:"background"`
			Foreground string `json:"foreground"`
			Highlight  string `json:"highlight"`
			Cursor     string `json:"cursor"`
		} `json:"dark"`
		Light struct {
			Background string `json:"background"`
			Foreground string `json:"foreground"`
			Highlight  string `json:"highlight"`
			Cursor     string `json:"cursor"`
		} `json:"light"`
	} `json:"themes"`
	Font struct {
		Size  int    `json:"size"`
		Style string `json:"style"`
	} `json:"font"`
	Layout struct {
		DirectoryBoxWidth int `json:"directoryBoxWidth"`
		FileBoxWidth      int `json:"fileBoxWidth"`
		SearchBoxHeight   int `json:"searchBoxHeight"`
		FileInfoBoxHeight int `json:"fileInfoBoxHeight"`
	} `json:"layout"`
	DefaultDirectory string `json:"defaultDirectory"`
	FileFilters      struct {
		ShowHiddenFiles bool     `json:"showHiddenFiles"`
		FileExtensions  []string `json:"fileExtensions"`
	} `json:"fileFilters"`
	Notifications struct {
		Enabled  bool `json:"enabled"`
		Duration int  `json:"duration"`
	} `json:"notifications"`
	Language string `json:"language"`
	AutoSave struct {
		Enabled  bool `json:"enabled"`
		Interval int  `json:"interval"`
	} `json:"autoSave"`
	Logging struct {
		Level string `json:"level"`
		File  string `json:"file"`
	} `json:"logging"`
	PreferredEditor string `json:"preferredEditor"`
}

type FileInfo struct {
	Name            string
	Permissions     string
	Owner           string
	IsExecutable    bool
	IsSymlink       bool
	SymlinkTarget   string
	MountPoint      string
	SELinuxContext  string
	GitRepoStatus   string
	getLastModified string
	Size            int64
	FileType        string
	LastAccessTime  string
	CreationTime    string
	Inode           uint64
	HardLinksCount  uint64
}

func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func openFileInEditor(editor, fileName string) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/C", fmt.Sprintf("%s %s", editor, fileName))
	} else {
		cmd = exec.Command("sh", "-c", fmt.Sprintf("%s %s", editor, fileName))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	os.Exit(0)
}

func readFileContents(fileName string) (string, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func main() {
	// Load config
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
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

			// Define styles using config colors
			textStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Text))
			borderStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Border))
			highlightStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Highlight)).Bold(true)
			commandStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Command)).Bold(true)
			blinkingStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Blinking)).Bold(true)
			labelStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Label))
			valueStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Value)).Bold(true)
			focusedStyle := tcell.StyleDefault.Foreground(tcell.GetColor(config.Colors.Focused)).Bold(true)

			// Calculate box dimensions
			boxWidth := width / 2
			boxHeight := height / 2
			halfBoxHeight := boxHeight / 2
			increasedBoxHeight := boxHeight + halfBoxHeight

			// Draw borders for the boxes
			drawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, borderStyle)                                    // Directories
			drawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, borderStyle)                                // Files
			drawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, borderStyle)     // Search
			drawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, borderStyle) // File Info

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
				displayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, selectedFile, labelStyle, valueStyle)
			} else if currentBox == 2 && bestMatch != nil {
				displayFileInfo(screen, boxWidth+3, increasedBoxHeight+1, width-1, *bestMatch, labelStyle, valueStyle)
			}

			// Display file contents in the directory box if a file is highlighted
			if currentBox == 1 && len(boxes[currentBox]) > 0 {
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				fileContents, err := readFileContents(selectedFile.Name)
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
				//screen.SetContent(1, 1, ' ', nil, focusedStyle)
				drawBorder(screen, 0, 0, boxWidth-1, increasedBoxHeight-1, focusedStyle)
			case 1:
				//screen.SetContent(1, increasedBoxHeight+1, ' ', nil, focusedStyle)
				drawBorder(screen, boxWidth, 0, width-1, increasedBoxHeight-1, focusedStyle)
			case 2:
				//screen.SetContent(1, increasedBoxHeight+1, ' ', nil, focusedStyle)
				drawBorder(screen, 0, increasedBoxHeight, boxWidth-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
			case 3:
				//screen.SetContent(1, increasedBoxHeight+halfBoxHeight+1, ' ', nil, focusedStyle)
				drawBorder(screen, boxWidth, increasedBoxHeight, width-1, increasedBoxHeight+halfBoxHeight-1, focusedStyle)
			}

			screen.Show()

			// Handle user input
			ev := screen.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyCtrlC:
					return
				case tcell.KeyEscape:
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
					if currentBox == 2 && bestMatch != nil { // Search box
						openFileInEditor(config.PreferredEditor, bestMatch.Name)
					} else if currentBox == 1 && len(boxes[currentBox]) > 0 { // File box
						selectedFile := boxes[currentBox][selectedIndices[currentBox]]
						openFileInEditor(config.PreferredEditor, selectedFile.Name)
					} else if currentBox == 0 && len(boxes[currentBox]) > 0 { // Directory box
						selectedFile := boxes[currentBox][selectedIndices[currentBox]]
						screen.Fini()
						changeDirectoryAndRerun(selectedFile.Name)
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
			case *tcell.EventResize:
				screen.Sync()
			}
		}
	}
}

// Check if the file is a symlink and get the target
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

// Get the mount point of the file
func getMountPoint(info os.FileInfo) string {
	if runtime.GOOS == "windows" {
		return "N/A" // Mount points are not applicable on Windows in the same way
	}
	// Implement logic to get mount point details for Unix-like systems
	cmd := exec.Command("findmnt", "-n", "-o", "TARGET", "--target", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

// Get the SELinux context of the file
func getSELinuxContext(info os.FileInfo) string {
	if runtime.GOOS != "linux" {
		return "N/A" // SELinux is specific to Linux
	}
	// Implement logic to get SELinux context for Linux
	cmd := exec.Command("ls", "-Z", info.Name())
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	parts := strings.Fields(string(output))
	if len(parts) > 3 {
		return parts[3] // SELinux context is usually the 4th field
	}
	return "N/A"
}

// Check if the file or directory is part of a Git repository
func getGitRepoStatus(file os.DirEntry) string {
	if file.IsDir() {
		// Check if the directory contains a .git directory
		gitDir := fmt.Sprintf("%s/.git", file.Name())
		if _, err := os.Stat(gitDir); os.IsNotExist(err) {
			return "Not a git repository"
		}
		return "Git repository"
	} else {
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
}

// How long since the file was last modified
func getLastModified(modTime time.Time) string {
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

func getFileType(info os.FileInfo) string {
	switch mode := info.Mode(); {
	case mode.IsRegular():
		return "Regular File"
	case mode.IsDir():
		return "Directory"
	case mode&os.ModeSymlink != 0:
		return "Symlink"
	case mode&os.ModeNamedPipe != 0:
		return "Named Pipe"
	case mode&os.ModeSocket != 0:
		return "Socket"
	case mode&os.ModeDevice != 0:
		return "Device"
	default:
		return "Unknown"
	}
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
		var symlinkTarget, mountPoint, selinuxContext, gitRepoStatus, LastModified string
		var size int64
		var fileType, lastAccessTime, creationTime string
		var inode, hardLinksCount uint64

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
			LastModified = getLastModified(info.ModTime())
			size = info.Size()
			fileType = getFileType(info)
			lastAccessTime = getLastModified(time.Unix(stat.Atim.Sec, stat.Atim.Nsec))
			creationTime = getLastModified(time.Unix(stat.Ctim.Sec, stat.Ctim.Nsec))
			inode = stat.Ino
			hardLinksCount = stat.Nlink
		} else {
			permissions = "N/A"
			owner = "N/A"
			isExecutable = false
			isSymlink = false
			symlinkTarget = "N/A"
			mountPoint = "N/A"
			selinuxContext = "N/A"
			gitRepoStatus = "N/A"
			LastModified = getLastModified(info.ModTime())
			size = info.Size()
			fileType = getFileType(info)
			lastAccessTime = "N/A"
			creationTime = "N/A"
			inode = 0
			hardLinksCount = 0
		}

		fileInfo := FileInfo{
			Name:            info.Name(),
			Permissions:     permissions,
			Owner:           owner,
			IsExecutable:    isExecutable,
			IsSymlink:       isSymlink,
			SymlinkTarget:   symlinkTarget,
			MountPoint:      mountPoint,
			SELinuxContext:  selinuxContext,
			GitRepoStatus:   gitRepoStatus,
			getLastModified: LastModified,
			Size:            size,
			FileType:        fileType,
			LastAccessTime:  lastAccessTime,
			CreationTime:    creationTime,
			Inode:           inode,
			HardLinksCount:  hardLinksCount,
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
	// Define labels and values for the first column
	labelsColumn1 := []string{
		"Name: ", "Owner: ", "User permissions: ", "Group permissions: ", "Others permissions: ", "File Type: ", "Size: ", "Last Access: ", "Creation Time: ",
	}
	valuesColumn1 := []string{
		file.Name, file.Owner, file.Permissions[1:4], file.Permissions[4:7], file.Permissions[7:10], file.FileType, fmt.Sprintf("%d bytes", file.Size), file.LastAccessTime, file.CreationTime,
	}

	// Define labels and values for the second column
	labelsColumn2 := []string{
		"Executable: ", "Git: ", "Mount: ", "Hard Links: ", "Inode: ", "Symlink: ", "Symlink Target: ",
	}
	valuesColumn2 := []string{
		strings.ToUpper(fmt.Sprint(file.IsExecutable)), file.GitRepoStatus, file.MountPoint, fmt.Sprintf("%d", file.HardLinksCount), fmt.Sprintf("%d", file.Inode), fmt.Sprintf("%t", file.IsSymlink), file.SymlinkTarget,
	}

	// Calculate the number of lines available in the file info box
	width, height := screen.Size()
	boxWidth := width / 2
	boxHeight := height / 2

	// Display labels and values in two columns
	columnWidth := boxWidth / 2

	// Display first column
	for i, label := range labelsColumn1 {
		labelX := x
		valueX := x + len(label) + 1
		row := y + i

		// Wrap text if it exceeds the box height
		if row >= y+boxHeight {
			break
		}

		// Display label
		for j, r := range label {
			if labelX+j < maxWidth {
				screen.SetContent(labelX+j, row, r, nil, labelStyle)
			}
		}

		// Display value
		value := valuesColumn1[i]
		for j, r := range value {
			if valueX+j < maxWidth {
				screen.SetContent(valueX+j, row, r, nil, valueStyle)
			}
		}
	}

	// Display second column
	for i, label := range labelsColumn2 {
		labelX := x + columnWidth
		valueX := labelX + len(label) + 1
		row := y + i

		// Wrap text if it exceeds the box height
		if row >= y+boxHeight {
			break
		}

		// Display label
		for j, r := range label {
			if labelX+j < maxWidth {
				screen.SetContent(labelX+j, row, r, nil, labelStyle)
			}
		}

		// Display value
		value := valuesColumn2[i]
		for j, r := range value {
			if valueX+j < maxWidth {
				screen.SetContent(valueX+j, row, r, nil, valueStyle)
			}
		}
	}
}

// Helper function to get the permission character or a dash if out of range
func getPermissionChar(permissions string, index int) string {
	if index < len(permissions) {
		return string(permissions[index])
	}
	return "-"
}
