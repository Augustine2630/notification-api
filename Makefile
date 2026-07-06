.PHONY: build docker release build-arm docker-arm release-arm

IMAGE = images.registry.twcstorage.ru/util/notification-api

_build:
	GOOS=linux GOARCH=$(GOARCH) go build -o app cmd/main.go

build: GOARCH=amd64
build: _build

build-arm: GOARCH=arm64
build-arm: _build

docker: build
	docker buildx build --platform linux/amd64 -t $(IMAGE):latest --push .

docker-arm: build-arm
	docker build -t $(IMAGE):latest --push .

release: docker-arm
	kubectl rollout restart deployment -n util notification-api
