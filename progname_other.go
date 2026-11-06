//go:build !linux

package main

// setProgramName names the app to the desktop on Linux only.
func setProgramName() {}

// registerAppImage registers the desktop entry of an AppImage, on Linux
// only.
func registerAppImage() {}
