//go:build !darwin

package platform

// SetActivationPolicy is a no-op on non-macOS platforms
func SetActivationPolicy() {
	// No-op on non-macOS platforms
}
