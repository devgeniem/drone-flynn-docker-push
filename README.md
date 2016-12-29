# drone-flynn-docker-push

[![Build Status](https://travis-ci.org/devgeniem/drone-flynn-docker-push.svg?branch=master)](https://travis-ci.org/devgeniem/drone-flynn-docker-push)

Drone plugin which can be used to build and publish Docker images to a [Flynn](https://flynn.io) cluster.
For the usage information and a listing of the available options
please take a look at [the docs](DOCS.md).

## Build

Build the binary with the following commands:

```
go build
go test
```

## Docker

Build the docker image with the following commands:

```
make docker
```

Please note incorrectly building the image for the correct x64 linux and with
GCO disabled will result in an error when running the Docker image:

```
docker: Error response from daemon: Container command
'/bin/drone-flynn-docker-push' not found or does not exist..
```

## Usage

Execute this plugin from the working directory:

```
docker run --rm \
  -e PLUGIN_TAG=latest \
  -e PLUGIN_REPO=devgeniem/wp-project \
  -e FLYNN_TLSPIN='XXXXXXXXXXXXXXXXXXXXXXX' \
  -e FLYNN_KEY='YYYYYYYYYYYYYYYYYYYYYYYY' \
  -e FLYNN_DOMAIN='flynn.example.com' \
  -e FLYNN_APP='my-application' \
  -e DRONE_COMMIT_SHA=d8dbe4d94f15fe89232e0402c6e8a0ddf21af3ab \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  --privileged \
  devgeniem/drone-flynn-docker-push --dry-run
```
