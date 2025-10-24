.PHONY: build clean

build:
	@echo "Checking for UPX..."
	@command -v upx >/dev/null 2>&1 || { \
		echo "Error: UPX is required but not installed."; \
		echo "Install with: brew install upx"; \
		exit 1; \
	}
	@echo "Building tls-keygen..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -extldflags '-static'" \
		-trimpath \
		-o ../../../tools/tls-keygen-linux-amd64 \
		./cmd/tls-keygen
	@echo "Built tls-keygen-linux-amd64"
	@echo "Compressing with UPX..."
	@upx --best --lzma ../../../tools/tls-keygen-linux-amd64 >/dev/null 2>&1
	@echo "Compressed successfully"

clean:
	@rm -f ../../../tools/tls-keygen-linux-amd64