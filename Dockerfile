FROM golang:latest as BUILDER

MAINTAINER zengchen1024<chenzeng765@gmail.com>

# build binary
WORKDIR /go/src/github.com/opensourceways/xihe-sync-repo
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 go build -a -o xihe-sync-repo .
RUN tar -xf ./app/tools/obsutil.tar.gz

# copy binary config and utils
FROM alpine:3.14
RUN apk update && apk add --no-cache \
        git \
        bash \
        libc6-compat
COPY --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/xihe-sync-repo /opt/app/xihe-sync-repo
COPY --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/obsutil /opt/app/obsutil
COPY --from=BUILDER /go/src/github.com/opensourceways/xihe-sync-repo/app/tools/sync_files.sh /opt/app/sync_file.sh

ENTRYPOINT ["/opt/app/xihe-sync-repo"]
