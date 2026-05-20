package app

import (
	"log"
	"time"

	"lds/config"
	"lds/events"
	"lds/fsinfo"
	"lds/keymap"
	"lds/ui"
	"lds/utils"

	"github.com/gdamore/tcell/v2"
)

// tickEvent, reloadEvent, errEvent are internal events posted into the
// screen's event queue so that App.Run has a single PollEvent reader.
// This avoids racing with screen.PollEvent calls made by PromptForInput
// and with screen.Fini/Init cycles around opening the editor.
type tickEvent struct{ t time.Time }

func (e *tickEvent) When() time.Time { return e.t }

type reloadEvent struct{ t time.Time }

func (e *reloadEvent) When() time.Time { return e.t }

type errEvent struct {
	t   time.Time
	err error
}

func (e *errEvent) When() time.Time { return e.t }

// App is the running TUI session. One instance per process.
type App struct {
	cfg     *config.Config
	cfgPath string
	screen  tcell.Screen
	reload  chan struct{}
	errs    chan error

	currentBox      int
	userInput       []rune
	cursorVisible   bool
	scrollPositions [4]int
	selectedIndices [4]int

	directories  []fsinfo.FileInfo
	regularFiles []fsinfo.FileInfo
	bestMatch    *fsinfo.FileInfo

	layout ui.Layout
	styles ui.Styles
	keymap keymap.Map

	blinkTicker *time.Ticker
}

// New constructs an App. cfg/cfgPath are pre-resolved; reload and errs are
// signalled by the config watcher goroutine. The screen must already be
// initialised by the caller; the caller is responsible for screen.Fini().
func New(cfg *config.Config, cfgPath string, screen tcell.Screen, reload chan struct{}, errs chan error) (*App, error) {
	a := &App{
		cfg:           cfg,
		cfgPath:       cfgPath,
		screen:        screen,
		reload:        reload,
		errs:          errs,
		currentBox:    2,
		cursorVisible: true,
	}
	a.applyConfig()
	if err := a.scanDir(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *App) applyConfig() {
	a.styles = ui.NewStyles(a.cfg)
	km, err := keymap.Build(a.cfg)
	if err != nil {
		log.Printf("keymap build error: %v", err)
	}
	a.keymap = km
}

func (a *App) scanDir() error {
	directories, regularFiles, bestMatch, err := utils.ReadDirectoryAndUpdateBestMatch("", a.cfg.FileFilters.ShowHiddenFiles)
	if err != nil {
		return err
	}
	a.directories = directories
	a.regularFiles = regularFiles
	a.bestMatch = bestMatch
	return nil
}

func (a *App) reloadConfig() {
	cfg, err := config.LoadConfig(a.cfgPath)
	if err != nil {
		log.Println("Error reloading config:", err)
		return
	}
	a.cfg = cfg
	a.applyConfig()
	_ = a.scanDir()
	log.Println("Config reloaded")
}

// Run is the main loop. It returns when the user quits or an
// unrecoverable error occurs. Screen.Fini is the caller's responsibility.
//
// Implementation note: a background goroutine forwards ticker/reload/errs
// signals into the screen's event queue via PostEvent. The main loop
// is the sole caller of screen.PollEvent, which means (a) PromptForInput
// (in events.Handle) can safely call PollEvent without racing, and
// (b) the Fini->editor->Init cycle does not leak/close any reader
// goroutine.
func (a *App) Run() error {
	a.blinkTicker = time.NewTicker(500 * time.Millisecond)
	defer a.blinkTicker.Stop()

	stop := make(chan struct{})
	defer close(stop)

	go func() {
		for {
			select {
			case <-stop:
				return
			case <-a.blinkTicker.C:
				_ = a.screen.PostEvent(&tickEvent{t: time.Now()})
			case <-a.reload:
				_ = a.screen.PostEvent(&reloadEvent{t: time.Now()})
			case err := <-a.errs:
				_ = a.screen.PostEvent(&errEvent{t: time.Now(), err: err})
			}
		}
	}()

	for {
		a.render()
		ev := a.screen.PollEvent()
		if ev == nil {
			// Screen was finalised externally; exit cleanly.
			return nil
		}
		switch ev := ev.(type) {
		case *tickEvent:
			a.cursorVisible = !a.cursorVisible
		case *reloadEvent:
			a.reloadConfig()
		case *errEvent:
			return ev.err
		default:
			if quit, err := a.handleEvent(ev); quit {
				return err
			}
		}
	}
}

func (a *App) handleEvent(ev tcell.Event) (bool, error) {
	inputStr := string(a.userInput)
	filteredDirectories := utils.FilterFiles(a.directories, inputStr)
	filteredFiles := utils.FilterFiles(a.regularFiles, inputStr)
	a.bestMatch = utils.FindBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

	kc := events.KeyContext{
		Cfg:        a.cfg,
		Keymap:     a.keymap,
		Layout:     a.layout,
		Boxes:      [3][]fsinfo.FileInfo{filteredDirectories, filteredFiles, nil},
		CurrentBox: a.currentBox,
		UserInput:  a.userInput,
		SelIdx:     a.selectedIndices,
		ScrollPos:  a.scrollPositions,
		BestMatch:  a.bestMatch,
	}
	res := events.Handle(a.screen, ev, kc)
	a.currentBox = res.CurrentBox
	a.userInput = res.UserInput
	a.selectedIndices = res.SelIdx
	a.scrollPositions = res.ScrollPos
	a.bestMatch = res.BestMatch
	if res.CurrentBox == -1 {
		return true, res.Err
	}
	return false, nil
}

func (a *App) render() {
	a.screen.Clear()
	width, height := a.screen.Size()
	a.layout = ui.ComputeLayout(width, height)
	l := a.layout

	ui.DrawBorder(a.screen, 0, 0, l.BoxWidth-1, l.IncreasedBoxHeight-1, a.styles.Border)
	ui.DrawBorder(a.screen, l.BoxWidth, 0, width-1, l.IncreasedBoxHeight-1, a.styles.Border)
	ui.DrawBorder(a.screen, 0, l.IncreasedBoxHeight, l.BoxWidth-1, l.IncreasedBoxHeight+l.HalfBoxHeight-1, a.styles.Border)
	ui.DrawBorder(a.screen, l.BoxWidth, l.IncreasedBoxHeight, width-1, l.IncreasedBoxHeight+l.HalfBoxHeight-1, a.styles.Border)

	ui.DrawTitles(a.screen, l, a.styles.Text)

	inputStr := string(a.userInput)
	filteredDirectories := utils.FilterFiles(a.directories, inputStr)
	filteredFiles := utils.FilterFiles(a.regularFiles, inputStr)
	a.bestMatch = utils.FindBestMatch(filteredDirectories, filteredFiles, nil, inputStr)

	ui.DrawBox(a.screen, 0, 0, l.BoxWidth, l.IncreasedBoxHeight, filteredDirectories, a.selectedIndices[0], a.scrollPositions[0], a.styles.Text, a.styles.Highlight, a.currentBox == 0)
	ui.DrawBox(a.screen, l.BoxWidth, 0, width-l.BoxWidth, l.IncreasedBoxHeight, filteredFiles, a.selectedIndices[1], a.scrollPositions[1], a.styles.Text, a.styles.Highlight, a.currentBox == 1)

	var highlighted *fsinfo.FileInfo
	switch {
	case a.currentBox == 1 && len(filteredFiles) > 0:
		f := filteredFiles[a.selectedIndices[1]]
		highlighted = &f
	case a.currentBox == 0 && len(filteredDirectories) > 0:
		f := filteredDirectories[a.selectedIndices[0]]
		highlighted = &f
	case a.currentBox == 2 && a.bestMatch != nil:
		highlighted = a.bestMatch
	}
	if highlighted != nil {
		utils.EnrichFileInfo(highlighted)
		ui.DisplayFileInfo(a.screen, l.BoxWidth+3, l.IncreasedBoxHeight+1, width-1, *highlighted, a.styles.Label, a.styles.Value)
		if highlighted.FileType == "Regular File" {
			ui.DrawFilePreview(a.screen, l.BoxWidth, l.IncreasedBoxHeight, width-l.BoxWidth, l.HalfBoxHeight, *highlighted, a.styles.Text)
		}
	}

	ui.DrawASCIIArt(a.screen, a.styles.Border)

	for i, r := range a.userInput {
		a.screen.SetContent(1+i, l.IncreasedBoxHeight+1, r, nil, a.styles.Text)
	}
	if a.currentBox == 2 && a.cursorVisible {
		a.screen.SetContent(1+len(a.userInput), l.IncreasedBoxHeight+1, '_', nil, a.styles.Blinking)
	}

	switch a.currentBox {
	case 0:
		ui.DrawBorder(a.screen, 0, 0, l.BoxWidth-1, l.IncreasedBoxHeight-1, a.styles.Focused)
	case 1:
		ui.DrawBorder(a.screen, l.BoxWidth, 0, width-1, l.IncreasedBoxHeight-1, a.styles.Focused)
	case 2:
		ui.DrawBorder(a.screen, 0, l.IncreasedBoxHeight, l.BoxWidth-1, l.IncreasedBoxHeight+l.HalfBoxHeight-1, a.styles.Focused)
	case 3:
		ui.DrawBorder(a.screen, l.BoxWidth, l.IncreasedBoxHeight, width-1, l.IncreasedBoxHeight+l.HalfBoxHeight-1, a.styles.Focused)
	}

	ui.DrawStatusBar(a.screen, width, height, a.styles.Text)

	a.screen.Show()
}
