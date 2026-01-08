//go:build darwin

package platform

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

int
SetActivationPolicy(void) {
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    return 0;
}
*/
import "C"
import "log"

// SetActivationPolicy sets the application activation policy (macOS only)
func SetActivationPolicy() {
	log.Println("Setting ActivationPolicy")
	C.SetActivationPolicy()
}
