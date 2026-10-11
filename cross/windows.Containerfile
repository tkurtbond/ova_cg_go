# The Windows cross-compiler for ova_cg_go: Go and MinGW-w64, which
# compiles Fyne's GLFW C code (cgo) for windows/amd64. See CROSS.md.
ARG GO_VERSION=1.26.8
FROM docker.io/library/golang:${GO_VERSION}-trixie

RUN apt-get update \
 && apt-get install -y --no-install-recommends gcc-mingw-w64-x86-64 g++-mingw-w64-x86-64 \
 && rm -rf /var/lib/apt/lists/*

ENV GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
    CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    GOTOOLCHAIN=local
WORKDIR /src
