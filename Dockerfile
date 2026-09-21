FROM golang:1.27.1-alpine@sha256:4cb7ac979db5fcc41cae44b2227ba5ab8a51e8807f40d9ba4dee20a0ad960b5b AS builder

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
