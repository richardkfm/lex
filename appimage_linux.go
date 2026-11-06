package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// appImageMarker marks the desktop entry that registerAppImage wrote, which
// it may rewrite and remove, from one that a package or install.sh did.
const appImageMarker = "X-Lex-AppImage=true"

// registerAppImage gives the window of an AppImage its icon in the taskbar.
// The window's app id is lex (setProgramName), and KWin, as GNOME Shell,
// takes the icon of a Wayland window from the desktop entry of that name,
// lex.desktop, or draws its placeholder, a W on yellow. Gear Lever and
// AppImageLauncher integrate the AppImage under other names, as
// gearlever_lex_….desktop, so run as an AppImage, Lex registers a hidden
// lex.desktop of its own, which runs the AppImage and shows its icon, unless
// one from a package or install.sh is there.
func registerAppImage() {
	appImage := os.Getenv("APPIMAGE") // which the AppImage's runtime sets
	if appImage == "" {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if !filepath.IsAbs(dataHome) {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	dataDirs := filepath.SplitList(os.Getenv("XDG_DATA_DIRS"))
	if len(dataDirs) == 0 {
		dataDirs = []string{"/usr/local/share", "/usr/share"}
	}
	icon := filepath.Join(filepath.Dir(exe), "lex.png")
	if err := writeAppImageEntry(dataHome, dataDirs, appImage, icon); err != nil {
		log.Printf("lex: registering the AppImage's desktop entry: %v", err)
	}
}

// writeAppImageEntry writes dataHome's applications/lex.desktop, running
// appImage, and its icon, a copy of iconSrc, unless another lex.desktop is
// in dataHome or dataDirs, in which case it removes its own from before.
// It writes nothing that is already as it would write it.
func writeAppImageEntry(dataHome string, dataDirs []string, appImage, iconSrc string) error {
	if !filepath.IsAbs(appImage) || strings.ContainsAny(appImage, "\n\r") {
		return errors.New("the AppImage's path cannot go in a desktop entry: " + appImage)
	}
	entry := filepath.Join(dataHome, "applications", "lex.desktop")
	ours := false
	if data, err := os.ReadFile(entry); err == nil {
		if !bytes.Contains(data, []byte("\n"+appImageMarker+"\n")) {
			return nil // install.sh's, which has the icon
		}
		ours = true
	}
	for _, dir := range dataDirs {
		if dir == "" || dir == dataHome {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "applications", "lex.desktop")); err == nil {
			// A package's, which ours would hide.
			if ours {
				return os.Remove(entry)
			}
			return nil
		}
	}

	png, err := os.ReadFile(iconSrc)
	if err != nil {
		return err
	}
	icon := filepath.Join(dataHome, "icons", "lex.png")
	if err := writeIfChanged(icon, png); err != nil {
		return err
	}
	return writeIfChanged(entry, []byte(appImageEntry(appImage, icon)))
}

// appImageEntry renders the desktop entry of the AppImage at appImage,
// with its icon at icon: hidden, as the entry that Gear Lever or
// AppImageLauncher made is the one in the applications menu, and gone once
// the AppImage is (TryExec).
func appImageEntry(appImage, icon string) string {
	return "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=Lex\n" +
		"Comment=A terminal of tabs, split panes and sessions that outlive the app\n" +
		"Exec=" + desktopExec(appImage) + "\n" +
		"TryExec=" + desktopString(appImage) + "\n" +
		"Icon=" + desktopString(icon) + "\n" +
		"Categories=System;TerminalEmulator;\n" +
		"Terminal=false\n" +
		"NoDisplay=true\n" +
		"StartupWMClass=lex\n" +
		appImageMarker + "\n"
}

// desktopExec quotes path as the program of an Exec key, in double quotes
// whose ", `, $ and \ take a backslash, and then as a string, whose
// backslashes double, with % doubled as Exec's field codes want.
func desktopExec(path string) string {
	q := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", `$`, `\$`).Replace(path)
	return strings.ReplaceAll(desktopString(`"`+q+`"`), "%", "%%")
}

// desktopString escapes s as a string value of a desktop entry.
func desktopString(s string) string {
	return strings.ReplaceAll(s, `\`, `\\`)
}

// writeIfChanged writes data to path, and its directory, unless path holds
// data already, through a file it renames, so that the desktop never reads
// half of one.
func writeIfChanged(path string, data []byte) error {
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, data) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
