BINARY_NAME=compose-diff
VERSION?=1.0.0
BUILD_DIR=dist
LDFLAGS=-ldflags "-s -w -X github.com/stackgen-cli/compose-diff/cmd.version=$(VERSION)"

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build clean test deps release install uninstall build-all test-coverage

all: clean deps test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

build-all:
	@mkdir -p $(BUILD_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} \
		$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}$$(if [ "$${platform%/*}" = "windows" ]; then echo ".exe"; fi) . ; \
		echo "Built $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}"; \
	done

test:
	$(GOTEST) -v ./...

test-coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

deps:
	$(GOMOD) download
	$(GOMOD) tidy

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

release: clean deps test build-all
	@mkdir -p $(BUILD_DIR)/release
	@for platform in $(PLATFORMS); do \
		archive_name=$(BINARY_NAME)-$(VERSION)-$${platform%/*}-$${platform#*/}; \
		if [ "$${platform%/*}" = "windows" ]; then \
			zip -j $(BUILD_DIR)/release/$$archive_name.zip $(BUILD_DIR)/$(BINARY_NAME)-$${platform%/*}-$${platform#*/}.exe; \
		else \
			tar -czvf $(BUILD_DIR)/release/$$archive_name.tar.gz -C $(BUILD_DIR) $(BINARY_NAME)-$${platform%/*}-$${platform#*/}; \
		fi; \
	done
	@echo "Release archives created in $(BUILD_DIR)/release/"

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

uninstall:
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from /usr/local/bin/"
