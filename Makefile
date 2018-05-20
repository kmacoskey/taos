# VERSION_TAG:=$(shell git describe --abbrev=0 --tags || echo "0.1")
VERSION_TAG:="1.0.0"
LDFLAGS:=-ldflags "-X github.com/kmacoskey/taos/app.Version=${VERSION_TAG}"

.PHONY: test clean

default: build

test:
	ginkgo -slowSpecThreshold 60 daos services terraform reaper handlers .

run:
	go run ${LDFLAGS} taos.go

build: clean
	go build ${LDFLAGS} -o taos *.go

debug: clean
	go build -gcflags "-N -l" -o taos *.go

clean:
	rm -rf taos coverage.out coverage-all.out
