FROM golang:latest as BUILDER

MAINTAINER zengchen1024<chenzeng765@gmail.com>

# build binary
COPY . /go/src/github.com/opensourceways/xihe-sync-repo
WORKDIR /go/src/github.com/opensourceways/xihe-sync-repo
RUN GO111MODULE=on CGO_ENABLED=0 go build -o xihe-sync-repo
RUN tar -xf ./app/tools/obsutil.tar.gz

# copy binary config and utils
FROM alpine:3.14
RUN apk update && apk add --no-cache \
        git \
        bash \
        libc6-compat

RUN adduser mindspore -u 5000 -D
WORKDIR /opt/app
RUN chown -R mindspore:mindspore /opt/app

COPY --chown=mindspore:mindspore --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/xihe-sync-repo /opt/app
COPY --chown=mindspore:mindspore --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/obsutil /opt/app
COPY --chown=mindspore:mindspore --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/app/tools/sync_files.sh /opt/app

USER mindspore

RUN mkdir /opt/app/workspace

ENTRYPOINT ["/opt/app/xihe-sync-repo"]
