package rex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestAppImage(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	img := filepath.Join(t.TempDir(), "Lex.AppImage")
	if err := os.WriteFile(img, []byte("AppImage"), 0o755); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(img, at, at); err != nil {
		t.Fatal(err)
	}

	// No AppImage: the executable itself.
	t.Setenv("APPIMAGE", "")
	t.Setenv("APPDIR", "")
	if got, _ := Executable(); got != exe {
		t.Errorf("without an AppImage, Executable() = %q, want %q", got, exe)
	}

	// Run from the AppImage's mount: the AppImage, and its time.
	t.Setenv("APPIMAGE", img)
	t.Setenv("APPDIR", filepath.Dir(exe))
	if got, gotAt := Executable(); got != img || !gotAt.Equal(at) {
		t.Errorf("in an AppImage, Executable() = %q, %v, want %q, %v", got, gotAt, img, at)
	}

	// Variables inherited from an AppImage the executable is not in.
	t.Setenv("APPDIR", t.TempDir())
	if got := appImage(); got != "" {
		t.Errorf("outside APPDIR, appImage() = %q, want none", got)
	}

	// A pane's shell inherits none of them.
	t.Setenv("ARGV0", "Lex.AppImage")
	t.Setenv("OWD", "/")
	for _, kv := range sessionEnv("id", nil) {
		k, _, _ := strings.Cut(kv, "=")
		if slices.Contains([]string{"APPIMAGE", "APPDIR", "ARGV0", "OWD"}, k) {
			t.Errorf("sessionEnv has %s", kv)
		}
	}
}
