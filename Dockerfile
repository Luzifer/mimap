FROM golang:1.25-alpine@sha256:f4622e3bed9b03190609db905ac4b02bba2368ba7e62a6ad4ac6868d2818d314 AS builder

COPY . /src/mimap
WORKDIR /src/mimap

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -modcacherw \
      -trimpath


FROM python:3.14-alpine@sha256:14558bb88225c495de95c0796c22de2c708500e48e9a6b84ce6b729d590fcb65

LABEL maintainer="Knut Ahlers <knut@ahlers.me>"

COPY pyproject.toml /src/

RUN set -ex \
 && apk --no-cache add \
      build-base \
      ca-certificates \
      jpeg-dev \
      zlib-dev \
 && pip install -e /src/ \
 && apk --no-cache del \
      build-base

COPY --from=builder /go/bin/mimap           /usr/local/bin/mimap
COPY --from=builder /src/mimap/build_map.py /src/

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/mimap"]
CMD ["--"]

# vim: set ft=Dockerfile:
