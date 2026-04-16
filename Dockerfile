FROM golang:1.26-alpine@sha256:27f829349da645e287cb195a9921c106fc224eeebbdc33aeb0f4fca2382befa6 AS builder

COPY . /src/mimap
WORKDIR /src/mimap

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -modcacherw \
      -trimpath


FROM python:3.14-alpine@sha256:27ac3ba1699f7a526ad19bf0d35c12369b43d3439e08297a880398d97899c3d8

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
