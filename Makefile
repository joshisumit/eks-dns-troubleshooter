# Usage:
#.PHONY: all container build push


# set Go specific env variables
export GOARCH = amd64
export GOOS = linux
export CGO_ENABLED = 0
export GO111MODULE = on
export GOPROXY = direct

TAG=v1.0.0
IMAGE_NAME=sumitj/eks-dnshooter
ARCH=amd64
OS=linux
PKG=github.com/joshisumit/eks-dns-troubleshooter
REPO_INFO=$(shell git config --get remote.origin.url)
GIT_COMMIT=git-$(shell git rev-parse --short HEAD)
BINARY_NAME=eks-dnshooter

LDFLAGS=-X $(PKG)/version.COMMIT=$(GIT_COMMIT) -X $(PKG)/version.RELEASE=$(TAG) -X $(PKG)/version.REPO=$(REPO_INFO)
BUILD_FLAGS=-ldflags '$(LDFLAGS)'


all: container

# Build go app
build:
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) ./cmd/

# Build Docker Image
container:
	docker build -t $(IMAGE_NAME):$(TAG) .

push:
	docker push $(IMAGE_NAME):$(TAG)
