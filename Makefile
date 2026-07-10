.PHONY: build docker release

IMAGE = images.registry.twcstorage.ru/util/notification-api

build:
	GOOS=linux GOARCH=arm64 go build -o app cmd/main.go

docker: build
	docker build --platform linux/arm64 -t $(IMAGE):latest --push .

release: docker
	kubectl rollout restart deployment -n util notification-api
