# VERSION_TAG:=$(shell git describe --abbrev=0 --tags || echo "0.1")
# VERSION:=${VERSION}\#$(shell git log -n 1 --pretty=format:"%h")
# PACKAGES:=$(shell go list ./... | sed -n '1!p' | grep -v /vendor/ | sed 's!.*/!!')
LDFLAGS:=-ldflags "-X github.com/kmacoskey/taos/app.Version=${VERSION}"

.PHONY: test clean

default: build

depends:
	../../../../bin/glide up

test:
	ginkgo -r

cover: test
	go tool cover -html=coverage-all.out

run:
	go run ${LDFLAGS} taos.go

build: clean
	go build ${LDFLAGS} -o taos *.go

debug: clean
	go build -gcflags "-N -l" -o taos *.go

clean:
	rm -rf taos coverage.out coverage-all.out
