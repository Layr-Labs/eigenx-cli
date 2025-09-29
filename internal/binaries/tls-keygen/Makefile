.PHONY: build clean

build:
	@echo "Building tls-keygen..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-s -w -extldflags '-static'" \
		-trimpath \
		-o ../../../tools/tls-keygen-linux-amd64 \
		./cmd/tls-keygen
	@echo "Built tls-keygen-linux-amd64"
	@if command -v upx >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		upx --best --lzma ../../../tools/tls-keygen-linux-amd64 >/dev/null 2>&1; \
		echo "Compressed successfully"; \
	else \
		echo "UPX not found, skipping compression"; \
	fi

clean:
	@rm -f ../../../tools/tls-keygen-linux-amd64