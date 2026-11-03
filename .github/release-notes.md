## Install

**Windows 10 1809 or later (x64):** run `Lex-Setup-<version>-windows-amd64.exe`. It installs Lex for
the current user, without administrator rights. The installer isn't signed, so SmartScreen may warn:
choose *More info → Run anyway*. Before you upgrade, run **Quit and End All Sessions** from the command
palette (Ctrl+Shift+P), because the session server keeps `Lex.exe` in use.

**Linux (x86-64, glibc 2.28 or later, GTK 3):**

- Debian or Ubuntu: `sudo apt install ./lex_<version>_amd64.deb`
- Any distribution, without root: `sh install.sh lex-<version>-linux-amd64.tar.gz`

`SHA256SUMS` lists the checksums of the files.

Lex is forked from [GoRex](https://github.com/egoist/gorex) by EGOIST, after
[Superlogical's Rex](https://www.superlogical.com). See the Credits in the README.
