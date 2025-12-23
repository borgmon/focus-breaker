//go:build !darwin

package main

func setActivationPolicy() {
	// No-op on non-macOS platforms
}
