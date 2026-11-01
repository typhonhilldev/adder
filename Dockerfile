# The image GoReleaser builds for each release: adder on the Go toolchain, so scans with typeCheck work too.
FROM golang:1.27-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/adder /usr/local/bin/adder

# Caches in /tmp, so the image also works as another user, such as with --user "$(id -u):$(id -g)".
ENV GOCACHE=/tmp/go/cache GOMODCACHE=/tmp/go/mod
WORKDIR /src
ENTRYPOINT ["adder"]
