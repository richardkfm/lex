package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// findOhMyPosh returns where oh-my-posh is installed: on the PATH, or in
// a directory its installers use, which a login shell's PATH has where
// the app's may not, as Homebrew's on macOS. "" when Lex finds none.
func findOhMyPosh() string {
	name := "oh-my-posh"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".local", "bin"), filepath.Join(home, "bin"),
		"/opt/homebrew/bin", "/usr/local/bin", "/home/linuxbrew/.linuxbrew/bin",
		filepath.Join(home, ".linuxbrew", "bin"), filepath.Join(home, ".nix-profile", "bin"),
	}
	if runtime.GOOS == "windows" {
		dirs = []string{filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "oh-my-posh", "bin"),
			filepath.Join(home, "scoop", "shims"), filepath.Join(home, "AppData", "Local", "Microsoft", "WindowsApps")}
	}
	for _, d := range dirs {
		p := filepath.Join(d, name)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}
