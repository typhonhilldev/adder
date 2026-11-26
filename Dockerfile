# The image GoReleaser builds for each release: adder on the Go toolchain, so scans with typeCheck work too.
FROM golang:1.27-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280

ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/adder /usr/local/bin/adder

# Caches in /tmp, so the image also works as another user, such as with --user "$(id -u):$(id -g)".
ENV GOCACHE=/tmp/go/cache GOMODCACHE=/tmp/go/mod
WORKDIR /src
ENTRYPOINT ["adder"]
