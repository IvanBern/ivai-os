# Variables
BINARY_NAME=ivai-os-linux
CLI_NAME=ivaictl
MAIN_PATH=./cmd/ivai/
CLI_PATH=./cmd/ivaictl/
VM_TARGET=ivai-os-linux@orb
BIN_DEST=/usr/local/bin/ivai-os
TEST_RESULTS=test-results
GOPATH=$(shell go env GOPATH)
PATH:=$(GOPATH)/bin:$(PATH)

# Build-time version injection (git describe, commit hash, UTC date).
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)

.PHONY: tidy build build-cli build-cli-linux deploy service run clean dev install-cli test test-reports deploy-secrets install-test-tools stop

tidy:
	go mod tidy

# 1. Compile for the Debian VM
build:
	@echo "🔨 Cross-compiling for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PATH)

# 1.5 Compile the Mac CLI
build-cli:
	@echo "🍎 Building macOS CLI..."
	go build -ldflags="$(LDFLAGS)" -o $(CLI_NAME) $(CLI_PATH)

# 1.6 Compile the CLI for Linux ARM64
build-cli-linux:
	@echo "🐧 Building Linux ARM64 CLI..."
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(CLI_NAME)-linux $(CLI_PATH)

install-cli: build-cli
	@echo "🚚 Installing ivaictl to /usr/local/bin..."
	sudo mv $(CLI_NAME) /usr/local/bin/$(CLI_NAME)
	@echo "✅ ivaictl installed!"

# 1.7 Testing
test:
	@echo "🧪 Running all tests..."
	go test -v ./...

install-test-tools:
	@echo "🛠 Installing test reporting tools..."
	go install github.com/jstemmer/go-junit-report/v2@latest
	go install github.com/jandelgado/gcov2lcov@latest
	go install github.com/vakenbolt/go-test-report@latest

test-reports:
	@echo "📊 Generating test reports..."
	@mkdir -p $(TEST_RESULTS)
	# 1. Run tests and capture JSON for go-test-report, tee for log
	go test -v -coverprofile=$(TEST_RESULTS)/coverage.out -json ./... | tee $(TEST_RESULTS)/test.json
	# 2. Generate JUnit XML for CI Dashboards (Azure/GitHub)
	@if command -v go-junit-report > /dev/null; then \
		cat $(TEST_RESULTS)/test.json | go-junit-report -parser gojson > $(TEST_RESULTS)/junit.xml; \
	else \
		echo "⚠️ go-junit-report not found. Run 'make install-test-tools'."; \
	fi
	# 3. Generate Professional HTML Test Report
	@if command -v go-test-report > /dev/null; then \
		cat $(TEST_RESULTS)/test.json | go-test-report -o $(TEST_RESULTS)/report.html; \
	else \
		echo "⚠️ go-test-report not found. Run 'make install-test-tools'."; \
	fi
	# 4. Generate HTML coverage report
	go tool cover -html=$(TEST_RESULTS)/coverage.out -o $(TEST_RESULTS)/coverage.html
	# Generate LCOV
	@if command -v gcov2lcov > /dev/null; then \
		gcov2lcov -infile $(TEST_RESULTS)/coverage.out -outfile $(TEST_RESULTS)/coverage.lcov; \
	else \
		echo "⚠️ gcov2lcov not found, skipping LCOV generation. Run 'make install-test-tools'."; \
	fi
	@echo "✅ Reports generated in $(TEST_RESULTS)/"

# 2. Build, push, and set permissions in one shot
deploy: build build-cli-linux
	@echo "🚀 Shipping binaries to Debian VM..."
	scp $(BINARY_NAME) $(CLI_NAME)-linux $(VM_TARGET):~/
	@echo "🔒 Securing binaries..."
	ssh $(VM_TARGET) "sudo mv ~/$(BINARY_NAME) $(BIN_DEST) && sudo mv ~/$(CLI_NAME)-linux /usr/local/bin/$(CLI_NAME) && sudo chown ivai:ivai $(BIN_DEST) /usr/local/bin/$(CLI_NAME) && sudo chmod +x $(BIN_DEST) /usr/local/bin/$(CLI_NAME)"
	@echo "✅ Deployment complete!"

# 2.1 Deploy environment secrets
deploy-secrets:
	@echo "🔐 Shipping secrets to Debian VM..."
	scp .env $(VM_TARGET):~/
	ssh $(VM_TARGET) "sudo mkdir -p /etc/ivai && sudo mv ~/.env /etc/ivai/.env && sudo chown ivai:ivai /etc/ivai/.env && sudo chmod 600 /etc/ivai/.env"
	@echo "✅ Secrets deployed!"

# 2.5 Deploy systemd service
service: deploy
	@echo "📝 Deploying systemd service..."
	scp ivai.service $(VM_TARGET):~/
	ssh $(VM_TARGET) "sudo mv ~/ivai.service /etc/systemd/system/ivai.service && sudo systemctl daemon-reload && sudo systemctl enable ivai && sudo systemctl restart ivai"
	@echo "✅ Service deployed and started!"

# 3. Run the OS securely inside the VM (Foreground/Interactive)
run:
	@echo "🧠 Waking up Ivai OS..."
	ssh -t $(VM_TARGET) "sudo -u ivai $(BIN_DEST)"

# 4. Do it all (Build -> Deploy -> Run)
dev: deploy run

# 5. Run locally on macOS
run-local:
	@echo "🍎 Waking up Ivai OS locally..."
	go run ./cmd/ivai/

clean:
	rm -f $(BINARY_NAME) $(CLI_NAME)
	rm -rf $(TEST_RESULTS)

stop:
	@echo "🛑 Stopping local Ivai OS..."
	@pkill -f "cmd/ivai" 2>/dev/null || true
	@lsof -ti:8080 -ti:8081 2>/dev/null | xargs kill 2>/dev/null || true
	@echo "✅ Ivai OS stopped."

.PHONY: reset
reset:
	@echo "🧹 Resetting Ivai OS memory..."
	@ssh $(VM_TARGET) "sudo systemctl stop ivai && sudo rm -f /etc/ivai/memory.db && sudo systemctl start ivai"
	@echo "✅ Memory reset. Ivai has a clean slate."

.PHONY: tag
tag:
	@echo "🏷️  Tagging release..."
	@TAG="v$$(date +%Y.%m.%d)-$$(git rev-parse --short HEAD)" && \
	 git tag -a "$$TAG" -m "Release $$TAG" && \
	 git push origin "$$TAG" && \
	 echo "✅ Tagged: $$TAG"

.PHONY: deploy-staging
deploy-staging: build
	@echo "🧪 Deploying to staging VM (ai-server)..."
	@scp $(BINARY_NAME) ivai-os-linux@ai-server:~/
	@ssh ivai-os-linux@ai-server "sudo mv ~/$(BINARY_NAME) /usr/local/bin/ivai-os-staging && sudo chmod +x /usr/local/bin/ivai-os-staging && sudo pkill -f ivai-os-staging 2>/dev/null; IVAI_PORT=8099 /usr/local/bin/ivai-os-staging &"
	@echo "✅ Staging deployed on ai-server:8099"
