package rex

import (
	"bytes"
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// The prompt Lex styles, when a window asks for it (CreateOptions.Prompt):
// Oh My Posh, installed by the user, in a theme of Lex's (lex.omp.json),
// whose colors are the terminal's sixteen, so that it takes the theme of
// the window and changes with it. The user's shell loads its own
// configuration first, and then Oh My Posh, from files Lex writes in its
// directory: bash, zsh and fish, and PowerShell on Windows. Other shells,
// and a shell without oh-my-posh on its PATH, keep their own prompts, as
// does every file of the user's, which Lex reads and never changes.

// PromptOhMyPosh asks for Oh My Posh in Lex's theme.
const PromptOhMyPosh = "omp"

//go:embed all:prompt
var promptFiles embed.FS

// promptDir returns the directory of the files that start shells with
// the prompt, written anew where they changed.
func promptDir() (string, error) {
	dir := filepath.Join(Dir(), "prompt")
	err := fs.WalkDir(promptFiles, "prompt", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := promptFiles.ReadFile(path)
		if err != nil {
			return err
		}
		to := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(path, "prompt/")))
		if old, err := os.ReadFile(to); err == nil && bytes.Equal(old, b) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return err
		}
		return os.WriteFile(to, b, 0o600)
	})
	return dir, err
}

// ohMyPoshInit are the commands that load Oh My Posh, in fish and in
// PowerShell, when it is installed.
const (
	fishOhMyPosh = `if type -q oh-my-posh; oh-my-posh init fish --config "$LEX_OMP_CONFIG" | source; end`
	pwshOhMyPosh = `if (Get-Command oh-my-posh -ErrorAction SilentlyContinue) { ` +
		`oh-my-posh init pwsh --config "$env:LEX_OMP_CONFIG" | Invoke-Expression }; `
)

// withPrompt returns the arguments and the environment that start the
// user's shell, named name, as argv starts it, with the prompt asked for:
// argv and no more for a command of the window's own, a prompt Lex does
// not know or a shell it cannot style.
func withPrompt(prompt string, cmd []string, name string, argv []string) ([]string, []string) {
	if prompt != PromptOhMyPosh || len(cmd) > 0 {
		return argv, nil
	}
	shell := strings.ToLower(name)
	switch shell {
	case "bash", "zsh", "fish", "pwsh", "powershell":
	default:
		return argv, nil
	}
	dir, err := promptDir()
	if err != nil {
		log.Printf("rex: writing the prompt's files: %v", err)
		return argv, nil
	}
	env := []string{"LEX_OMP_CONFIG=" + filepath.Join(dir, "lex.omp.json")}
	switch shell {
	case "zsh":
		// zsh reads its files from ZDOTDIR: Lex's, which load the user's.
		if z, ok := os.LookupEnv("ZDOTDIR"); ok {
			env = append(env, "LEX_USER_ZDOTDIR="+z)
		}
		env = append(env, "ZDOTDIR="+filepath.Join(dir, "zsh"))
		return argv, env
	case "bash":
		// A login shell reads no file of the window's: Lex's, in place
		// of .bashrc, reads those a login shell would.
		return []string{"bash", "--init-file", filepath.Join(dir, "bashrc")}, env
	case "fish":
		// -C runs after the configuration.
		return append(argv[:len(argv):len(argv)], "-C", fishOhMyPosh), env
	default:
		// PowerShell: Oh My Posh's prompt before Lex's own wraps it.
		out := append([]string(nil), argv...)
		for i := range out[:len(out)-1] {
			if strings.EqualFold(out[i], "-Command") {
				out[i+1] = pwshOhMyPosh + out[i+1]
				return out, env
			}
		}
		return append(out, "-NoExit", "-Command", pwshOhMyPosh), env
	}
}
