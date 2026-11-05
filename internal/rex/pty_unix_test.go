//go:build darwin || linux

package rex

import (
	"slices"
	"testing"
)

// TestShellCommand checks that the login shell starts as one, and fish
// without its greeting.
func TestShellCommand(t *testing.T) {
	for _, c := range []struct {
		shell string
		argv  []string
	}{
		{"/bin/zsh", []string{"-zsh"}},
		{"/usr/bin/fish", []string{"-fish", "-C", "function fish_greeting; end"}},
	} {
		t.Setenv("SHELL", c.shell)
		path, argv, name, err := shellCommand(nil)
		if err != nil || path != c.shell || !slices.Equal(argv, c.argv) || "-"+name != c.argv[0] {
			t.Errorf("SHELL=%s: %q %q %q %v, want %q", c.shell, path, argv, name, err, c.argv)
		}
	}
}
