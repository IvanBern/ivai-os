# Variables
BINARY_NAME=ivai-os-linux
MAIN_PATH=cmd/ivai/main.go
VM_TARGET=ivai-os-linux@orb
BIN_DEST=/usr/local/bin/ivai-os

.PHONY: tidy build deploy run clean dev

tidy:
	go mod tidy

# 1. Compile for the Debian VM
build:
	@echo "🔨 Cross-compiling for Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_NAME) $(MAIN_PATH)

# 2. Build, push, and set permissions in one shot
deploy: build
	@echo "🚀 Shipping binary to Debian VM..."
	scp $(BINARY_NAME) $(VM_TARGET):~/
	@echo "🔒 Securing binary..."
	ssh $(VM_TARGET) "sudo mv ~/$(BINARY_NAME) $(BIN_DEST) && sudo chown ivai:ivai $(BIN_DEST) && sudo chmod +x $(BIN_DEST)"
	@echo "✅ Deployment complete!"

# 2.5 Deploy systemd service
service:
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