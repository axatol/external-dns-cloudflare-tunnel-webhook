OUTPUT ?= dist/app
BUILD_TIME ?= $(shell date +"%Y-%m-%dT%H:%M:%S%z")
BUILD_COMMIT ?= $(shell git rev-parse HEAD)
IMAGE_PLATFORM ?= linux/amd64
IMAGE_NAME ?= axatol/external-dns-cloudflare-tunnel-webhook
IMAGE_TAG ?= latest

LDFLAGS = -s -w
LDFLAGS += -X main.buildTime=$(BUILD_TIME)
LDFLAGS += -X main.buildCommit=$(BUILD_COMMIT)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT)

image:
	docker build \
		--tag $(IMAGE_NAME):$(IMAGE_TAG) \
		--platform $(IMAGE_PLATFORM) \
		--build-arg BUILD_COMMIT=$(BUILD_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
