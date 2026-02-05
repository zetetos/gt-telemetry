
# Change these variables as necessary.
MAIN_PACKAGE_PATH := ./cmd/demo
BINARY_PATH := ./bin
BINARY_NAME := demo


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
	go vet ./pkg/... ./internal/reader ./internal/salsa20 ./internal/units # ignore generated Kaitai Struct files as they trip some rules
	@echo DISABLED: go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	go test -race -buildvcs -vet=off ./...

## lint: run linters
.PHONY: lint
lint:
	golangci-lint run --max-issues-per-linter 0 --max-same-issues 0

## lint/fix: run linter against the project and fix issues where possible
.PHONY: lint/fix
lint/fix:
	golangci-lint run --fix


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

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	@go run github.com/oligot/go-mod-upgrade@latest

## build/kaitai: compile the GT telemetry package from the Kaitai Struct
.PHONY: build/kaitai
build/kaitai:
	@docker build --output=internal/telemetry --progress=plain -f build/Dockerfile .

## build: build the demo application for the local platform
.PHONY: build
build:
	@go build -o bin/${BINARY_NAME} ${MAIN_PACKAGE_PATH}

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
	@GOOS=windows GOARCH=amd64 go build -o ${BINARY_PATH}/${BINARY_NAME}-amd64.exe ${MAIN_PACKAGE_PATH}

## run: run the  application
.PHONY: run
run: build
	@./bin/${BINARY_NAME}

## run/watch: run the application locally and reload on file changes
.PHONY: run/watch
run/watch:
	go run github.com/air-verse/air@latest \
		--build.cmd "make build" --build.bin "${BINARY_PATH}/${BINARY_NAME}" --build.delay "100" \
		--build.include_ext "go" \
		--build.send_interrupt "true" \
		--misc.clean_on_exit "true"


## run/capture-replay: capture a replay and save to gt7-replay.gtz
.PHONY: run/capture-replay
run/capture-replay:
	@go run cmd/capture_replay/main.go
	@echo "Replay saved to gt7-replay.gtz"

## update/vehicledb: update the vehicle inventory JSON file from GT7 website
.PHONY: update/vehicledb
update/vehicledb:
	@go run tools/vehicle_inventory/*.go update pkg/vehicles/vehicles.json

## update/circuitdb: update the circuit inventory JSON file from saved circuit data
.PHONY: update/circuitdb
update/circuitdb:
	@ go run tools/circuit_inventory/main.go data/circuits pkg/circuits/circuits.json

## clean: clean up project and return to a pristine state
.PHONY: clean
clean:
	@go clean
	@rm -rf ./bin
	@rm -f coverage.out