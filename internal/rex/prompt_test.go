//go:build darwin || linux

package rex

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// fakeOhMyPosh stands in for oh-my-posh: its init sets a prompt naming the
// shell, and tells whether it was given Lex's theme.
const fakeOhMyPosh = `#!/bin/sh
theme=no
[ "$3" = --config ] && [ -r "$4" ] && theme=yes
case "$2" in
zsh|bash) echo "PS1='OMP:$2:$theme> '" ;;
fish) echo "function fish_prompt; echo -n 'OMP:fish:$theme> '; end" ;;
esac
`

// TestPromptShells starts bash, zsh (with and without a ZDOTDIR of the
// user's) and fish, those installed, with Oh My Posh asked for: each loads
// the user's own files, and then Oh My Posh in Lex's theme.
func TestPromptShells(t *testing.T) {
	home, err := os.MkdirTemp("/tmp", "rexhome")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)
	bin := filepath.Join(home, "bin")
	files := map[string]string{
		"bin/oh-my-posh": fakeOhMyPosh,
		// zsh's, in the home and in a ZDOTDIR of their own.
		".zshenv":        "export USER_ENV=env\n",
		".zprofile":      "export USER_PROFILE=profile\n",
		".zshrc":         "export USER_RC=rc\nPS1='user> '\n",
		".zlogin":        "export USER_LOGIN=login\n",
		"zdot/.zshenv":   "export USER_ENV=zdot-env\n",
		"zdot/.zprofile": "export USER_PROFILE=zdot-profile\n",
		"zdot/.zshrc":    "export USER_RC=zdot-rc\nPS1='user> '\n",
		"zdot/.zlogin":   "export USER_LOGIN=zdot-login\n",
		// bash's: a login shell reads .bash_profile, which reads .bashrc.
		".bash_profile": "export USER_PROFILE=profile\n. ~/.bashrc\n",
		".bashrc":       "export USER_RC=rc\nPS1='user> '\n",
		// fish's.
		".config/fish/config.fish": "set -gx USER_RC rc\nfunction fish_prompt; echo -n 'user> '; end\n",
	}
	for name, body := range files {
		p := filepath.Join(home, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	dir, err := os.MkdirTemp("/tmp", "rex")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("LEX_DIR", dir)
	// The terminal's library stays where it was fetched to, out of the
	// home of the test's own.
	if cache, err := os.UserCacheDir(); err == nil {
		t.Setenv("XDG_CACHE_HOME", cache)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	os.Unsetenv("ZDOTDIR")
	go Serve()
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(filepath.Join(dir, "server.sock")); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	c, err := Connect()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Shutdown()

	for _, tc := range []struct {
		name, shell, zdotdir string
		show, want           string
	}{
		{"bash", "bash", "", `echo "[$USER_PROFILE|$USER_RC]"`, "[profile|rc]"},
		{"zsh", "zsh", "", `echo "[$USER_ENV|$USER_PROFILE|$USER_RC|$USER_LOGIN|${ZDOTDIR-none}|${LEX_USER_ZDOTDIR-none}]"`,
			"[env|profile|rc|login|none|none]"},
		{"zsh with ZDOTDIR", "zsh", filepath.Join(home, "zdot"), `echo "[$USER_ENV|$USER_PROFILE|$USER_RC|$USER_LOGIN|${ZDOTDIR-none}]"`,
			"[zdot-env|zdot-profile|zdot-rc|zdot-login|" + filepath.Join(home, "zdot") + "]"},
		{"fish", "fish", "", `echo "[$USER_RC]"`, "[rc]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path, err := exec.LookPath(tc.shell)
			if err != nil {
				t.Skipf("no %s", tc.shell)
			}
			t.Setenv("SHELL", path)
			if tc.zdotdir != "" {
				t.Setenv("ZDOTDIR", tc.zdotdir)
			} else {
				os.Unsetenv("ZDOTDIR")
			}
			info, err := c.Create(CreateOptions{Dir: home, Cols: 100, Rows: 30, Prompt: PromptOhMyPosh})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Kill(info.ID)
			s := c.Stream(info.ID, 100, 30)
			defer s.Close()
			chunks := make(chan []byte, 64)
			go func() {
				defer close(chunks)
				for {
					buf := make([]byte, 4096)
					n, err := s.Read(buf)
					if n > 0 {
						chunks <- buf[:n]
					}
					if err != nil {
						return
					}
				}
			}()
			var out bytes.Buffer
			read := func(want string) bool {
				timeout := time.After(5 * time.Second)
				for !strings.Contains(out.String(), want) {
					select {
					case b, ok := <-chunks:
						if !ok {
							return false
						}
						out.Write(b)
					case <-timeout:
						return false
					}
				}
				return true
			}
			prompt := "OMP:" + tc.shell + ":yes> "
			if !read(prompt) {
				t.Fatalf("no prompt %q: %q", prompt, out.String())
			}
			s.Write([]byte(tc.show + "\r"))
			if !read(tc.want) {
				t.Fatalf("the shell shows %q, not %q", out.String(), tc.want)
			}
		})
	}
}

// TestWithPrompt leaves alone what Lex cannot style: commands of the
// window's own, shells it does not know, and no prompt asked for.
func TestWithPrompt(t *testing.T) {
	t.Setenv("LEX_DIR", t.TempDir())
	argv := []string{"-sh"}
	for _, tc := range []struct {
		prompt, name string
		cmd          []string
	}{
		{"", "zsh", nil},
		{PromptOhMyPosh, "sh", nil},
		{PromptOhMyPosh, "nu", nil},
		{PromptOhMyPosh, "zsh", []string{"htop"}},
	} {
		got, env := withPrompt(tc.prompt, tc.cmd, tc.name, argv)
		if !slices.Equal(got, argv) || env != nil {
			t.Errorf("%q %s %q: %q %q", tc.prompt, tc.name, tc.cmd, got, env)
		}
	}
	got, _ := withPrompt(PromptOhMyPosh, nil, "pwsh", []string{"pwsh.exe", "-NoExit", "-Command", "lex"})
	if !strings.HasPrefix(got[3], "if (Get-Command oh-my-posh") || !strings.HasSuffix(got[3], "lex") {
		t.Errorf("PowerShell runs %q", got)
	}
}
