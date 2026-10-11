# ova_cg_go -- an OVA character generator in Go, with Fyne for its window
# and yaml.v3 for its files: the port of ova_cg_po (Oberon-2, FLTK).
# See AGENTS.md and PLAN.md.
#
# go builds and caches everything itself; make only names the steps and
# keeps what they write in $(OBJ), and the program in ./$(PROGRAM).

SHELL = bash

GO   ?= go
# Build tags: wayland builds Fyne for Wayland instead of X11.
TAGS ?=
GOFLAGS_TAGS := $(if $(TAGS),-tags '$(TAGS)')

OBJ     := obj
PROGRAM := ova_cg_go

# Seconds a test may run before it counts as hung, and how long the
# program must keep running to count as started (test-gui).
TIMEOUT ?= 10m
RUNTIME ?= 3

.PHONY: all test check check-pages test-viewers test-gui test-headless test-sway display \
  install uninstall clean $(PROGRAM)
.NOTPARALLEL:

all: $(PROGRAM)

$(OBJ):
	mkdir -p $@

# go decides what needs building, so this always asks it.
$(PROGRAM):
	$(GO) build $(GOFLAGS_TAGS) -o $@ ./cmd/ova_cg_go

# ova_fmt_po, the oracle for files: what the generator saves must format
# as the file it loaded (test/files.sh, after internal/save's test has
# written obj/files/).
OVA_FMT_PO ?= $(HOME)/Repos/RPG/Tools/ova_fmt_po/ova_fmt_po

# What needs no display: the Go tests (the window's included: Fyne's
# test driver opens nothing), test/files.sh, then test/cli.sh; fail at
# the end if any failed.
test: $(PROGRAM)
	@status=0; \
	echo "== go test"; $(GO) test $(GOFLAGS_TAGS) -count=1 -timeout $(TIMEOUT) ./... || status=1; \
	echo "== files.sh"; test/files.sh $(OVA_FMT_PO) || status=1; \
	echo "== cli.sh"; test/cli.sh ./$(PROGRAM) || status=1; \
	exit $$status

check:
	$(GO) vet $(GOFLAGS_TAGS) ./...
	$(GO) test $(GOFLAGS_TAGS) -count=1 -race -timeout $(TIMEOUT) ./...

# Every page in package rules against the rulebook: the entries'
# headings, found by their font in the PDF (tools/rulebook-headings.py,
# which needs mutool). Not part of make test, which can't count on the
# PDF.
RULEBOOK ?= $(HOME)/Reference/RPG/OVA/OVA_The_Anime_Role-Playing_Game.pdf
check-pages: | $(OBJ)
	@$(GO) build -o $(OBJ)/listrules ./cmd/listrules
	@tools/rulebook-headings.py $(RULEBOOK) > $(OBJ)/headings.txt
	@$(OBJ)/listrules > $(OBJ)/rules.txt
	@if diff $(OBJ)/headings.txt $(OBJ)/rules.txt; then \
	  echo "ok   - all $$(wc -l < $(OBJ)/rules.txt) pages match the rulebook's headings"; \
	else echo "FAIL - package rules (>) differs from the rulebook's headings (<)"; exit 1; fi

# package viewer with the real viewers, okular, evince and mupdf, on a
# headless sway of test/viewer.sh's own, on a private D-Bus session bus,
# with a scratch HOME. Not part of make test: it needs the viewers, and
# takes a minute or so.
test-viewers: | $(OBJ)
	$(GO) build -o $(OBJ)/openpage ./cmd/openpage
	test/viewer.sh $(OBJ)/openpage

# An agent never runs the program on the user's desktop (AGENTS.md): use
# test-headless or test-sway, which give test-gui a display of its own.
display:
	@if [ -z "$$DISPLAY$$WAYLAND_DISPLAY" ]; then \
	  echo "test-gui needs a display: use make test-headless or make test-sway"; exit 1; \
	fi

# The program itself, which waits for the user: it must still be running
# after RUNTIME seconds, when timeout ends it (status 124): it started,
# drew, and didn't crash. Then test/quit.sh: started again, Ctrl+E opens
# Limits from a text input, Escape, and it quits on Ctrl+Q (X11 only;
# skipped on Wayland). Its preferences and Fyne's settings are kept in
# obj/home (quit.sh's in a directory of its own), never the user's.
test-gui: $(PROGRAM) display | $(OBJ)
	@mkdir -p $(OBJ)/home/.config $(OBJ)/home/.cache; \
	status=0; \
	echo "== $(PROGRAM) (must still be running after $(RUNTIME) s)"; \
	HOME=$(CURDIR)/$(OBJ)/home XDG_CONFIG_HOME=$(CURDIR)/$(OBJ)/home/.config \
	  XDG_CACHE_HOME=$(CURDIR)/$(OBJ)/home/.cache timeout $(RUNTIME) ./$(PROGRAM); st=$$?; \
	if [ $$st -eq 124 ]; then echo "ok   - $(PROGRAM) started and ran"; \
	else echo "FAIL - $(PROGRAM) exited with $$st"; status=1; fi; \
	echo "== quit.sh"; test/quit.sh ./$(PROGRAM) || status=1; \
	exit $$status

# On a virtual X server (Xvfb). WAYLAND_DISPLAY is unset so that nothing
# reaches the desktop's compositor.
test-headless: $(PROGRAM)
	env -u WAYLAND_DISPLAY xvfb-run -a $(MAKE) test-gui

# On Wayland, on a headless sway (tools/with-sway.sh), built for Wayland.
test-sway:
	tools/with-sway.sh $(MAKE) TAGS=wayland test-gui

# ---- Installation ----
#
# To $(PREFIX)/bin, $HOME/local/bin by default, as ova_cg_po's. DESTDIR
# is prepended, for staging.
PREFIX ?= $(HOME)/local
BINDIR ?= $(PREFIX)/bin

install: $(PROGRAM)
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 $(PROGRAM) $(DESTDIR)$(BINDIR)/$(PROGRAM)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(PROGRAM)

clean:
	rm -rf $(OBJ) $(PROGRAM)

print-%: ; @echo $* = $($*)
