package events

import (
	"log"

	"lds/config"
	"lds/fileops"
	"lds/fsinfo"
	"lds/keymap"
	"lds/ui"
	"lds/utils"

	"github.com/fsnotify/fsnotify"
	"github.com/gdamore/tcell/v2"
)

// WatchConfigFile watches filename for write events, signalling reload, and
// reports unrecoverable errors via errs. The caller is responsible for
// reacting to errs (typically by tearing down and returning).
func WatchConfigFile(filename string, reload chan<- struct{}, errs chan<- error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		errs <- err
		return
	}
	defer watcher.Close()

	if err := watcher.Add(filename); err != nil {
		errs <- err
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				reload <- struct{}{}
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

// KeyContext is the immutable per-event input bundle delivered to Handle.
type KeyContext struct {
	Cfg        *config.Config
	Keymap     keymap.Map
	Layout     ui.Layout
	Boxes      [3][]fsinfo.FileInfo
	CurrentBox int
	UserInput  []rune
	SelIdx     [4]int
	ScrollPos  [4]int
	BestMatch  *fsinfo.FileInfo
}

// KeyResult is the mutable state returned by Handle. CurrentBox == -1
// signals the caller to quit; if Err is non-nil the caller should
// propagate it after Fini.
type KeyResult struct {
	CurrentBox int
	UserInput  []rune
	SelIdx     [4]int
	ScrollPos  [4]int
	BestMatch  *fsinfo.FileInfo
	Err        error
}

// Handle dispatches one tcell event through the keymap, returning the
// updated UI state.
func Handle(screen tcell.Screen, ev tcell.Event, kc KeyContext) KeyResult {
	res := KeyResult{
		CurrentBox: kc.CurrentBox,
		UserInput:  kc.UserInput,
		SelIdx:     kc.SelIdx,
		ScrollPos:  kc.ScrollPos,
		BestMatch:  kc.BestMatch,
	}

	switch ev := ev.(type) {
	case *tcell.EventResize:
		screen.Sync()
		return res
	case *tcell.EventKey:
		action := kc.Keymap.Match(ev)
		switch action {
		case keymap.ActionQuit:
			res.CurrentBox = -1
			return res

		case keymap.ActionNextBox:
			// Cycle through the three focusable boxes (Directories, Files, Search).
			res.CurrentBox = (kc.CurrentBox + 1) % 3
			return res

		case keymap.ActionPreviousBox:
			res.CurrentBox = (kc.CurrentBox + 2) % 3
			return res

		case keymap.ActionSelectUp:
			if kc.CurrentBox < len(res.SelIdx) && res.SelIdx[kc.CurrentBox] > 0 {
				res.SelIdx[kc.CurrentBox]--
				if res.SelIdx[kc.CurrentBox] < res.ScrollPos[kc.CurrentBox] {
					res.ScrollPos[kc.CurrentBox]--
				}
			}
			return res

		case keymap.ActionSelectDown:
			var maxHeight int
			switch kc.CurrentBox {
			case 0, 1:
				maxHeight = kc.Layout.IncreasedBoxHeight
			case 2, 3:
				maxHeight = kc.Layout.HalfBoxHeight
			}
			if kc.CurrentBox < len(kc.Boxes) && res.SelIdx[kc.CurrentBox] < len(kc.Boxes[kc.CurrentBox])-1 {
				res.SelIdx[kc.CurrentBox]++
				if res.SelIdx[kc.CurrentBox] >= res.ScrollPos[kc.CurrentBox]+maxHeight-3 {
					res.ScrollPos[kc.CurrentBox]++
				}
			}
			return res

		case keymap.ActionExecute:
			return execute(screen, kc, res)

		case keymap.ActionBackspace:
			// Backspace: edit search input only when search box is focused;
			// otherwise treat as parent-dir navigation (issue 7, §3.5).
			if kc.CurrentBox == 2 {
				if len(res.UserInput) > 0 {
					res.UserInput = res.UserInput[:len(res.UserInput)-1]
				}
				return res
			}
			fallthrough

		case keymap.ActionParentDir, keymap.ActionGoBack:
			screen.Fini()
			utils.ChangeDirectoryAndRerun("", true)
			res.CurrentBox = -1
			return res

		case keymap.ActionRename:
			return mutateFile(screen, kc, res, "Rename to:", fileops.RenameFile, "rename", "renamed")

		case keymap.ActionMove:
			return mutateFile(screen, kc, res, "Move to:", fileops.MoveFile, "move", "moved")

		case keymap.ActionCopy:
			return mutateFile(screen, kc, res, "Copy to:", fileops.CopyFile, "copy", "copied")

		case keymap.ActionDelete:
			if kc.CurrentBox == 1 && len(kc.Boxes[kc.CurrentBox]) > 0 {
				selected := kc.Boxes[kc.CurrentBox][res.SelIdx[kc.CurrentBox]]
				if err := fileops.DeleteFile(selected.Name); err != nil {
					log.Println("Error deleting file:", err)
				} else {
					log.Println("File deleted successfully")
				}
			}
			return res

		case keymap.ActionNone:
			// Fall through: rune append in search box.
			if kc.CurrentBox == 2 && ev.Key() == tcell.KeyRune {
				res.UserInput = append(res.UserInput, ev.Rune())
			}
			return res
		}
	}
	return res
}

func execute(screen tcell.Screen, kc KeyContext, res KeyResult) KeyResult {
	switch {
	case kc.CurrentBox == 2 && kc.BestMatch != nil:
		if kc.BestMatch.FileType == "Directory" {
			screen.Fini()
			utils.ChangeDirectoryAndRerun(kc.BestMatch.Name, false)
			res.CurrentBox = -1
			return res
		}
		screen.Fini()
		fileops.OpenFileInEditor(kc.Cfg.PreferredEditor, kc.BestMatch.Name)
		if err := screen.Init(); err != nil {
			res.CurrentBox = -1
			res.Err = err
			return res
		}
	case kc.CurrentBox == 1 && len(kc.Boxes[1]) > 0:
		selected := kc.Boxes[1][res.SelIdx[1]]
		if selected.FileType == "Directory" {
			screen.Fini()
			utils.ChangeDirectoryAndRerun(selected.Name, false)
			res.CurrentBox = -1
			return res
		}
		screen.Fini()
		fileops.OpenFileInEditor(kc.Cfg.PreferredEditor, selected.Name)
		if err := screen.Init(); err != nil {
			res.CurrentBox = -1
			res.Err = err
			return res
		}
	case kc.CurrentBox == 0 && len(kc.Boxes[0]) > 0:
		selected := kc.Boxes[0][res.SelIdx[0]]
		screen.Fini()
		utils.ChangeDirectoryAndRerun(selected.Name, false)
		res.CurrentBox = -1
		return res
	}
	return res
}

func mutateFile(screen tcell.Screen, kc KeyContext, res KeyResult, prompt string, op func(string, string) error, verb, past string) KeyResult {
	if kc.CurrentBox != 1 || len(kc.Boxes[1]) == 0 {
		return res
	}
	selected := kc.Boxes[1][res.SelIdx[1]]
	target := PromptForInput(screen, prompt)
	if target == "" {
		return res
	}
	if err := op(selected.Name, target); err != nil {
		log.Printf("Error %sing file: %v", verb, err)
	} else {
		log.Printf("File %s successfully", past)
	}
	return res
}
