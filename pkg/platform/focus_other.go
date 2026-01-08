//go:build !darwin

package platform

// IsAppActive always returns true on non-macOS platforms
func IsAppActive() bool {
	return true
}

// ActivateApp is a no-op on non-macOS platforms
func ActivateApp() {
	// No-op on non-macOS platforms
}
