APP_NAME := openlist-sync
CMD_PATH := ./cmd/openlist-sync
BIN_DIR := bin
DIST_DIR := dist

.PHONY: help build clean cross darwin-arm64 linux-amd64 linux-arm64

help:
	@echo "Targets:"
	@echo "  make build         Build local binary to $(BIN_DIR)/$(APP_NAME)"
	@echo "  make cross         Cross-build: darwin/arm64, linux/amd64, linux/arm64"
	@echo "  make clean         Remove build artifacts"

build:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 go build -trimpath -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)

cross: darwin-arm64 linux-amd64 linux-arm64

darwin-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)

linux-amd64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)

linux-arm64:
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
