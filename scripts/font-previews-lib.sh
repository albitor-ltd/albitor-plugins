#!/usr/bin/env bash
# Sourced by subset-previews.sh and check-font-previews.sh; needs $root set.
# shellcheck shell=bash disable=SC2154

font_python() { # print a python with the pinned tools, creating the venv on first use
  local key venv
  key="$(sha256sum "$root/scripts/font-previews.pin" | cut -c1-12)"
  venv="${FONT_TOOLS_VENV:-${TMPDIR:-/tmp}/albitor-plugins-fonttools-$key}"
  if [[ ! -x "$venv/bin/python" ]] || ! "$venv/bin/python" -c 'import fontTools, brotli' 2>/dev/null; then
    rm -rf "$venv"
    python3 -m venv "$venv" >&2
    "$venv/bin/pip" install --quiet --disable-pip-version-check -r "$root/scripts/font-previews.pin" >&2
  fi
  echo "$venv/bin/python"
}
