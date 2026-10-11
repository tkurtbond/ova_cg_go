#!/usr/bin/env bash
# Cross-builds ova_cg_go in a podman container: build.sh windows|macos.
# CROSS.md explains the process.
#
# windows: obj/cross/windows/ova_cg_go.exe (windows/amd64).
# macos:   obj/cross/macos/ova_cg_go, a universal binary (darwin/amd64
#          and darwin/arm64); needs Apple's SDK in cross/sdk/ (see
#          macos.Containerfile).
#
# Each target's image (ova_cg_go-cross-TARGET) is built when missing, or
# again with REBUILD=1. The repository is mounted at /src and the output
# is written there; rootless podman maps the container's root to the
# user, so the files are the user's. The Go module and build caches are
# podman volumes, kept between runs.
set -euo pipefail

target=${1:?usage: build.sh windows|macos}
here=$(cd "$(dirname "$0")" && pwd)
repo=$(dirname "$here")
image=ova_cg_go-cross-$target
out=obj/cross/$target

case $target in
windows|macos) ;;
*) echo "build.sh: unknown target $target (windows or macos)" >&2; exit 2 ;;
esac

if [[ $target == macos ]] && ! compgen -G "$here/sdk/MacOSX*.sdk.tar.*" >/dev/null; then
  echo "build.sh: no macOS SDK: put MacOSX<version>.sdk.tar.xz in cross/sdk/ (see cross/macos.Containerfile)" >&2
  exit 1
fi

if [[ ${REBUILD:-} == 1 ]] || ! podman image exists "$image"; then
  podman build -t "$image" -f "$here/$target.Containerfile" "$here"
fi

mkdir -p "$repo/$out"

# label=disable: the repository is bind-mounted without relabelling it
# for SELinux.
run() {
  podman run --rm --security-opt label=disable \
    -v "$repo:/src" -w /src \
    -v ova_cg_go-gomod:/go/pkg/mod \
    -v "ova_cg_go-gocache-$target:/root/.cache/go-build" \
    "$@"
}

# -s -w: no symbol table or DWARF, as a release wants.
case $target in
windows)
  # -H=windowsgui: a GUI program, started without a console window.
  run "$image" go build -trimpath -ldflags '-s -w -H=windowsgui' \
    -o "$out/ova_cg_go.exe" ./cmd/ova_cg_go
  ;;
macos)
  run "$image" bash -c "
    set -e
    GOARCH=amd64 CC=o64-clang CXX=o64-clang++ \
      go build -trimpath -ldflags '-s -w' -o $out/ova_cg_go-amd64 ./cmd/ova_cg_go
    GOARCH=arm64 CC=oa64-clang CXX=oa64-clang++ \
      go build -trimpath -ldflags '-s -w' -o $out/ova_cg_go-arm64 ./cmd/ova_cg_go
    lipo=\$(ls /osxcross/target/bin/*-lipo | head -1)
    \$lipo -create -output $out/ova_cg_go $out/ova_cg_go-amd64 $out/ova_cg_go-arm64
    rm $out/ova_cg_go-amd64 $out/ova_cg_go-arm64
  "
  ;;
esac

ls -l "$repo/$out"
