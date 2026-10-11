# The macOS cross-compiler for ova_cg_go: Go and osxcross (clang, Apple's
# ld64 and cctools), for darwin/amd64 and darwin/arm64. See CROSS.md.
#
# osxcross needs Apple's macOS SDK, which can't be downloaded or
# redistributed: put one packaged tarball (MacOSX<version>.sdk.tar.xz,
# made on a Mac with osxcross's tools/gen_sdk_package.sh) in cross/sdk/,
# which git ignores. The image holds the SDK: keep it local.
ARG GO_VERSION=1.26.8
FROM docker.io/library/golang:${GO_VERSION}-trixie

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      clang lld llvm-dev cmake git patch python3 make xz-utils bzip2 cpio \
      libssl-dev liblzma-dev libxml2-dev zlib1g-dev uuid-dev libbz2-dev \
 && rm -rf /var/lib/apt/lists/*

ARG OSXCROSS_COMMIT=27d21e4977c9751d01199c7a226a6faf494c3dd9
RUN git clone https://github.com/tpoechtrager/osxcross /osxcross \
 && git -C /osxcross checkout -q ${OSXCROSS_COMMIT}

COPY sdk/ /osxcross/tarballs/
RUN cd /osxcross && ls tarballs/MacOSX*.sdk.tar.* >/dev/null \
 && UNATTENDED=1 ./build.sh \
 && rm -rf build tarballs/MacOSX*

# 11.0, the first macOS on Apple silicon.
ENV PATH=/osxcross/target/bin:$PATH \
    MACOSX_DEPLOYMENT_TARGET=11.0 \
    GOOS=darwin CGO_ENABLED=1 \
    GOTOOLCHAIN=local
WORKDIR /src
