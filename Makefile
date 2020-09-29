ENV_COMMON_VAR=GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
ENV_BUILD_VAR=CGO_ENABLED=0

LOCAL_REGISTRY=localhost:5000/k8s-scheduler-plugins
LOCAL_IMAGE=kube-scheduler:latest

RELEASE_REGISTRY?=registry.management-dta.yolt.io
RELEASE_VERSION?=$(shell git describe --tags --match "v*")
RELEASE_IMAGE:=kube-scheduler:$(RELEASE_VERSION)

.PHONY: all
all: build

.PHONY: build
build: build-scheduler

.PHONY: build-scheduler
build-scheduler:
	$(ENV_COMMON_VAR) $(ENV_BUILD_VAR) go build -ldflags '-w' -o bin/kube-scheduler cmd/scheduler/main.go

.PHONY: local-image
local-image: clean
	docker build -f ./build/scheduler/Dockerfile -t $(LOCAL_REGISTRY)/$(LOCAL_IMAGE) .
	docker build -f ./build/controller/Dockerfile -t $(LOCAL_REGISTRY)/$(LOCAL_CONTROLLER_IMAGE) .

.PHONY: release-image
release-image: clean
	docker build -f ./build/scheduler/Dockerfile -t $(RELEASE_REGISTRY)/$(RELEASE_IMAGE) .
	docker build -f ./build/controller/Dockerfile -t $(RELEASE_REGISTRY)/$(RELEASE_CONTROLLER_IMAGE) .

.PHONY: push-release-image
push-release-image: release-image
	gcloud auth configure-docker
	docker push $(RELEASE_REGISTRY)/$(RELEASE_IMAGE)
	docker push $(RELEASE_REGISTRY)/$(RELEASE_CONTROLLER_IMAGE)

.PHONY: clean
clean:
	rm -rf ./bin
