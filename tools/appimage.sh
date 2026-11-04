#!/bin/sh
# Packs the Linux build of Lex as an AppImage, one file that runs on any
# x86-64 distribution with glibc 2.28 or later, GTK 3 and OpenGL, like the
# tarball: GTK and the graphics drivers are the system's, not bundled.
#
#   go tool mygo build -platform linux/amd64
#   sh tools/appimage.sh
#
# makes build/linux-amd64/Lex-<version>-x86_64.AppImage at mygo.json's
# version, with its .zsync for AppImageUpdate and the like. appimagetool
# and the AppImage runtime are downloaded once, at the versions below, into
# build/appimage-tools, and checked against their checksums. zsyncmake
# (apt install zsync) makes the .zsync; without it there is none.
set -eu

APPIMAGETOOL_VERSION=1.9.1
APPIMAGETOOL_SHA256=ed4ce84f0d9caff66f50bcca6ff6f35aae54ce8135408b3fa33abfc3cb384eb0
RUNTIME_VERSION=20251108
RUNTIME_SHA256=2fca8b443c92510f1483a883f60061ad09b46b978b2631c807cd873a47ec260d

# Where AppImageUpdate and the like look for a newer Lex: the .zsync of the
# latest release on GitHub.
UPDATE_INFO="gh-releases-zsync|richardkfm|lex|latest|Lex-*-x86_64.AppImage.zsync"

cd "$(dirname "$0")/.."
version="$(sed -n 's/^ *"version": *"\([^"]*\)".*/\1/p' mygo.json)"
build=build/linux-amd64
tools=build/appimage-tools
out="$build/Lex-$version-x86_64.AppImage"

tarball="$build/lex-$version-linux-amd64.tar.gz"
if [ ! -f "$tarball" ]; then
	echo "appimage.sh: no $tarball: run go tool mygo build -platform linux/amd64 first" >&2
	exit 1
fi

# fetch URL FILE SHA256 downloads URL into FILE, unless FILE is there with
# that checksum.
fetch() {
	if [ -f "$2" ] && echo "$3  $2" | sha256sum -c --status; then
		return
	fi
	curl -fsSL -o "$2.download" "$1"
	if ! echo "$3  $2.download" | sha256sum -c --status; then
		echo "appimage.sh: $1 does not have the checksum $3" >&2
		rm -f "$2.download"
		exit 1
	fi
	mv "$2.download" "$2"
	chmod +x "$2"
}
mkdir -p "$tools"
fetch "https://github.com/AppImage/appimagetool/releases/download/$APPIMAGETOOL_VERSION/appimagetool-x86_64.AppImage" \
	"$tools/appimagetool-$APPIMAGETOOL_VERSION" "$APPIMAGETOOL_SHA256"
fetch "https://github.com/AppImage/type2-runtime/releases/download/$RUNTIME_VERSION/runtime-x86_64" \
	"$tools/runtime-$RUNTIME_VERSION" "$RUNTIME_SHA256"

# The AppDir is the tarball's app, its library and resources next to the
# executable where it looks for them, its desktop entry and icon, which
# appimagetool wants at the top too, and AppRun, the entry point, a link
# to the executable.
appdir="$build/Lex.AppDir"
rm -rf "$appdir"
mkdir -p "$appdir"
tar -xzf "$tarball" -C "$appdir"
ln -s lex "$appdir/AppRun"
ln -s lex.png "$appdir/.DirIcon"

# zsyncmake writes the .zsync into the current directory, so appimagetool
# runs in the build's. It runs from its own extracted files, as FUSE may
# not be there (in containers and CI).
rm -f "$out" "$out.zsync"
tool="$(pwd)/$tools/appimagetool-$APPIMAGETOOL_VERSION"
runtime="$(pwd)/$tools/runtime-$RUNTIME_VERSION"
(
	cd "$build"
	ARCH=x86_64 APPIMAGE_EXTRACT_AND_RUN=1 VERSION="$version" "$tool" --no-appstream \
		--runtime-file "$runtime" -u "$UPDATE_INFO" \
		Lex.AppDir "Lex-$version-x86_64.AppImage"
)
rm -rf "$appdir"
echo "appimage.sh: built $out"
if [ -f "$out.zsync" ]; then
	echo "appimage.sh: built $out.zsync"
fi
