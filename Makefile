VERSION_TAG:=$(shell git describe --abbrev=0 --tags || echo "0.1")
VERSION:=${VERSION}\#$(shell git log -n 1 --pretty=format:"%h")
PACKAGES:=$(shell go list ./... | sed -n '1!p' | grep -v /vendor/)
LDFLAGS:=-ldflags "-X github.com/kmacoskey/taos/app.Version=${VERSION}"

default: build

depends:
	../../../../bin/glide up

test:
	echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES), \
		go test -p=1 -cover -covermode=count -coverprofile=coverage.out ${pkg}; \
		tail -n +2 coverage.out >> coverage-all.out;)

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
