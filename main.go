package main

import (
	"fmt"
	"log"
	"os"

	"lds/app"
	"lds/config"
	"lds/events"
	"lds/logging"

	"github.com/gdamore/tcell/v2"
)

func main() {
	cfgPath, err := config.FindConfigFile()
	if err != nil {
		if cerr, ok := err.(*config.ConfigError); ok {
			fmt.Fprintf(os.Stderr, "Error: %s\n", cerr.Message)
			fmt.Fprintln(os.Stderr, "Searched in the following locations:")
			for _, p := range cerr.Paths {
				fmt.Fprintf(os.Stderr, "  - %s\n", p)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error finding config: %v\n", err)
		}
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Config file found at: %s", cfgPath)
	_ = os.Setenv("LDS_CONFIG", cfgPath)
	if err := logging.SetupLogging(cfg.Logging.File); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logging: %v\n", err)
		os.Exit(1)
	}

	reload := make(chan struct{}, 1)
	errs := make(chan error, 1)
	watcherStop := make(chan struct{})
	defer close(watcherStop)
	go events.WatchConfigFile(cfgPath, reload, errs, watcherStop)

	screen, err := tcell.NewScreen()
	if err != nil {
		logging.LogErrorAndExit("Error creating screen", err)
	}
	if err := screen.Init(); err != nil {
		logging.LogErrorAndExit("Error initializing screen", err)
	}
	defer screen.Fini()

	a, err := app.New(cfg, cfgPath, screen, reload, errs)
	if err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "lds: %v\n", err)
		os.Exit(1)
	}
	if err := a.Run(); err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "lds: %v\n", err)
		os.Exit(1)
	}
}
