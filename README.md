# Lex

A terminal of tabs, split panes and sessions that outlive the app, after
[Superlogical's Rex](https://www.superlogical.com/updates/public-testing-beginning)
and forked from [GoRex](https://github.com/egoist/gorex) by EGOIST. Written in
Go with [MyGo](https://mygo.egoist.dev)'s native UI and its terminal plugin
(Ghostty's libghostty-vt). No webview, no cgo: one ~15 MB app, for Windows,
Linux and macOS.

![Lex](docs/screenshot.png)

## What it does

- **Tabs and split panes.** Every pane is a flat card in a hairline rim, the
  focused one in a brighter rim, with its program's icon, name and directory
  in its header, and buttons to split right, split down, zoom and close.
  Drag the gaps between panes to resize them (double-click resets a split
  to half).
- **Persistent sessions.** Shells run in a session server, a background
  process of the same binary (`Lex -server`). Quit Lex and everything
  keeps running; open it again and every tab, split and pane comes back as
  it was, programs still running. The server keeps each session's screen in
  a headless libghostty-vt emulator and sends a snapshot on attach, so
  full-screen programs (lazygit, vim, htop…) come back exactly.
- **Program activity.** The server watches each terminal's foreground
  process (`tcgetpgrp`, `/proc`, `sysctl`, `proc_pidinfo`; on Windows, the
  shell's newest child, its command line and directory): headers and tabs
  name the program (`Node`, `Git Changes` for lazygit, `Codex`, `SSH host`…)
  and its working directory, a green dot shows a program printing, and an
  orange dot marks a pane whose program finished or rang the bell out of
  sight. When the window is in the background, a long command finishing
  posts a notification.
- **Tab icons** stack the tiles of a tab's programs, the focused pane's in
  front, as Rex does: little rimmed cards in their programs' colors, the
  others peeking out behind. Tabs reorder by dragging, rename by double-click, and
  have a context menu.
- **Host chip** with the machine's name and model; its popover shows the
  chip, memory, OS and the session server.
- **Command palette** (Ctrl+Shift+P, or the ⚡ button): every command, and
  every pane to jump to, fuzzy-matched.
- **The aurora look** of [patina](https://github.com/patina-os/patina):
  true black with neon green as the accent and violet as the second color,
  in the window, the panes and the terminals' sixteen colors; KDE's thin
  Breeze icons on the buttons and
  [Papirus](https://github.com/PapirusDevelopmentTeam/papirus-icon-theme)'s
  glyphs for programs. Dark by default; light (lavender white) or the
  system's.
  JetBrains Mono embedded.
- **Settings** behind the gear of the title bar (Ctrl+Shift+, or
  View ▸ Settings…): appearance, text size (Ctrl+= Ctrl+- Ctrl+0), line spacing, the
  cursor's shape and blinking, ligatures, and the notification of a long
  command finishing in the background. They show in every pane at once
  and are kept in `settings.json`, in the data directory.

## Install

Every release has a Windows installer and Linux packages for x86-64
(amd64). Download them from the release page, or with the
[GitHub CLI](https://cli.github.com):

```sh
gh release download --repo richardkfm/lex --pattern '*'
```

### Windows

Windows 10 1809 or later. Run `Lex-Setup-<version>-windows-amd64.exe`: it
installs Lex for the current user in `%LOCALAPPDATA%\Programs\Lex`, without
administrator rights, with a Start menu shortcut and an uninstaller in
Settings. The installer isn't signed, so SmartScreen may warn about it:
choose *More info → Run anyway*.

Before you install a new version, run **Quit and End All Sessions** from the
command palette (Ctrl+Shift+P). The session server keeps `Lex.exe` in use,
and the installer can't replace it while the server runs.

### Linux

x86-64 with glibc 2.28 or later (Debian 10, Ubuntu 20.04 and later), GTK 3
and OpenGL. On Debian and Ubuntu, install the package, which puts Lex in
`/opt/lex` and in the applications menu:

```sh
sudo apt install ./lex_<version>_amd64.deb
```

On other distributions, or without root, run the install script as your
user. It installs the archive in `~/.local/lex.app` with its desktop entry
and icon:

```sh
sh install.sh lex-<version>-linux-amd64.tar.gz
```

`sh install.sh --uninstall` removes Lex again, and keeps its settings and
sessions. `SHA256SUMS` lists the checksums of the files. Neither method adds
a `lex` command, because `/usr/bin/lex` belongs to flex's `lex`. Start Lex
from the applications menu, or run `/opt/lex/lex`.

## Shortcuts

| Keys | |
|---|---|
| Ctrl+Shift+T | New tab |
| Ctrl+Shift+D / Ctrl+Shift+E | Split right / down |
| Ctrl+Shift+W | Close the pane |
| Ctrl+Shift+Enter | Zoom the pane |
| Alt+arrows | Focus the pane in that direction |
| Alt+Shift+arrows | Move the nearest divider |
| Ctrl+Shift+1…9, Ctrl+PageUp/PageDown, Ctrl+Tab | Switch tabs |
| Ctrl+Shift+R | Rename tab |
| Ctrl+Shift+P | Command palette |
| Ctrl+Shift+K | Clear |
| Ctrl+Shift+, | Settings |
| Ctrl+= Ctrl+- Ctrl+0 | Text size |

Close Tab, Equalize Panes and Quit and End All Sessions are in the command
palette. Control and a letter are the terminal's (Ctrl+C, Ctrl+D,
Ctrl+W…), so Lex's shortcuts take Shift too, as copying and pasting do
(Ctrl+Shift+C, Ctrl+Shift+V), and none take Control and Alt, which type
characters (AltGr) on many keyboards.

On macOS the shortcuts take ⌘ instead (⌘T, ⌘D, ⇧⌘P…), and the menus
list them all.

## Develop and build

Lex needs [Go](https://go.dev/dl/) 1.27+ and MyGo 0.3.0, whose CLI
`go.mod` pins as a tool:

```sh
go tool mygo dev     # Lex Dev, rebuilt and restarted as the code changes
go test ./...        # the server, and the view without a window
go tool mygo build   # build/<os>-<arch>/: the app and its packages
go run ./tools/mkicon  # redraw resources/icon.png
```

`go tool mygo build -platform windows/amd64,linux/amd64` builds the Windows
installer and the Linux packages on any machine, without cgo. The Windows
installer needs [NSIS](https://nsis.sourceforge.io) (`apt install nsis`,
`brew install makensis`) off Windows.

`go get -tool github.com/egoist/mygo/cmd/mygo@latest` updates MyGo and its
CLI together.

The app keeps its state in its data directory: the server's socket and
log, `layout.json` and `settings.json`. That is `~/Library/Application
Support/Lex` on macOS, `%AppData%\Lex` on Windows and `~/.config/Lex` on
Linux, and `Lex Dev` for `mygo dev`'s, so that developing never touches the
sessions of the app you use. `LEX_DIR` names another directory, and so does
GoRex's `GOREX_DIR`. The sessions outlive the app, but not a rebuild: a
development build replaces a server that an older build started, ending
its sessions, which then start again in the same directories.

### Windows

Lex runs on Windows 10 1809 or later, whose ConPTY runs the shells. Panes
run `$SHELL` when it names a Windows program (`nu`,
`C:\Program Files\Git\bin\bash.exe`), else PowerShell (`pwsh`, then
`powershell`), else `cmd`. PowerShell's `cd` leaves its process's directory
where it was, so Lex starts it with a prompt that sets it, wrapping the
profile's own prompt: the header then follows it, and new panes open there.
The window is frameless and its title bar draws minimize, maximize and close
with the other buttons; it resizes from its sides and bottom, not its top.

### Linux

Lex runs on GTK 3 and draws with OpenGL (or on the CPU without a GPU). Panes
run your login shell, `$SHELL`, else `/bin/sh`. As on Windows, the window
is frameless and its title bar draws minimize, maximize and close on its
right, beside the other buttons. A Wayland compositor that decorates windows
(KWin, Sway…) is told not to. The menus (Shell, Edit, View, Tabs, Window)
sit above the title bar. The host popover reads the model
from the firmware (`/sys/class/dmi/id`, or the device tree), and the
processor and memory from `/proc`.

### Releasing

Bump `version` in `mygo.json`, commit, and push its tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

[`.github/workflows/release.yml`](.github/workflows/release.yml) checks that
the tag is `mygo.json`'s version, tests Lex on Linux and Windows, builds the
Windows installer and the Linux packages on Ubuntu, and uploads them with
`SHA256SUMS` to a draft release. Review the draft and publish it. The same
build runs for every pull request, and its packages are kept as an artifact
of the run ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)).

## Layout

| | |
|---|---|
| `main.go` | the app, its window, and `-server` |
| `state.go` | tabs, the tree of splits, panes, saving and restoring the layout |
| `view.go`, `tabs.go`, `host.go`, `commands.go`, `settingsview.go` | the interface: title bar, tabs, panes, host popover, menus, palette and settings panel |
| `programs.go` | how programs show: names and glyphs |
| `style.go`, `settings.go` | colors, fonts, terminal themes; the settings and how they apply |
| `internal/rex` | the session server and its client: PTYs, foreground processes, headless screens, the socket protocol |

Not replicated: Rex's connections to servers on other machines. Lex's
server listens on a local socket only.

## Credits

- **[Superlogical](https://www.superlogical.com)** made
  [Rex](https://www.superlogical.com/updates/public-testing-beginning), the
  terminal whose design and behavior Lex follows. Lex is not affiliated with
  or endorsed by Superlogical. Rex, its name and its design are theirs.
- **[GoRex](https://github.com/egoist/gorex)** by
  [EGOIST](https://github.com/egoist) is the Go replica of Rex that Lex is
  forked from. The session server, the interface and the Windows port are
  EGOIST's work.
- [MyGo](https://mygo.egoist.dev) (MIT), by EGOIST and its contributors, is
  the native UI, the terminal plugin and the build tool.
- [libghostty-vt](https://github.com/ghostty-org/ghostty) (MIT), from
  Ghostty, emulates the terminals.
- [purego](https://github.com/ebitengine/purego) (Apache-2.0),
  [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys) and
  [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause),
  and [go-text/typesetting](https://github.com/go-text/typesetting)
  (BSD-3-Clause or Unlicense).
- [patina](https://github.com/patina-os/patina)'s aurora theme
  (GPL-3.0-or-later), by the patina contributors, gives Lex its colors.
- Icons: KDE's [Breeze icons](https://invent.kde.org/frameworks/breeze-icons)
  (LGPL-3.0-or-later), the
  [Papirus icon theme](https://github.com/PapirusDevelopmentTeam/papirus-icon-theme)
  (GPL-3.0; `assets/icons/README.md` lists which is which) and
  [Simple Icons](https://simpleicons.org) (CC0). Font:
  [JetBrains Mono](https://www.jetbrains.com/lp/mono/) (OFL).
