SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help

IMAGE ?= shynome/yu
WITH_FRONTEND ?= 1
TAG_LATEST ?= 1
GOOS ?= linux
GOARCH ?= $(shell go env GOARCH)
DIST := dist

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION := $(subst /,-,$(VERSION))
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X remoon.net/yu/backend/internal/version.Version=$(VERSION) \
	-X remoon.net/yu/backend/internal/version.Commit=$(COMMIT) \
	-X remoon.net/yu/backend/internal/version.BuildTime=$(BUILD_TIME)

INCLUDE_FRONTEND := $(if $(filter 1 true yes on,$(WITH_FRONTEND)),true,false)

ifeq ($(INCLUDE_FRONTEND),true)
IMAGE_TAG := $(VERSION)
DOCKER_TAGS := -t $(IMAGE):$(VERSION)
ifeq ($(TAG_LATEST),1)
DOCKER_TAGS += -t $(IMAGE):latest
endif
else
IMAGE_TAG := $(VERSION)-api
DOCKER_TAGS := -t $(IMAGE):$(VERSION)-api
endif

.PHONY: help all build frontend backend static docker docker-api push push-api check clean

help:
	@echo "Yu 构建与镜像"
	@echo ""
	@echo "  make docker          含前端静态文件，生成 $(IMAGE):$(IMAGE_TAG)（默认还打 latest）"
	@echo "  make docker-api      仅后端 API，生成 $(IMAGE):$(VERSION)-api"
	@echo "  make push            构建并推送含前端镜像"
	@echo "  make push-api        构建并推送仅后端镜像"
	@echo "  make build           只生成本地产物：$(DIST)/yu-server 与 $(DIST)/public"
	@echo "  make check           gofmt / go build / go test / 前端检查"
	@echo "  make clean           清理 $(DIST)/"
	@echo ""
	@echo "变量: IMAGE=$(IMAGE) WITH_FRONTEND=$(WITH_FRONTEND) TAG_LATEST=$(TAG_LATEST) GOARCH=$(GOARCH)"
	@echo "版本: $(VERSION) ($(COMMIT))"

all: docker

frontend:
	@echo "==> 构建前端"
	cd frontend && { [ -d node_modules ] && [ ! package-lock.json -nt node_modules ] || npm ci; }
	cd frontend && npm run build

backend:
	@echo "==> 构建后端 $(VERSION) ($(COMMIT))"
	mkdir -p $(DIST)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/yu-server ./backend/cmd/server

ifeq ($(INCLUDE_FRONTEND),true)
static: frontend
else
static:
endif
	@echo "==> 准备静态资源"
	rm -rf $(DIST)/public
	mkdir -p $(DIST)/public
ifeq ($(INCLUDE_FRONTEND),true)
	cp -a frontend/build/. $(DIST)/public/
endif

build: backend static

docker: build
	@echo "==> 构建镜像 $(IMAGE):$(IMAGE_TAG)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg WITH_FRONTEND=$(INCLUDE_FRONTEND) \
		$(DOCKER_TAGS) .
	@echo "完成: $(IMAGE):$(IMAGE_TAG)"

docker-api:
	$(MAKE) docker WITH_FRONTEND=0

push: docker
	docker push $(IMAGE):$(IMAGE_TAG)
ifeq ($(INCLUDE_FRONTEND),true)
ifeq ($(TAG_LATEST),1)
	docker push $(IMAGE):latest
endif
endif

push-api:
	$(MAKE) push WITH_FRONTEND=0

check:
	@files="$$(gofmt -l backend)"; \
	if [ -n "$$files" ]; then echo "需要 gofmt: $$files" >&2; exit 1; fi
	go build ./...
	go test ./...
	cd frontend && npm run check

clean:
	rm -rf $(DIST)
