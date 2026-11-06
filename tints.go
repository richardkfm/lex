package main

import (
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"

	"lex/internal/rex"
)

// Pane tints: a pane takes a color, from the theme, while what runs in it
// is of a kind to keep in mind, as a remote or a root shell, and loses it
// once that program ends. The settings' Panes section turns them on and
// off, all or by kind, and says how strongly they show.

// tintKinds are the kinds of panes that take a tint: chip is the word
// their header shows.
var tintKinds = []struct{ id, label, hint, chip string }{
	{"remote", "Remote sessions", "ssh, mosh, et, telnet", "Remote"},
	{"root", "Root and sudo", "sudo, su, doas, run0", "Root"},
	{"container", "Containers", "docker or podman exec, run and attach, kubectl exec", "Container"},
	{"agent", "AI agents", "Claude Code, Codex", "Agent"},
}

// tintStrengths are how strongly tints show: in the rim, in the rim and
// the header with its chip, or in those and a wash over the whole pane.
var tintStrengths = []struct{ id, label, hint string }{
	{"rim", "Rim", "Only the rim takes the tint: the quietest."},
	{"header", "Header", "The rim and the header take the tint, with a word naming the kind of pane."},
	{"wash", "Wash", "As Header, and the whole pane takes a faint wash of the color."},
}

// paneKind returns the kind of a pane by the program in its foreground,
// "" for none: a shell at its prompt has none.
func paneKind(in rex.SessionInfo) string {
	if in.Idle || in.Program == "" {
		return ""
	}
	name := strings.ToLower(strings.TrimPrefix(path.Base(strings.ReplaceAll(in.Program, `\`, "/")), "-"))
	name = strings.TrimSuffix(name, ".exe")
	words := commandWords(name, in.Args)
	switch name {
	case "ssh", "mosh", "mosh-client", "et", "telnet", "autossh":
		return "remote"
	case "sudo", "su", "doas", "run0", "pkexec":
		return "root"
	case "claude", "codex":
		return "agent"
	case "docker", "podman", "nerdctl":
		// docker exec, run and attach, as docker compose's.
		if len(words) > 1 && words[0] == "compose" {
			words = words[1:]
		}
		if len(words) > 0 && slices.Contains([]string{"exec", "run", "attach"}, words[0]) {
			return "container"
		}
	case "kubectl", "oc":
		if slices.ContainsFunc(words, func(w string) bool { return w == "exec" || w == "attach" || w == "debug" }) {
			return "container"
		}
	}
	return ""
}

// commandWords returns a command's arguments after its name that are not
// options, as its subcommands. The name is the first argument, or for a
// script, the one naming it after its interpreter's.
func commandWords(name string, args []string) []string {
	start := 1
	for i, a := range args {
		if base := strings.TrimSuffix(strings.ToLower(path.Base(strings.ReplaceAll(a, `\`, "/"))), ".exe"); base == name {
			start = i + 1
			break
		}
	}
	var words []string
	for _, a := range args[min(start, len(args)):] {
		if !strings.HasPrefix(a, "-") {
			words = append(words, a)
		}
	}
	return words
}

// tintOn tells whether the settings tint the kind of pane.
func (s settings) tintOn(kind string) bool {
	if s.NoPaneTints {
		return false
	}
	switch kind {
	case "remote":
		return !s.NoTintRemote
	case "root":
		return !s.NoTintRoot
	case "container":
		return !s.NoTintContainer
	case "agent":
		return s.TintAgents
	}
	return false
}

// toggleTint turns the tint of a kind of pane on or off.
func (s *settings) toggleTint(kind string) {
	switch kind {
	case "remote":
		s.NoTintRemote = !s.NoTintRemote
	case "root":
		s.NoTintRoot = !s.NoTintRoot
	case "container":
		s.NoTintContainer = !s.NoTintContainer
	case "agent":
		s.TintAgents = !s.TintAgents
	}
}

// paneTintStrength is how strongly the settings show tints.
func (s settings) paneTintStrength() string {
	switch s.PaneTint {
	case "rim", "wash":
		return s.PaneTint
	}
	return "header"
}

// tintColor is a kind's color in the theme: remote sessions take its
// danger, root its attention, containers its terminals' blue and agents
// their magenta, which differ in every theme.
func tintColor(k *colors, kind string) ui.Color {
	switch kind {
	case "remote":
		return k.danger
	case "root":
		return k.attention
	case "container":
		return k.blue
	}
	return k.magenta
}

// tint is how a pane shows its kind; zero for a pane of none.
type tint struct {
	on   bool
	col  ui.Color
	chip string
	// header tells that the header takes the tint, and wash the pane.
	header, wash bool
}

// paneTint returns the tint of a pane, as the settings ask.
func paneTint(k *colors, in rex.SessionInfo) tint {
	kind := paneKind(in)
	if kind == "" || !prefs.tintOn(kind) {
		return tint{}
	}
	t := tint{on: true, col: tintColor(k, kind)}
	for _, tk := range tintKinds {
		if tk.id == kind {
			t.chip = tk.chip
		}
	}
	s := prefs.paneTintStrength()
	t.header, t.wash = s != "rim", s == "wash"
	return t
}

// mixColor mixes b into a by t, from 0 (a) to 1 (b).
func mixColor(a, b ui.Color, t float32) ui.Color {
	m := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t + 0.5) }
	return ui.Color{R: m(a.R, b.R), G: m(a.G, b.G), B: m(a.B, b.B), A: m(a.A, b.A)}
}
