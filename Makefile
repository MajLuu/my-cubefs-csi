CommitID=$(shell git rev-parse --short=8 HEAD)
Branch=$(shell git symbolic-ref --short -q HEAD)
BuildTime=$(shell date +%Y-%m-%dT%H:%M)
IMAGE_PREFIX=registry.cn-hangzhou.aliyuncs.com/docker-repo-lusx/cubefs
TAG?=v0.0.1

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
      -trimpath \
      -gcflags=-trimpath=$(shell pwd) -asmflags=-trimpath=$(shell pwd) \
      -ldflags="-s -w -X main.CommitID=${CommitID} -X main.BuildTime=${BuildTime} -X main.Branch=${Branch} " \
      -o bin/cfs-csi-driver ./cmd && echo "build cfs-csi-driver success"

images:
	docker build -t ${IMAGE_PREFIX}:${TAG} .
	docker push ${IMAGE_PREFIX}:${TAG}
	rm -rf bin/cfs-csi-driver