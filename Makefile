CommitID=$(shell git rev-parse --short=8 HEAD)
Branch=$(shell git symbolic-ref --short -q HEAD)
BuildTime=$(shell date +%Y-%m-%dT%H:%M)

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -gcflags=-trimpath=$(shell pwd) -asmflags=-trimpath=$(shell pwd) \
      -ldflags="-s -w -X main.CommitID=${CommitID} -X main.BuildTime=${BuildTime} -X main.Branch=${Branch} " \
      -o bin/cfs-csi-driver ./cmd && echo "build cfs-csi-driver success"