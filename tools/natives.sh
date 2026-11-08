#!/bin/sh
# Downloads the native libraries that Lex's mygo plugins name, for the
# platforms given, into the user's cache, where `go test` and `go tool mygo
# build` look before they download them themselves:
#
#   sh tools/natives.sh linux-amd64 windows-amd64
#
# puts each at ${XDG_CACHE_HOME:-~/.cache}/mygo/natives/<sha256>/<name>,
# checked against its SHA-256, and leaves one already there alone. CI runs
# it before the tests and the build: they download a missing library with
# a few seconds of retries, and GitHub's release downloads answer 504 for
# minutes at times. This tries for up to five minutes a file. Needs go,
# jq, curl and sha256sum.
set -eu

if [ $# -eq 0 ]; then
	echo "usage: sh tools/natives.sh <os>-<arch>..." >&2
	exit 2
fi
cache="${XDG_CACHE_HOME:-$HOME/.cache}/mygo/natives"

# The manifests of the plugins Lex builds with, in mygo's module.
manifests=$(go list -deps -f '{{if .Module}}{{if eq .Module.Path "github.com/egoist/mygo"}}{{.Dir}}{{end}}{{end}}' ./... |
	while read -r dir; do
		if [ -f "$dir/mygo-plugin.json" ]; then echo "$dir/mygo-plugin.json"; fi
	done | sort -u)
if [ -z "$manifests" ]; then
	echo "natives: no mygo plugin names a native library" >&2
	exit 1
fi

fetch() {
	sha=$1 name=$2 url=$3
	path="$cache/$sha/$name"
	if [ -f "$path" ] && echo "$sha  $path" | sha256sum -c --status; then
		echo "natives: have $name ($sha)"
		return
	fi
	mkdir -p "$cache/$sha"
	curl -fsSL --retry 10 --retry-all-errors --retry-delay 15 --retry-max-time 300 -o "$path.part" "$url"
	if ! echo "$sha  $path.part" | sha256sum -c --status; then
		rm -f "$path.part"
		echo "natives: $url is not $sha" >&2
		exit 1
	fi
	mv "$path.part" "$path"
	echo "natives: downloaded $name ($sha)"
}

for target in "$@"; do
	found=
	for manifest in $manifests; do
		files=$(jq -r --arg t "$target" '.libraries[].files[$t] // empty | [.sha256, .name, .url] | @tsv' "$manifest")
		if [ -z "$files" ]; then continue; fi
		found=1
		echo "$files" | while IFS="$(printf '\t')" read -r sha name url; do
			fetch "$sha" "$name" "$url"
		done
	done
	if [ -z "$found" ]; then
		echo "natives: no library for $target" >&2
		exit 1
	fi
done
