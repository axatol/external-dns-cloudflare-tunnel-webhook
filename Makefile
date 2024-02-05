OUTPUT ?= dist/app
BUILD_TIME ?= $(shell date +"%Y-%m-%dT%H:%M:%S%z")
BUILD_COMMIT ?= $(shell git rev-parse HEAD)
IMAGE_PLATFORM ?= linux/amd64
IMAGE_NAME ?= public.ecr.aws/axatol/external-dns-cloudflare-tunnel-webhook
IMAGE_TAG ?= latest

LDFLAGS = -s -w
LDFLAGS += -X main.buildTime=$(BUILD_TIME)
LDFLAGS += -X main.buildCommit=$(BUILD_COMMIT)

build-binary:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT)

build-image:
	docker build \
		--tag $(IMAGE_NAME):$(IMAGE_TAG) \
		--platform $(IMAGE_PLATFORM) \
		--build-arg BUILD_COMMIT=$(BUILD_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.

publish-image:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
