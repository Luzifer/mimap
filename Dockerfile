FROM golang:1.27.4-alpine@sha256:0a5f7fe796599ff487605987432e7619ae45beeb2fba4c2ce0b907879932aebf AS builder

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
