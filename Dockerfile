FROM golang:1.26.4-alpine@sha256:3ad57304ad93bbec8548a0437ad9e06a455660655d9af011d58b993f6f615648 AS builder

COPY . /src/mimap
WORKDIR /src/mimap

RUN set -ex \
 && apk add --update git \
 && go install \
      -ldflags "-X main.version=$(git describe --tags --always || echo dev)" \
      -mod=readonly \
      -modcacherw \
      -trimpath


FROM python:3.14-alpine@sha256:26730869004e2b9c4b9ad09cab8625e81d256d1ce97e72df5520e806b1709f92

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
