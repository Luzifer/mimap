FROM golang:alpine as builder

COPY . /go/src/github.com/Luzifer/mimap
WORKDIR /go/src/github.com/Luzifer/mimap

RUN set -ex \
 && apk add --update git \
 && go install -ldflags "-X main.version=$(git describe --tags || git rev-parse --short HEAD || echo dev)"

FROM python:3.9-alpine

LABEL maintainer "Knut Ahlers <knut@ahlers.me>"

COPY --from=builder /go/src/github.com/Luzifer/mimap/requirements.txt /src/requirements.txt

RUN set -ex \
 && apk --no-cache add \
      build-base \
      ca-certificates \
      jpeg-dev \
      zlib-dev \
 && pip install -r /src/requirements.txt \
 && apk --no-cache del \
      build-base

COPY --from=builder /go/bin/mimap /usr/local/bin/mimap
COPY --from=builder /go/src/github.com/Luzifer/mimap/build_map.py /src/

EXPOSE 3000
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/mimap"]
CMD ["--"]

# vim: set ft=Dockerfile:
