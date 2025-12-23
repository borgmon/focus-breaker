package main

import (
	"fmt"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type AlertWindow struct {
	window        fyne.Window
	app           fyne.App
	event         Event
	snoozeMinutes int
	onClose       func()
	onSnooze      func()

	closeProgress  float64
	snoozeProgress float64
	closeTicker    *time.Ticker
	snoozeTicker   *time.Ticker
	closeHeld      bool
	snoozeHeld     bool
	audioPlayer    *AudioPlayer
}

func NewAlertWindow(app fyne.App, event Event, snoozeMinutes int, onClose, onSnooze func()) *AlertWindow {
	aw := &AlertWindow{
		app:           app,
		event:         event,
		snoozeMinutes: snoozeMinutes,
		onClose:       onClose,
		onSnooze:      onSnooze,
	}

	// Play alarm sound
	aw.audioPlayer = playAlarmSound()

	// Create window and build UI on the main Fyne thread
	fyne.Do(func() {
		aw.window = app.NewWindow("Meeting Alert")
		aw.window.SetFullScreen(true)
		aw.buildUI()

		// Stop sound when window is closed
		aw.window.SetOnClosed(func() {
			if aw.audioPlayer != nil {
				aw.audioPlayer.Stop()
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
	closeButton = NewHoldButton("Close (Hold 5s)", func() {
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
		snoozeButton = NewHoldButton(fmt.Sprintf("Snooze %dm (Hold 5s)", aw.snoozeMinutes), func() {
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

	aw.closeTicker = time.NewTicker(50 * time.Millisecond)

	go func() {
		for range aw.closeTicker.C {
			if !aw.closeHeld {
				return
			}

			aw.closeProgress += 0.01
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

	aw.snoozeTicker = time.NewTicker(50 * time.Millisecond)

	go func() {
		for range aw.snoozeTicker.C {
			if !aw.snoozeHeld {
				return
			}

			aw.snoozeProgress += 0.01
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
