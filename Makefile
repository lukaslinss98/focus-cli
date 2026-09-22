BINARY := focus
BUILD_DIR := bin
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: build install test clean

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/focus

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(INSTALL_DIR)/$(BINARY)

test:
	go test ./...
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
