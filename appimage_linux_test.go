package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopExec(t *testing.T) {
	for _, tt := range []struct{ path, want string }{
		{"/home/a/AppImages/lex.appimage", `"/home/a/AppImages/lex.appimage"`},
		{"/home/a/My Apps/Lex.AppImage", `"/home/a/My Apps/Lex.AppImage"`},
		{`/x/"q"/$v/a\b/100%`, `"/x/\\"q\\"/\\$v/a\\\\b/100%%"`},
		{"/x/`b`", "\"/x/\\\\`b\\\\`\""},
	} {
		if got := desktopExec(tt.path); got != tt.want {
			t.Errorf("desktopExec(%q) = %s, want %s", tt.path, got, tt.want)
		}
	}
}

// appImageDirs makes a data home and a system data directory, and the
// AppImage's icon.
func appImageDirs(t *testing.T) (home, system, icon string) {
	t.Helper()
	dir := t.TempDir()
	home, system = filepath.Join(dir, "home"), filepath.Join(dir, "usr")
	icon = filepath.Join(dir, "mount", "lex.png")
	if err := os.MkdirAll(filepath.Dir(icon), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icon, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	return home, system, icon
}

func TestWriteAppImageEntry(t *testing.T) {
	home, system, icon := appImageDirs(t)
	entry := filepath.Join(home, "applications", "lex.desktop")
	if err := writeAppImageEntry(home, []string{system}, "/a/lex.appimage", icon); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(entry)
	if err != nil {
		t.Fatal(err)
	}
	copied := filepath.Join(home, "icons", "lex.png")
	for _, line := range []string{`Exec="/a/lex.appimage"`, "TryExec=/a/lex.appimage", "Icon=" + copied, "NoDisplay=true", "StartupWMClass=lex", appImageMarker} {
		if !strings.Contains(string(data), "\n"+line+"\n") {
			t.Errorf("the entry has no line %s:\n%s", line, data)
		}
	}
	if png, err := os.ReadFile(copied); err != nil || string(png) != "png" {
		t.Errorf("the icon is %q, %v", png, err)
	}

	// Moved, the AppImage runs from its new place.
	if err := writeAppImageEntry(home, []string{system}, "/b/lex.appimage", icon); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(entry); !strings.Contains(string(data), `Exec="/b/lex.appimage"`) {
		t.Errorf("the entry runs the old AppImage:\n%s", data)
	}

	// A package's entry comes in: ours goes, as it would hide it.
	pkg := filepath.Join(system, "applications", "lex.desktop")
	if err := os.MkdirAll(filepath.Dir(pkg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkg, []byte("[Desktop Entry]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAppImageEntry(home, []string{system}, "/b/lex.appimage", icon); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(entry); !os.IsNotExist(err) {
		t.Errorf("our entry is still there next to the package's: %v", err)
	}
}

func TestWriteAppImageEntryKeepsInstallScripts(t *testing.T) {
	home, system, icon := appImageDirs(t)
	entry := filepath.Join(home, "applications", "lex.desktop")
	installed := "[Desktop Entry]\nExec=\"/home/a/.local/lex.app/lex\"\n"
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte(installed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAppImageEntry(home, []string{system}, "/a/lex.appimage", icon); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(entry); string(data) != installed {
		t.Errorf("install.sh's entry became:\n%s", data)
	}
}
