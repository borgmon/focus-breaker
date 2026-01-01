package main

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"golang.design/x/hotkey"
)

type AlertWindow struct {
	window          fyne.Window
	app             fyne.App
	event           Event
	snoozeMinutes   int
	holdTimeSeconds int
	onClose         func()
	onSnooze        func()

	closeProgress  float64
	snoozeProgress float64
	closeTicker    *time.Ticker
	snoozeTicker   *time.Ticker
	closeHeld      bool
	snoozeHeld     bool
	audioPlayer    *AudioPlayer
	cmdQHotkey     *hotkey.Hotkey
	stopMonitoring chan struct{}
}

func NewAlertWindow(app fyne.App, event Event, snoozeMinutes int, holdTimeSeconds int, onClose, onSnooze func()) *AlertWindow {
	aw := &AlertWindow{
		app:             app,
		event:           event,
		snoozeMinutes:   snoozeMinutes,
		holdTimeSeconds: holdTimeSeconds,
		onClose:         onClose,
		onSnooze:        onSnooze,
		stopMonitoring:  make(chan struct{}),
	}

	// Play alarm sound
	aw.audioPlayer = playAlarmSound()

	// Create window and build UI on the main Fyne thread
	fyne.Do(func() {
		aw.window = app.NewWindow("Meeting Alert")
		aw.window.SetFullScreen(true)
		aw.buildUI()

		// Register Cmd+Q hotkey when window is focused
		aw.registerCmdQPrevention()

		// Monitor window focus and refocus when needed
		aw.setupFocusMonitoring()

		// Stop sound when window is closed
		aw.window.SetOnClosed(func() {
			// Stop monitoring first
			close(aw.stopMonitoring)

			if aw.audioPlayer != nil {
				aw.audioPlayer.Stop()
			}
			// Unregister hotkey when window is closed
			if aw.cmdQHotkey != nil {
				aw.cmdQHotkey.Unregister()
			}
		})
	})

	return aw
}

func (aw *AlertWindow) buildUI() {
	title := canvas.NewText(aw.event.Title, nil)
	title.TextSize = 32
	title.Alignment = fyne.TextAlignCenter

	timeInfo := fmt.Sprintf("Start: %s\nEnd: %s",
		aw.event.StartTime.Format("3:04 PM"),
		aw.event.EndTime.Format("3:04 PM"))
	timeLabel := widget.NewLabel(timeInfo)
	timeLabel.Alignment = fyne.TextAlignCenter

	description := widget.NewLabel(aw.event.Description)
	description.Wrapping = fyne.TextWrapWord
	description.Alignment = fyne.TextAlignCenter

	var linkButton *widget.Button
	if aw.event.MeetingLink != "" {
		linkButton = widget.NewButton("Join Meeting", func() {
			if u, err := url.Parse(aw.event.MeetingLink); err == nil {
				fyne.CurrentApp().OpenURL(u)
			}
		})
		linkButton.Importance = widget.HighImportance
	}

	var closeButton *HoldButton
	closeButton = NewHoldButton(fmt.Sprintf("Close (Hold %ds)", aw.holdTimeSeconds), func() {
		aw.startCloseProgress(closeButton)
	}, func() {
		aw.stopCloseProgress(closeButton)
	})

	content := container.NewVBox(
		container.NewPadded(title),
		timeLabel,
		widget.NewSeparator(),
		container.NewPadded(description),
	)

	if linkButton != nil {
		content.Add(container.NewCenter(linkButton))
	}

	content.Add(widget.NewSeparator())

	// Button row
	buttonRow := container.NewHBox()
	if aw.snoozeMinutes > 0 {
		var snoozeButton *HoldButton
		snoozeButton = NewHoldButton(fmt.Sprintf("Snooze %dm (Hold %ds)", aw.snoozeMinutes, aw.holdTimeSeconds), func() {
			aw.startSnoozeProgress(snoozeButton)
		}, func() {
			aw.stopSnoozeProgress(snoozeButton)
		})
		buttonRow.Add(snoozeButton)
	}
	buttonRow.Add(closeButton)

	content.Add(buttonRow)

	centered := container.NewCenter(
		container.NewVBox(
			content,
		),
	)

	aw.window.SetContent(container.NewPadded(centered))
}

func (aw *AlertWindow) startCloseProgress(button *HoldButton) {
	if aw.closeHeld {
		return
	}

	aw.closeHeld = true
	aw.closeProgress = 0
	fyne.Do(func() {
		button.SetProgress(0)
	})

	tickInterval := 50 * time.Millisecond
	totalTicks := float64(aw.holdTimeSeconds*1000) / float64(tickInterval.Milliseconds())
	progressIncrement := 1.0 / totalTicks

	aw.closeTicker = time.NewTicker(tickInterval)

	go func() {
		for range aw.closeTicker.C {
			if !aw.closeHeld {
				return
			}

			aw.closeProgress += progressIncrement
			currentProgress := aw.closeProgress

			fyne.Do(func() {
				button.SetProgress(currentProgress)
			})

			if currentProgress >= 1.0 {
				aw.closeTicker.Stop()
				if aw.onClose != nil {
					aw.onClose()
				}
				fyne.Do(func() {
					aw.window.Close()
				})
				return
			}
		}
	}()
}

func (aw *AlertWindow) stopCloseProgress(button *HoldButton) {
	aw.closeHeld = false
	if aw.closeTicker != nil {
		aw.closeTicker.Stop()
	}
	aw.closeProgress = 0
	fyne.Do(func() {
		button.SetProgress(0)
	})
}

func (aw *AlertWindow) startSnoozeProgress(button *HoldButton) {
	if aw.snoozeHeld {
		return
	}

	aw.snoozeHeld = true
	aw.snoozeProgress = 0
	fyne.Do(func() {
		button.SetProgress(0)
	})

	tickInterval := 50 * time.Millisecond
	totalTicks := float64(aw.holdTimeSeconds*1000) / float64(tickInterval.Milliseconds())
	progressIncrement := 1.0 / totalTicks

	aw.snoozeTicker = time.NewTicker(tickInterval)

	go func() {
		for range aw.snoozeTicker.C {
			if !aw.snoozeHeld {
				return
			}

			aw.snoozeProgress += progressIncrement
			currentProgress := aw.snoozeProgress

			fyne.Do(func() {
				button.SetProgress(currentProgress)
			})

			if currentProgress >= 1.0 {
				aw.snoozeTicker.Stop()
				if aw.onSnooze != nil {
					aw.onSnooze()
				}
				fyne.Do(func() {
					aw.window.Close()
				})
				return
			}
		}
	}()
}

func (aw *AlertWindow) stopSnoozeProgress(button *HoldButton) {
	aw.snoozeHeld = false
	if aw.snoozeTicker != nil {
		aw.snoozeTicker.Stop()
	}
	aw.snoozeProgress = 0
	fyne.Do(func() {
		button.SetProgress(0)
	})
}

func (aw *AlertWindow) Show() {
	fyne.Do(func() {
		if aw.window != nil {
			aw.window.Show()
		}
	})
}

func (aw *AlertWindow) registerCmdQPrevention() {
	go func() {
		// Register Cmd+Q (Cmd is ModCmd on macOS)
		hk := hotkey.New([]hotkey.Modifier{hotkey.ModCmd}, hotkey.KeyQ)
		if err := hk.Register(); err != nil {
			log.Printf("Failed to register Cmd+Q hotkey prevention: %v", err)
			return
		}
		aw.cmdQHotkey = hk

		// Start listen hotkey event whenever it is ready
		// This loop will consume Cmd+Q events and prevent default quit behavior
		for range hk.Keydown() {
			log.Println("Cmd+Q blocked - use the Close button to dismiss the alert")
		}
	}()
}

func (aw *AlertWindow) setupFocusMonitoring() {
	// Monitor if window loses focus and unregister hotkey
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		wasFocused := true
		for {
			select {
			case <-aw.stopMonitoring:
				// Window closed, stop monitoring
				log.Println("Stopping focus monitoring")
				return
			case <-ticker.C:
				if aw.window == nil {
					return
				}

				// Check if app is active using macOS native API
				isFocused := isAppActive()

				// Detect focus change
				if wasFocused && !isFocused {
					// Window lost focus - unregister hotkey
					if aw.cmdQHotkey != nil {
						log.Println("Window lost focus - unregistering Cmd+Q hotkey")
						aw.cmdQHotkey.Unregister()
						aw.cmdQHotkey = nil
					}
				} else if !wasFocused && isFocused {
					// Window gained focus - register hotkey
					if aw.cmdQHotkey == nil {
						log.Println("Window gained focus - registering Cmd+Q hotkey")
						aw.registerCmdQPrevention()
					}
				}

				// If app is not focused, bring it to front
				if !isFocused {
					log.Println("Alert window not active - bringing to front")
					activateApp()
					fyne.Do(func() {
						if aw.window != nil {
							aw.window.Show()
						}
					})
				}

				wasFocused = isFocused
			}
		}
	}()
}
