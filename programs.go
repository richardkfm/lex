package main

import (
	"embed"
	"path"
	"strings"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/icons/*.svg assets/brands/*.svg
var assets embed.FS

var svgs = map[string]*ui.SVG{}

// icon returns an embedded SVG: "x" from the icons, "brand:git" from the
// brands.
func icon(name string) *ui.SVG {
	if s, ok := svgs[name]; ok {
		return s
	}
	file := "assets/icons/" + name + ".svg"
	if b, ok := strings.CutPrefix(name, "brand:"); ok {
		file = "assets/brands/" + b + ".svg"
	}
	data, err := assets.ReadFile(file)
	if err != nil {
		panic(err)
	}
	s := ui.MustParseSVG(data)
	svgs[name] = s
	return s
}

// program is how a program shows: its name, its glyph in headers, and the
// glyph on its tile in tabs. Tiles are one color, the theme's, whatever the
// program: logos keep their shapes, not their brands' colors.
type program struct {
	Name string
	// Glyph is the icon of the pane's header, in the color of the text.
	Glyph string
	// TileGlyph is the glyph on the program's tile in tabs, Glyph when
	// empty.
	TileGlyph string
	// Shell is a shell at its prompt.
	Shell bool
}

var (
	shellTile = program{Glyph: "utilities-terminal", TileGlyph: "terminal-prompt", Shell: true}

	programs = map[string]program{
		"codex":     {Name: "Codex", Glyph: "brand:openai"},
		"claude":    {Name: "Claude Code", Glyph: "brand:claude"},
		"node":      {Name: "Node", Glyph: "node-hex", TileGlyph: "brand:nodedotjs"},
		"bun":       {Name: "Bun", Glyph: "brand:bun"},
		"deno":      {Name: "Deno", Glyph: "brand:deno"},
		"python":    {Name: "Python", Glyph: "brand:python"},
		"lazygit":   {Name: "Git Changes", Glyph: "brand:git"},
		"tig":       {Name: "Git Log", Glyph: "vcs-branch"},
		"git":       {Name: "Git", Glyph: "brand:git"},
		"vim":       {Name: "Vim", Glyph: "brand:vim"},
		"nvim":      {Name: "Neovim", Glyph: "brand:neovim"},
		"hx":        {Name: "Helix", Glyph: "brand:helix"},
		"emacs":     {Name: "Emacs", Glyph: "brand:gnuemacs"},
		"htop":      {Name: "Activity", Glyph: "utilities-system-monitor", TileGlyph: "activity-pulse"},
		"btop":      {Name: "Activity", Glyph: "utilities-system-monitor", TileGlyph: "activity-pulse"},
		"top":       {Name: "Activity", Glyph: "utilities-system-monitor", TileGlyph: "activity-pulse"},
		"ssh":       {Name: "SSH", Glyph: "network-workgroup"},
		"mosh":      {Name: "Mosh", Glyph: "network-workgroup"},
		"docker":    {Name: "Docker", Glyph: "brand:docker"},
		"go":        {Name: "Go", Glyph: "brand:go"},
		"cargo":     {Name: "Cargo", Glyph: "brand:rust"},
		"rustc":     {Name: "Rust", Glyph: "brand:rust"},
		"npm":       {Name: "npm", Glyph: "brand:npm"},
		"pnpm":      {Name: "pnpm", Glyph: "brand:pnpm"},
		"yarn":      {Name: "Yarn", Glyph: "brand:yarn"},
		"ruby":      {Name: "Ruby", Glyph: "brand:ruby"},
		"irb":       {Name: "Ruby", Glyph: "brand:ruby"},
		"lua":       {Name: "Lua", Glyph: "brand:lua"},
		"php":       {Name: "PHP", Glyph: "brand:php"},
		"psql":      {Name: "PostgreSQL", Glyph: "brand:postgresql"},
		"mysql":     {Name: "MySQL", Glyph: "brand:mysql"},
		"redis-cli": {Name: "Redis", Glyph: "brand:redis"},
		"tmux":      {Name: "tmux", Glyph: "brand:tmux"},
		"make":      {Name: "Make", Glyph: "run-build"},
		"swift":     {Name: "Swift", Glyph: "brand:swift"},
		"kotlin":    {Name: "Kotlin", Glyph: "brand:kotlin"},
	}

	shells = map[string]bool{"zsh": true, "bash": true, "fish": true, "sh": true, "dash": true, "nu": true, "pwsh": true, "elvish": true, "xonsh": true, "tcsh": true, "csh": true, "ksh": true,
		"cmd": true, "powershell": true}
)

func init() {
	programs["python3"] = programs["python"]
	programs["ipython"] = programs["python"]
	programs["view"] = programs["vim"]
	programs["vi"] = programs["vim"]
	programs["less"] = program{Name: "Pager", Glyph: "text-x-generic"}
	programs["man"] = programs["less"]
}

// programOf returns how a program shows, by the name of its process.
func programOf(name string) program {
	name = strings.TrimPrefix(path.Base(name), "-")
	if p, ok := programs[strings.ToLower(name)]; ok {
		return p
	}
	if shells[name] {
		p := shellTile
		p.Name = name
		return p
	}
	// Any other program: its own name, with a terminal's glyph.
	return program{Name: name, Glyph: "utilities-terminal", TileGlyph: "terminal-prompt"}
}
