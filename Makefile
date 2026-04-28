# Variables
BINARY_NAME=ivai-os-linux
CLI_NAME=ivaictl
MAIN_PATH=cmd/ivai/main.go
CLI_PATH=cmd/ivaictl/main.go
VM_TARGET=ivai-os-linux@orb
BIN_DEST=/usr/local/bin/ivai-os
TEST_RESULTS=test-results
GOPATH=$(shell go env GOPATH)
PATH:=$(GOPATH)/bin:$(PATH)

.PHONY: tidy build build-cli deploy service run clean dev install-cli test test-reports deploy-secrets install-test-tools

tidy:
	go mod tidy

# 1. Compile for the Debian VM
build:
	@echo "🔨 Cross-compiling for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME) $(MAIN_PATH)

# 1.5 Compile the Mac CLI
build-cli:
	@echo "🍎 Building macOS CLI..."
	go build -o $(CLI_NAME) $(CLI_PATH)

install-cli: build-cli
	@echo "🚚 Installing ivaictl to /usr/local/bin..."
	sudo mv $(CLI_NAME) /usr/local/bin/$(CLI_NAME)
	@echo "✅ ivaictl installed!"

# 1.6 Testing
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
deploy: build
	@echo "🚀 Shipping binary to Debian VM..."
	scp $(BINARY_NAME) $(VM_TARGET):~/
	@echo "🔒 Securing binary..."
	ssh $(VM_TARGET) "sudo mv ~/$(BINARY_NAME) $(BIN_DEST) && sudo chown ivai:ivai $(BIN_DEST) && sudo chmod +x $(BIN_DEST)"
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
	go run cmd/ivai/main.go

clean:
	rm -f $(BINARY_NAME) $(CLI_NAME)
	rm -rf $(TEST_RESULTS)
