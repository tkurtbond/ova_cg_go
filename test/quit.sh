#!/bin/bash
# quit.sh PROGRAM: the program, started for real, opens Character >
# Limits... on Ctrl+E with the cursor in Character Name, where it
# starts (a text input types Ctrl+H, I, J, L and M into itself, so a
# shortcut must be none of those); then it quits on Ctrl+Q (File >
# Quit's shortcut) with status 0. X11 only (xdotool sends the key), so
# make test-gui runs it under test-headless (Xvfb), and skips it on
# Wayland.
#
# Fyne draws its dialogs inside the window, so Limits is no window of
# its own for xdotool to find (po_ova_cg's quit.sh looked for one). That
# it opened is seen instead by what it does to the focus: it lets go of
# Character Name, so an x typed then changes nothing. Escape closing it
# is internal/ui's TestEscape.
#
# The preferences and Fyne's own settings go to a directory of the
# script's own, never the user's.
set -u
prog=$1
if [ -n "${WAYLAND_DISPLAY:-}" ] || [ -z "${DISPLAY:-}" ] || ! command -v xdotool >/dev/null; then
  echo "skip - quit.sh needs X11 and xdotool"; exit 0
fi
home=$(mktemp -d) || exit 2
trap 'rm -rf "$home"' EXIT
mkdir -p "$home/.cache" "$home/.config"
HOME=$home XDG_CONFIG_HOME=$home/.config "$prog" & pid=$!
# Wait for the window (up to 10 s).
win=
for i in $(seq 50); do
  win=$(xdotool search --name '^OVA Character Generator$' 2>/dev/null | head -1)
  [ -n "$win" ] && break; sleep 0.2
done
if [ -z "$win" ]; then echo "FAIL - the window never appeared"; kill $pid; exit 1; fi
sleep 1 # Fyne's first frame
title() { xdotool getwindowname "$win"; }
xdotool windowfocus --sync "$win" key ctrl+e; sleep 0.5
xdotool windowfocus --sync "$win" type x; sleep 0.5
if [ "$(title)" != "OVA Character Generator" ]; then
  echo "FAIL - Ctrl+E in Character Name didn't open Limits: the title is $(title)"; kill $pid; exit 1
fi
echo "ok   - Ctrl+E in Character Name opens Limits, the character unchanged"
xdotool windowfocus --sync "$win" key Escape; sleep 0.5
xdotool windowfocus --sync "$win" key ctrl+q
# It must exit by itself (up to 10 s).
for i in $(seq 50); do kill -0 $pid 2>/dev/null || break; sleep 0.2; done
if kill -0 $pid 2>/dev/null; then echo "FAIL - Ctrl+Q didn't end the program"; kill $pid; exit 1; fi
wait $pid; st=$?
if [ $st -eq 0 ]; then echo "ok   - Ctrl+Q quits, status 0"; else echo "FAIL - Ctrl+Q quit with status $st"; exit 1; fi
