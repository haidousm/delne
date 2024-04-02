REGISTRY ?= docker.io
REPOSITORY ?= haidousm
IMAGE_NAME ?= delne
VERSION ?= development-amd64
PLATFORM ?= linux/amd64

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

## dev/web: run the cmd/web application in dev mode (with air)
.PHONY: dev/web
dev/web:
	air -c .air.toml

## build/web: build the cmd/web application
.PHONY: build/web
build/web:
	@echo 'Building cmd/web...'
	go build -ldflags='-s -w' -o=./bin/web ./cmd/web

## docker/build: build the docker image (amd64)
.PHONY: docker/build
docker/build:
	@echo 'Building docker image...'
	docker build -t ${REGISTRY}/${REPOSITORY}/${IMAGE_NAME}:${VERSION} --platform ${PLATFORM}  -f ./Dockerfile .

## docker/push: push the docker image to the registry
.PHONY: docker/push
docker/push:
	@echo 'Pushing docker image...'
	docker push ${REGISTRY}/${REPOSITORY}/${IMAGE_NAME}:${VERSION}

## docker/release: build and push the docker image
.PHONY: docker/release
docker/release: docker/build docker/push

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck ./...
	@echo 'Running tests...'
	go test -race -vet=off ./...

## vendor: tidy and vendor dependencies
# .PHONY: vendor
# vendor:
# 	@echo 'Tidying and verifying module dependencies...'
# 	go mod tidy
# 	go mod verify
# 	@echo 'Vendoring dependencies...'
# 	go mod vendor

.PHONY: migration/up
migration/up:
	@echo 'Running migrations...'
	migrate -path ./migrations -database ${DATABASE_URL} up

.PHONY: migration/down
migration/down:
	@echo 'Rolling back migrations...'
	migrate -path ./migrations -database ${DATABASE_URL} down

.PHONY: migration/new
migration/new:
	@echo 'Creating new migration...'
	migrate create -ext sql -dir ./migrations -seq ${NAME}