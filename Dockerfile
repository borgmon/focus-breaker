FROM golang:1.25-trixie

RUN apt update && apt install -y clang llvm lld bash patch xz-utils bzip2 zip && \
    ln -s /usr/bin/ld.lld /usr/bin/ld64.lld

# Install osxcross for macOS cross-compilation
RUN git clone --branch 2.0-llvm-based https://github.com/tpoechtrager/osxcross /osxcross

COPY MacOSX14.5.sdk.tar.xz /osxcross/tarballs/

RUN cd /osxcross && \
    UNATTENDED=yes OSX_VERSION_MIN=11.0 ./build.sh

ENV PATH="/osxcross/target/bin:${PATH}"

RUN go install fyne.io/tools/cmd/fyne@latest
WORKDIR /app
COPY . .

ENV CGO_ENABLED=1
ENV GOOS=darwin
ENV GOARCH=arm64
ENV CC=arm64-apple-darwin23.5-clang
ENV CXX=arm64-apple-darwin23.5-clang++

RUN fyne package -os darwin -icon assets/Icon.png -release
RUN zip -r focus-breaker-darwin.zip "Focus Breaker.app"
# RUN fyne package -os linux -release
ENTRYPOINT ["tail", "-f", "/dev/null"]
