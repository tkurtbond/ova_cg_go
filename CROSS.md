# CROSS.md - building ova_cg_go for Windows and macOS

ova_cg_go is built and tested on Linux. Builds for Windows and macOS
are made on Linux too, each inside a podman container that holds a
cross-compiler. This file says how to run them, how they work, what has
been checked, and what hasn't.

| Command | Makes | Needs |
|---|---|---|
| `make windows` | `obj/cross/windows/ova_cg_go.exe`: 64-bit Windows, a GUI program | podman |
| `make macos` | `obj/cross/macos/ova_cg_go`: one binary for Intel Macs and Apple silicon, macOS 11 or later | podman, and Apple's macOS SDK in `cross/sdk/` |
| `make clean-cross` | removes the images and the cached Go modules | |

`make clean` removes `obj/`, the binaries included, but not the images
or the caches.

## What you need

- **podman**, rootless (as the user, not root). Docker isn't supported:
  `cross/build.sh` calls `podman`.
- **The network**, the first time: podman pulls the Go image, `apt-get`
  installs the cross-compiler, and Go downloads the modules. The macOS
  image also clones osxcross from GitHub. Later runs use what was
  downloaded and need no network.
- **Disk space**: the Go image is about 900 MB, the Windows image
  1.7 GB (the Go image included). The macOS image is larger: it holds
  LLVM, osxcross and the SDK.
- **For macOS only: Apple's macOS SDK**, version 11 or later (below).

## How it works

The files are in `cross/`:

```
cross/build.sh               builds the image if missing, then the program in it
cross/windows.Containerfile  the Windows image: Go and MinGW-w64
cross/macos.Containerfile    the macOS image: Go and osxcross
cross/sdk/                   where the macOS SDK goes; git ignores it
```

`make windows` and `make macos` only run `cross/build.sh windows` or
`cross/build.sh macos`. The script:

1. Builds the image `ova_cg_go-cross-windows` (or `-macos`) from its
   Containerfile, if podman doesn't have it yet. `REBUILD=1` builds it
   again even if it does (`REBUILD=1 make windows`).
2. Runs `go build` in a container from that image, with:
   - the repository mounted at `/src`. Rootless podman maps the
     container's root to you, so the binaries it writes are yours.
   - `--security-opt label=disable`, so SELinux lets the container read
     the repository without podman relabelling your files (as `:Z`
     would).
   - the podman volume `ova_cg_go-gomod` as Go's module cache, shared by
     both targets, and `ova_cg_go-gocache-windows` (or `-macos`) as its
     build cache. The volumes outlive the containers, so a second build
     downloads nothing and compiles only what changed.
3. Lists what it wrote in `obj/cross/windows/` or `obj/cross/macos/`.

Both builds use `-trimpath` (no paths from this machine in the binary)
and `-ldflags '-s -w'` (no symbol table or debugging information).

Both images start from the official `golang:1.26.8-trixie` image (Go on
Debian 13). They set `GOTOOLCHAIN=local`, so Go uses the image's own
version and never downloads another one in the middle of a build.

Fyne draws with OpenGL through GLFW, which is C, so the program can't
be built with `CGO_ENABLED=0`: each target needs a C compiler for that
system. That is what the containers are for.

## Windows

```sh
make windows
```

The image adds Debian's MinGW-w64 (`gcc-mingw-w64-x86-64`), a GCC that
makes Windows programs, and sets `GOOS=windows`, `GOARCH=amd64`,
`CGO_ENABLED=1` and `CC=x86_64-w64-mingw32-gcc`. The build adds
`-H=windowsgui` to the link flags, so the program opens no console
window when it starts.

The first run took about four minutes, mostly downloads; later runs
take seconds, plus however long Fyne's C code takes to compile if the
build cache was emptied.

The result is one file, `ova_cg_go.exe`, about 25 MB. Copy it to the
Windows machine and run it; it needs no installer, and the only DLLs it
uses are Windows' own (GDI32, KERNEL32, msvcrt, OPENGL32, SHELL32,
USER32).

### Trying it under Wine

Wine can run the `.exe` on Linux. Run it on Xvfb, never on the desktop
(AGENTS.md), with a Wine prefix of its own, so that neither Wine nor
the program writes your files:

```sh
S=$(mktemp -d)
env -u WAYLAND_DISPLAY -u DBUS_SESSION_BUS_ADDRESS \
  WINEPREFIX=$S/prefix WINEDEBUG=-all \
  WINEDLLOVERRIDES="mscoree,mshtml=" \
  xvfb-run -a bash -c 'wineboot -i >/dev/null 2>&1
    timeout 25 wine obj/cross/windows/ova_cg_go.exe; echo "status $?"
    wineserver -k'
```

`status 124` means it was still running after 25 seconds, when
`timeout` ended it: it started, drew and didn't crash. This is the same
check `make test-gui` makes on Linux. The DRI3 warnings Mesa prints on
Xvfb don't matter. Wine passes neither `HOME` nor `XDG_CONFIG_HOME`
to a Windows program, so the program's preferences go to the Wine
prefix's `%AppData%`, under `$S/prefix`, and never reach your own.

## macOS

### The SDK

osxcross, the cross-compiler, needs the headers and libraries of
Apple's macOS SDK. Apple's licence doesn't let anyone redistribute the
SDK, so neither osxcross nor this repository includes it: you make an
SDK package from Xcode, or from the Xcode Command Line Tools, yourself.
Read the [Xcode licence](https://www.apple.com/legal/sla/docs/xcode.pdf)
first. osxcross's
[README.SDK.md](https://github.com/tpoechtrager/osxcross/blob/master/README.SDK.md)
has all the ways; the usual two:

- **On a Mac.** Download Xcode (or the Command Line Tools) from
  [developer.apple.com](https://developer.apple.com/download/all/) and
  get osxcross (`git clone https://github.com/tpoechtrager/osxcross`).
  - From Xcode: mount `Xcode.dmg` (Xcode needn't be installed), then
    run `./tools/gen_sdk_package.sh` in osxcross.
  - From the Command Line Tools: install them, then run
    `./tools/gen_sdk_package_tools.sh`.
- **On Linux**, from Xcode's `.xip` (Xcode 8 or later; needs up to
  45 GB free): install clang, make and the OpenSSL, lzma and libxml2
  development packages, then run `./tools/gen_sdk_package_pbzx.sh
  Xcode_<version>.xip` in osxcross.

Either way you get a file such as `MacOSX15.5.sdk.tar.xz`. Put it in
`cross/sdk/` here. `cross/build.sh` won't start without a
`cross/sdk/MacOSX*.sdk.tar.*`. Use one SDK only, version 11 or later:
the build targets macOS 11, and osxcross picks the SDK by the file's
name.

### Building

```sh
make macos
```

The first run builds the image, which:

1. installs clang, LLVM and what osxcross needs to build;
2. clones osxcross at the commit pinned in `macos.Containerfile`
   (`OSXCROSS_COMMIT`), so a later change there can't change the build
   unannounced;
3. copies `cross/sdk/` into the image and runs osxcross's `build.sh`
   with `UNATTENDED=1`, which answers its questions with the defaults:
   the "stable" flavour, Apple's cctools and ld64 linker built from
   source. That takes a while: it compiles the linker and the tools.
4. deletes osxcross's build directory and the SDK tarball (the SDK
   itself is unpacked in `/osxcross/target`, where the compilers use
   it).

**The image holds the SDK, so keep it on this machine**: don't push it
to a registry.

Then the build compiles the program twice in the container, for
`GOARCH=amd64` with `CC=o64-clang` and for `GOARCH=arm64` with
`CC=oa64-clang` (osxcross's names for its clang for each architecture),
and joins the two into one universal binary with osxcross's `lipo`.
`MACOSX_DEPLOYMENT_TARGET=11.0` (macOS Big Sur, the first to run on
Apple silicon) is the oldest macOS the binary runs on.

You can look at the result on Linux:

```sh
file obj/cross/macos/ova_cg_go   # Mach-O universal binary with 2 architectures
```

### Running it on a Mac

Copy `ova_cg_go` to the Mac and run it from Terminal
(`./ova_cg_go`). It is a bare program, not an `.app`, so Finder won't
treat it as an application.

The binary isn't signed or notarised. If you copied it with a browser
or AirDrop, macOS marks it as downloaded and Gatekeeper refuses it;
clear the mark with

```sh
xattr -d com.apple.quarantine ova_cg_go
```

On Apple silicon, macOS won't run arm64 code with no signature at all.
Apple's linker signs it ad hoc by itself; whether osxcross's does too
hasn't been checked. If the Mac kills the program at once ("Killed:
9"), sign it ad hoc on the Mac:

```sh
codesign --force --sign - ova_cg_go
```

## Changing versions

- **Go**: change `ARG GO_VERSION` in both Containerfiles (`go.mod`'s
  `go` line is the oldest that works, and `GOTOOLCHAIN=local` makes an
  older one an error, not a download), then `REBUILD=1 make windows`
  and `REBUILD=1 make macos`.
- **osxcross**: change `OSXCROSS_COMMIT` in `macos.Containerfile` to
  the commit you want, then `REBUILD=1 make macos`.
- **The SDK**: replace the tarball in `cross/sdk/`, then
  `REBUILD=1 make macos`.
- **MinGW-w64** comes from Debian; `REBUILD=1 make windows` gets
  Debian's latest for the image's release.

## What hasn't been done

- **The macOS build has never been run**: there was no SDK when the
  containers were set up (2026-10-11). The Containerfile and the build
  steps follow osxcross's documentation at the pinned commit; expect
  to fix something the first time.
- **Neither binary has been run on Windows or a Mac.** The `.exe` has
  run under Wine only.
- **No packaging.** There is no icon, no `.app` bundle, no installer,
  and `fyne package` isn't used. An `.app` needs an icon and
  `FyneApp.toml` first.
- **No signing or notarisation**, for either system. Windows'
  SmartScreen may warn about the `.exe` the first time.
- **The rulebook viewer.** The program's default viewer, when none is
  installed of okular, evince and mupdf, is `xdg-open`, which neither
  Windows nor macOS has. Choose a command with **Rulebook >
  Viewer...**: `open %f` on macOS (Preview, or your default PDF
  viewer, at no particular page), or, with [SumatraPDF](https://www.sumatrapdfreader.org/),
  `SumatraPDF.exe -page %p %f` on Windows. A command with spaces in its
  path won't work: the template is split at blanks.
- **Closing the viewer.** With mupdf, the program ends the viewer it
  started before starting another. It recognises its own viewer by its
  start time in `/proc`, which only Linux has, so on Windows and macOS
  it never ends one.
- **The tests run only on Linux.** `internal/spawn`'s and
  `internal/viewer`'s tests use Unix calls and don't compile for
  Windows.

## Source changes for Windows

Two packages needed changes.

**Preferences.** The program keeps its preferences in
`$XDG_CONFIG_HOME/ova_cg_go/ova_cg_go.prefs`, else under
`$HOME/.config`. Windows usually sets neither, which would have put the
file in `.config\ova_cg_go\` under whatever folder the program was
started from. On Windows, when `XDG_CONFIG_HOME` isn't set,
`prefs.DefaultPath` now uses `os.UserConfigDir`, which is `%AppData%`:
`C:\Users\NAME\AppData\Roaming\ova_cg_go\ova_cg_go.prefs`. A
Windows build under Wine printed that path. macOS has `HOME`, so it
keeps `~/.config`.

**Starting and ending the viewer.** `internal/spawn` starts the viewer in a session of
its own (`Setsid`) and ends it with `SIGTERM` (`syscall.Kill`), and
Windows has neither. Those two calls are in `spawn_unix.go`, built for
everything but Windows, and `spawn_windows.go`, which starts the viewer
in a process group of its own (`CREATE_NEW_PROCESS_GROUP`) and ends it
with `TerminateProcess`. macOS has `Setsid` and `SIGTERM`, so it builds
the Unix file. Linux behaves as before.

## Troubleshooting

- **`build.sh: no macOS SDK`**: there is no `MacOSX*.sdk.tar.*` in
  `cross/sdk/`.
- **Permission denied reading `/src` in the container**: SELinux. The
  script passes `--security-opt label=disable`; if you run podman by
  hand, pass it too.
- **`go: go.mod requires go >= ...`**: `go.mod` was raised past the
  image's Go. Raise `GO_VERSION` (above).
- **An old compiler or SDK after changing a Containerfile**: the script
  builds an image only when it is missing. Use `REBUILD=1`.
- **Odd failures after an image update**: `make clean-cross` removes
  the images and the caches, and the next build starts from nothing.
