package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/emersion/go-autostart"
)

func setupAutostart(enable bool) error {
	// Get the executable path
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// Resolve symlinks if any
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return err
	}

	app := &autostart.App{
		Name:        "focus-breaker",
		DisplayName: "Focus Breaker",
		Exec:        []string{execPath},
	}

	if enable {
		if !app.IsEnabled() {
			if err := app.Enable(); err != nil {
				log.Printf("Failed to enable autostart: %v", err)
				return err
			}
			log.Println("Autostart enabled")
		}
	} else {
		if app.IsEnabled() {
			if err := app.Disable(); err != nil {
				log.Printf("Failed to disable autostart: %v", err)
				return err
			}
			log.Println("Autostart disabled")
		}
	}

	return nil
}
