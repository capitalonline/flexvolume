FROM golang:1.13.6-alpine AS build-env
COPY . /go/src/github.com/capitalonline/flexvolume/
RUN cd /go/src/github.com/capitalonline/flexvolume/ && ./build.sh

FROM alpine:3.7
RUN apk --no-cache add fuse curl libxml2 openssl libstdc++ libgcc
COPY package /cds
COPY --from=build-env /go/src/github.com/capitalonline/flexvolume/flexvolume-linux /cds/flexvolume
RUN chmod 755 /cds/*

ENTRYPOINT ["/cds/entrypoint.sh"]
