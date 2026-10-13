FROM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder

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
