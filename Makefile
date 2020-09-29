ENV_COMMON_VAR=GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(subst x86_64,amd64,$(patsubst i%86,386,$(shell uname -m)))
ENV_BUILD_VAR=CGO_ENABLED=0

LOCAL_REGISTRY=localhost:5000
LOCAL_IMAGE=kube-scheduler:latest

RELEASE_REGISTRY?=localhost
RELEASE_VERSION?=$(shell git describe --tags --match "v*")
RELEASE_IMAGE:=kube-scheduler:$(RELEASE_VERSION)

.PHONY: all
all: clean build local-image delete-deployment create-deployment

.PHONY: build
build: build-scheduler

.PHONY: build-scheduler
build-scheduler:
	$(ENV_COMMON_VAR) $(ENV_BUILD_VAR) go build -ldflags '-w' -o bin/kube-scheduler cmd/scheduler/main.go

.PHONY: local-image
local-image:
	docker build --no-cache -f ./build/scheduler/Dockerfile -t $(LOCAL_IMAGE) .

.PHONY: push-local-image
push-local-image: local-image
	docker push $(LOCAL_REGISTRY)/$(LOCAL_IMAGE)

.PHONY: release-image
release-image:
	docker build -f ./build/scheduler/Dockerfile -t $(RELEASE_REGISTRY)/$(RELEASE_IMAGE) .

.PHONY: push-release-image
push-release-image: release-image
	docker push $(RELEASE_REGISTRY)/$(RELEASE_IMAGE)

.PHONY: create-deployment
create-deployment:
	kubectl create -f ./manifests/kube-scheduler.yaml || exit 0

.PHONY: delete-deployment
delete-deployment:
	kubectl -n kube-system delete deployment limit-await-scheduler || exit 0

.PHONY: clean
clean:
	rm -rf ./bin
