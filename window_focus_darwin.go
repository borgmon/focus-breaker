//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework AppKit
#import <Cocoa/Cocoa.h>
#import <AppKit/AppKit.h>

int isAppActive() {
    return [NSApp isActive] ? 1 : 0;
}

void activateApp() {
    [NSApp activateIgnoringOtherApps:YES];
}
*/
import "C"

// IsAppActive returns true if the application is currently active/focused
func isAppActive() bool {
	return C.isAppActive() == 1
}

// ActivateApp brings the application to the front
func activateApp() {
	C.activateApp()
}
