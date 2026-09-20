FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

COPY . /src/mimap
WORKDIR /src/mimap

RUN <<-EOF
  set -ex

  apk add --update \
    git

  install -dm0755 /rootfs/usr/local/bin

  go build \
    -ldflags "-s -w -X main.version=$(git describe --tags --always || echo dev)" \
    -mod=readonly \
    -modcacherw \
    -trimpath \
    -o /rootfs/usr/local/bin/mimap
EOF


FROM scratch

LABEL maintainer="Knut Ahlers <knut@ahlers.me>"

COPY --from=builder /rootfs/ /

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/mimap"]
