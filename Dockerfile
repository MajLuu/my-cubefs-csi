FROM golang:1.22.9-alpine3.20 as builder

WORKDIR /workspace
ADD . .
ENV GOPROXY=https://goproxy.cn,direct

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -gcflags=-trimpath=$(shell pwd) -asmflags=-trimpath=$(shell pwd) \
      -ldflags="-s -w -X main.CommitID=${CommitID} -X main.BuildTime=${BuildTime} -X main.Branch=${Branch} " \
      -o bin/cfs-csi-driver ./cmd && echo "build cfs-csi-driver success"

FROM alpine:3.20

COPY --from=builder /workspace/bin/cfs-csi-driver /cfs-csi-driver

# TODO: use tini as entrypoint

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk add --no-cache bind-tools fuse tzdata && \
    mkdir -p /cfs/bin /cfs/conf /cfs/logs && \
    chmod +x /cfs-csi-driver
