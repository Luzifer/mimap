FROM golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS builder

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
