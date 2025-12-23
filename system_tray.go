package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

func (fb *FocusBreaker) setupSystemTray() {
	fb.updateSystemTrayMenu()
}

func (fb *FocusBreaker) updateSystemTrayMenu() {
	if desk, ok := fb.app.(desktop.App); ok {
		menuItems := []*fyne.MenuItem{
			fyne.NewMenuItem("Settings", func() {
				fb.showConfigWindow()
			}),
			fyne.NewMenuItem("Sync Now", func() {
				go fb.syncEvents()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Quit", func() {
				fb.quit()
			}),
		}

		menu := fyne.NewMenu("Focus Breaker", menuItems...)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconTransparentPng)
	}
}
