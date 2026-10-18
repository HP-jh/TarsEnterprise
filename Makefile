MODULE := github.com/HP-jh/TarsEnterprise
VERSION := v0.5.0
BINARY := te
BUILD_FLAGS := -ldflags "-s -w -X $(MODULE)/pkg/config.Version=$(VERSION)"

.PHONY: all build test race clean release

all: test build

build:
	go build $(BUILD_FLAGS) -o bin/$(BINARY) ./cmd/te

test:
	go test -v ./...

race:
	go test -race -v ./...

clean:
	rm -rf bin/ dist/

release: clean
	mkdir -p dist
	# linux amd64
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-linux-amd64 ./cmd/te
	# linux arm64
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-linux-arm64 ./cmd/te
	# darwin amd64
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-darwin-amd64 ./cmd/te
	# darwin arm64
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-darwin-arm64 ./cmd/te
	# windows amd64
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-windows-amd64.exe ./cmd/te
	# windows arm64
	GOOS=windows GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-windows-arm64.exe ./cmd/te
	# android arm64
	GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o dist/$(BINARY)-$(VERSION)-android-arm64 ./cmd/te
	# SHA256SUMS
	cd dist && sha256sum * > SHA256SUMS.txt
	@echo "Release artifacts ready in dist/"
