.PHONY: docker

EXECUTABLE ?= drone-flynn-docker-push
IMAGE ?= devgeniem/$(EXECUTABLE)
docker:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-s -w'
	docker build --rm -t $(IMAGE) .
