package events

import (
	"log"

	"lds/config"
	"lds/fileops"
	"lds/ui"
	"lds/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell/v2"
)

func WatchConfigFile(filename string, reloadConfig chan<- struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add(filename)
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				reloadConfig <- struct{}{}
			}
		case err := <-watcher.Errors:
			log.Println("error:", err)
		}
	}
}

func PromptForInput(screen tcell.Screen, prompt string) string {
	screen.Clear()
	ui.DrawPrompt(screen, prompt)
	screen.Show()

	var input []rune
	for {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyEnter:
				return string(input)
			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(input) > 0 {
					input = input[:len(input)-1]
				}
			default:
				if ev.Rune() != 0 {
					input = append(input, ev.Rune())
				}
			}
		case *tcell.EventResize:
			screen.Sync()
		}
		ui.DrawPrompt(screen, prompt+string(input))
		screen.Show()
	}
}

func HandleUserInput(screen tcell.Screen, cfg *config.Config, currentBox int, userInput []rune, boxes [][]config.FileInfo, selectedIndices []int, scrollPositions []int, bestMatch *config.FileInfo) (int, []rune, []int, []int, *config.FileInfo) {
	ev := screen.PollEvent()
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyCtrlC:
			return -1, userInput, selectedIndices, scrollPositions, bestMatch
		case tcell.KeyEscape:
			if currentBox == 0 && len(boxes[currentBox]) > 0 { // Directory box
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				up := true
				screen.Fini()
				utils.ChangeDirectoryAndRerun(selectedFile.Name, up)
			}
		case tcell.KeyTab:
			currentBox = (currentBox + 1) % len(ui.Titles)
		case tcell.KeyUp:
			if currentBox < len(selectedIndices) && selectedIndices[currentBox] > 0 {
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
				maxHeight = ui.IncreasedBoxHeight
			case 2, 3:
				maxHeight = ui.HalfBoxHeight
			}
			if currentBox < len(selectedIndices) && selectedIndices[currentBox] < len(boxes[currentBox])-1 {
				selectedIndices[currentBox]++
				if selectedIndices[currentBox] >= scrollPositions[currentBox]+maxHeight-3 {
					scrollPositions[currentBox]++
				}
			}
		case tcell.KeyEnter:
			if currentBox == 2 && bestMatch != nil { // Search box
				fileops.OpenFileInEditor(cfg.PreferredEditor, bestMatch.Name)
			} else if currentBox == 1 && len(boxes[currentBox]) > 0 { // File box
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				fileops.OpenFileInEditor(cfg.PreferredEditor, selectedFile.Name)
			} else if currentBox == 0 && len(boxes[currentBox]) > 0 { // Directory box
				selectedFile := boxes[currentBox][selectedIndices[currentBox]]
				up := false
				screen.Fini()
				utils.ChangeDirectoryAndRerun(selectedFile.Name, up)
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if currentBox == 2 && len(userInput) > 0 {
				userInput = userInput[:len(userInput)-1]
			}
		case tcell.KeyRune:
			if ev.Rune() == 'r' && ev.Modifiers() == (tcell.ModAlt) {
				if currentBox == 1 && len(boxes[currentBox]) > 0 {
					selectedFile := boxes[currentBox][selectedIndices[currentBox]]
					newName := PromptForInput(screen, "Rename to:")
					if newName != "" {
						err := fileops.RenameFile(selectedFile.Name, newName)
						if err != nil {
							log.Println("Error renaming file:", err)
						} else {
							log.Println("File renamed successfully")
						}
					}
				}
			} else if ev.Rune() == 'm' && ev.Modifiers() == (tcell.ModAlt) {
				if currentBox == 1 && len(boxes[currentBox]) > 0 {
					selectedFile := boxes[currentBox][selectedIndices[currentBox]]
					newLocation := PromptForInput(screen, "Move to:")
					if newLocation != "" {
						err := fileops.MoveFile(selectedFile.Name, newLocation)
						if err != nil {
							log.Println("Error moving file:", err)
						} else {
							log.Println("File moved successfully")
						}
					}
				}
			} else if ev.Rune() == 'd' && ev.Modifiers() == (tcell.ModAlt) {
				if currentBox == 1 && len(boxes[currentBox]) > 0 {
					selectedFile := boxes[currentBox][selectedIndices[currentBox]]
					err := fileops.DeleteFile(selectedFile.Name)
					if err != nil {
						log.Println("Error deleting file:", err)
					} else {
						log.Println("File deleted successfully")
					}
				}
			} else if ev.Rune() == 'c' && ev.Modifiers() == (tcell.ModAlt) {
				if currentBox == 1 && len(boxes[currentBox]) > 0 {
					selectedFile := boxes[currentBox][selectedIndices[currentBox]]
					newLocation := PromptForInput(screen, "Copy to:")
					if newLocation != "" {
						err := fileops.CopyFile(selectedFile.Name, newLocation)
						if err != nil {
							log.Println("Error copying file:", err)
						} else {
							log.Println("File copied successfully")
						}
					}
				}
			} else {
				if currentBox == 2 {
					userInput = append(userInput, ev.Rune())
				}
			}
		}
	case *tcell.EventResize:
		screen.Sync()
	}
	return currentBox, userInput, selectedIndices, scrollPositions, bestMatch
}
