package rex

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// inspect returns the name, arguments and working directory of a process.
func inspect(pid int) (p procInfo) {
	if pid <= 0 {
		return
	}
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		p.name = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
		p.args = strings.Split(strings.TrimRight(string(b), "\x00"), "\x00")
	}
	p.dir, _ = os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	return p
}

func sysctlString(string) string { return "" }

// linuxHost fills in what Linux tells of the machine: its model, from the
// firmware or the device tree, its processor, its memory, and whether it
// has a battery.
func linuxHost(h *HostInfo) {
	h.Model = "Linux"
	for _, f := range []string{"/sys/class/dmi/id/product_name", "/sys/firmware/devicetree/base/model"} {
		if b, err := os.ReadFile(f); err == nil {
			if m := strings.TrimSpace(strings.TrimRight(string(b), "\x00")); m != "" && !placeholderModel(m) {
				h.Model = m
				break
			}
		}
	}
	h.Chip = procField("/proc/cpuinfo", "model name")
	if kb, err := strconv.ParseUint(strings.TrimSuffix(procField("/proc/meminfo", "MemTotal"), " kB"), 10, 64); err == nil {
		h.Memory = kb << 10
	}
	if bats, _ := filepath.Glob("/sys/class/power_supply/BAT*"); len(bats) > 0 {
		h.Portable = true
	}
}

// placeholderModel reports a model name that firmware leaves for the maker
// of the machine to fill in, and that names nothing.
func placeholderModel(m string) bool {
	switch strings.ToLower(m) {
	case "to be filled by o.e.m.", "system product name", "default string", "none", "not applicable", "o.e.m.":
		return true
	}
	return false
}

// procField returns the value of the first line "key: value" of a file of
// /proc, as /proc/cpuinfo and /proc/meminfo have them.
func procField(file, key string) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), ":")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func sysctlUint64(string) uint64 { return 0 }
