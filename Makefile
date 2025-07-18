
# Change these variables as necessary.
MAIN_PACKAGE_PATH := ./examples/simple
BINARY_PATH := ./examples/bin
BINARY_NAME := gt-telemetry


# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	git diff --exit-code


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go mod verify
	go vet ./ ./internal/utils   # ignore Kaitai Struct files as they trip some rules
	@echo DISABLED: go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...

## lint: run linters
.PHONY: lint
lint:
	golangci-lint run


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=coverage.out ./...
	@grep -v "gran_turismo_telemetry.go:" coverage.out > trimmed.out && mv trimmed.out coverage.out

## test/cover/show: run all tests and display coverage in a browser
.PHONY: test/cover/show
test/cover/show: test/cover
	go tool cover -html=coverage.out

## build/kaitai: compile the GT telemetry package from the Kaitai Struct
.PHONY: build/kaitai
build/kaitai:
	@docker build --output=internal/gttelemetry --progress=plain -f build/Dockerfile .

## build: build the application for the local platform
.PHONY: build
build:
	@go build -o examples/bin/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

## build/darwin/silicon: build the application for Apple Silicon
.PHONY: build/darwin/silicon
build/darwin/silicon:
	@GOOS=darwin GOARCH=arm64 go build -o ${BINARY_PATH}/${BINARY_NAME}-darwin-arm64 ${MAIN_PACKAGE_PATH}

## build/linux: build the application for Linux amd64
.PHONY: build/linux
build/linux:
	@GOOS=linux GOARCH=amd64 go build -o ${BINARY_PATH}/${BINARY_NAME}-linux-amd64 ${MAIN_PACKAGE_PATH}

## build/rpi/v6: build the application for Raspberry Pi ARMv6
.PHONY: build/rpi/v6
build/rpi/v6:
	@GOOS=linux GOARCH=arm GOARM=6 go build -o ${BINARY_PATH}/${BINARY_NAME}-rpi-armv6 ${MAIN_PACKAGE_PATH}

## build: build the application for Raspberry Pi ARMv7
.PHONY: build/rpi/v7
build/rpi/v7:
	@GOOS=linux GOARCH=arm GOARM=7 go build -o ${BINARY_PATH}/${BINARY_NAME}-rpi-armv6 ${MAIN_PACKAGE_PATH}

## build/rpi/v8: build the application for Raspberry Pi ARMv8
.PHONY: build/rpi/v8
build/rpi/armv8:
	@GOOS=linux GOARCH=arm64 GOARM=8 go build -o ${BINARY_PATH}/${BINARY_NAME}-rpi-arm64 ${MAIN_PACKAGE_PATH}

## build/windows: build the application for Windows amd64
.PHONY: build/windows
build/windows:
	@GOOS=windows GOARCH=amd64 go build -o examples/bin/${BINARY_NAME}-amd64.exe ${MAIN_PACKAGE_PATH}

## run: run the  application
.PHONY: run
run: build
	@./examples/bin/${BINARY_NAME}

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live:
	@go run ${MAIN_PACKAGE_PATH}/main.go

## run/capture-replay: capture a replay and save to gt7-replay.gtz
.PHONY: run/capture-replay
run/capture-replay:
	@go run cmd/capture_replay/main.go
	@echo "Replay saved to gt7-replay.gtz"

## clean: clean up project and return to a pristine state
.PHONY: clean
clean:
	@go clean
	@rm -rf examples/bin
	@rm -f coverage.out