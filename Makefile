# Variables
BINARY_NAME=ivai-os-linux
CLI_NAME=ivaictl
MAIN_PATH=cmd/ivai/main.go
CLI_PATH=cmd/ivaictl/main.go
VM_TARGET=ivai-os-linux@orb
BIN_DEST=/usr/local/bin/ivai-os

.PHONY: tidy build build-cli deploy service run clean dev install-cli

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

clean:
	rm -f $(BINARY_NAME)