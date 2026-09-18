SHELL := /bin/sh

.PHONY: all build debug test test-tools vet vet-tools vet-windows check vulncheck clean

all: build

build: check
	@set -eu; \
	version="$$(go run ./cmd/resourcecheck -root .)"; \
	echo "Building BareMark version $$version"; \
	mkdir -p dist; \
	amd64_resource="cmd/baremark/resource_windows_amd64.syso"; \
	arm64_resource="cmd/baremark/resource_windows_arm64.syso"; \
	amd64_output="dist/baremark-$$version-amd64.exe"; \
	arm64_output="dist/baremark-$$version-arm64.exe"; \
	amd64_temp="dist/.baremark-$$version-amd64.$$$$.tmp"; \
	arm64_temp="dist/.baremark-$$version-arm64.$$$$.tmp"; \
	trap 'rm -f "$$amd64_resource" "$$arm64_resource" "$$amd64_temp" "$$arm64_temp"' EXIT; \
	go -C build/tools run -mod=readonly ./cmd/resourcegen -root ../.. -arch amd64 -version "$$version" -original-name "baremark-$$version-amd64.exe" -o "../../$$amd64_resource"; \
	go run ./cmd/artifactcheck -kind syso -arch amd64 -path "$$amd64_resource"; \
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags='-H=windowsgui -s -w' -o "$$amd64_temp" ./cmd/baremark; \
	go run ./cmd/artifactcheck -kind exe -arch amd64 -path "$$amd64_temp" -icons-root "assets/icons" -manifest "build/windows/baremark.manifest" -version "$$version" -original-name "baremark-$$version-amd64.exe" -publish "$$amd64_output"; \
	go -C build/tools run -mod=readonly ./cmd/resourcegen -root ../.. -arch arm64 -version "$$version" -original-name "baremark-$$version-arm64.exe" -o "../../$$arm64_resource"; \
	go run ./cmd/artifactcheck -kind syso -arch arm64 -path "$$arm64_resource"; \
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -ldflags='-H=windowsgui -s -w' -o "$$arm64_temp" ./cmd/baremark; \
	go run ./cmd/artifactcheck -kind exe -arch arm64 -path "$$arm64_temp" -icons-root "assets/icons" -manifest "build/windows/baremark.manifest" -version "$$version" -original-name "baremark-$$version-arm64.exe" -publish "$$arm64_output"

debug: check
	@set -eu; \
	version="$$(go run ./cmd/resourcecheck -root .)"; \
	echo "Building BareMark version $$version"; \
	mkdir -p dist; \
	resource="cmd/baremark/resource_windows_amd64.syso"; \
	output="dist/baremark-$$version-amd64-debug.exe"; \
	temporary="dist/.baremark-$$version-amd64-debug.$$$$.tmp"; \
	trap 'rm -f "$$resource" "$$temporary"' EXIT; \
	go -C build/tools run -mod=readonly ./cmd/resourcegen -root ../.. -arch amd64 -version "$$version" -original-name "baremark-$$version-amd64-debug.exe" -o "../../$$resource"; \
	go run ./cmd/artifactcheck -kind syso -arch amd64 -path "$$resource"; \
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -buildvcs=true -gcflags='all=-N -l' -ldflags='-H=windowsgui' -o "$$temporary" ./cmd/baremark; \
	go run ./cmd/artifactcheck -kind exe -arch amd64 -path "$$temporary" -icons-root "assets/icons" -manifest "build/windows/baremark.manifest" -version "$$version" -original-name "baremark-$$version-amd64-debug.exe" -publish "$$output"

test:
	go test ./...

test-tools:
	go -C build/tools test ./...

vet-tools:
	go -C build/tools vet -unsafeptr=false ./...

vet:
	go vet -unsafeptr=false ./...
	@echo "Passed go vet (-unsafeptr=false)"

vet-windows:
	GOOS=windows go vet -unsafeptr=false ./...
	@echo "Passed go vet (GOOS=windows, -unsafeptr=false)"

check: test-tools vet-tools test vet vet-windows

vulncheck:
	govulncheck ./...

clean:
	go clean
	rm -f cmd/baremark/resource_windows_amd64.syso cmd/baremark/resource_windows_arm64.syso
