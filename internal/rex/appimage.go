package rex

import (
	"os"
	"path/filepath"
	"strings"
)

// appImage returns the AppImage that runs this process, or "" when none
// does. An AppImage mounts itself in a temporary directory for as long as
// its first process runs and runs the executable from there, so the
// executable's path changes from one run to the next and is gone once the
// window closes: the session server is the AppImage run with -server, a
// mount of its own, and a build is known by the AppImage's file.
//
// The runtime sets APPIMAGE and APPDIR; they are trusted only while the
// executable is inside APPDIR, since a program started in one of Lex's
// panes would otherwise inherit them (sessionEnv leaves them out too).
func appImage() string {
	img, dir := os.Getenv("APPIMAGE"), os.Getenv("APPDIR")
	if img == "" || dir == "" || !filepath.IsAbs(img) {
		return ""
	}
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if p, err := filepath.EvalSymlinks(exe); err == nil {
		exe = p
	}
	if p, err := filepath.EvalSymlinks(dir); err == nil {
		dir = p
	}
	if !strings.HasPrefix(exe, dir+string(filepath.Separator)) {
		return ""
	}
	if st, err := os.Stat(img); err != nil || !st.Mode().IsRegular() {
		return ""
	}
	return img
}

// appImageEnv reports whether k is one of the variables an AppImage's
// runtime sets for its own processes, which a pane's shell must not
// inherit.
func appImageEnv(k string) bool {
	return k == "APPIMAGE" || k == "APPDIR" || k == "ARGV0" || k == "OWD"
}
