package main

import (
	"os"
	"path/filepath"

	"github.com/ebitengine/purego"
)

// setProgramName names the app to the desktop after its executable, "lex",
// before GTK starts. GTK takes the window's WM_CLASS and Wayland app id
// from the program name, which is argv[0]'s otherwise: an AppImage's own
// file name (Lex-0.2.1-x86_64.AppImage, or lex.appimage under Gear Lever),
// which matches no desktop entry's StartupWMClass, so that KDE shows the
// window with a placeholder icon.
func setProgramName() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	lib, err := purego.Dlopen("libglib-2.0.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return // MyGo says why, as it loads GLib too
	}
	sym, err := purego.Dlsym(lib, "g_set_prgname")
	if err != nil {
		return
	}
	var setPrgname func(name string)
	purego.RegisterFunc(&setPrgname, sym)
	setPrgname(filepath.Base(exe)) // which GLib copies
}
