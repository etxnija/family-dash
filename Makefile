# Build, deploy and run family-dash on the Raspberry Pi Zero W.
# `make help` lists the targets. Everything runs from your Mac over SSH.
HOST      ?= familydash
# Install directory on the Pi, relative to the login user's home.
DIR       ?= family-dash
CONFIG    ?= sources.local.json
BIN       := family-dashboard-pi
SERVICE   := family-dashboard
SSH       := ssh $(HOST)

.DEFAULT_GOAL := help
# deploy runs its steps in order; never run them in parallel.
.NOTPARALLEL:
.PHONY: help test lint build-pi push-binary push-config install-service deploy deploy-config \
        start stop restart status logs logs-last doctor

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-16s %s\n", $$1, $$2}'

test: ## Run the tests (sensors/test.sh)
	sh sensors/test.sh

lint: ## Run the linter (sensors/lint.sh)
	sh sensors/lint.sh

build-pi: ## Cross-compile the Pi binary (ARMv6, no cgo)
	GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o $(BIN) ./cmd/family-dashboard
	@ls -l $(BIN)

push-binary: build-pi ## Copy the binary to the Pi (atomic rename; the old one keeps running until restart)
	$(SSH) 'mkdir -p $(DIR) && umask 022 && cat > $(DIR)/$(BIN).new && chmod 755 $(DIR)/$(BIN).new && mv -f $(DIR)/$(BIN).new $(DIR)/$(BIN)' < $(BIN)

push-config: ## Copy the config file to the Pi (mode 600; it holds private calendar URLs)
	$(SSH) 'mkdir -p $(DIR) && umask 077 && cat > $(DIR)/$(CONFIG).new && mv -f $(DIR)/$(CONFIG).new $(DIR)/$(CONFIG)' < $(CONFIG)

install-service: ## Install and enable the systemd unit (needs sudo on the Pi)
	@user=$$($(SSH) id -un) && home=$$($(SSH) 'echo $$HOME') && \
	sed -e "s|@USER@|$$user|g" -e "s|@DIR@|$$home/$(DIR)|g" -e "s|@BIN@|$(BIN)|g" -e "s|@CONFIG@|$(CONFIG)|g" deploy/$(SERVICE).service.in | \
	$(SSH) 'sudo tee /etc/systemd/system/$(SERVICE).service >/dev/null && sudo systemctl daemon-reload && sudo systemctl enable $(SERVICE)'

deploy: test lint push-binary push-config install-service restart status ## Test, lint, build, copy binary + config, install the unit, restart

deploy-config: push-config restart status ## Copy only the config and restart (e.g. after a calendar link changed)

start: ## Start the service
	$(SSH) 'sudo systemctl start $(SERVICE)'

stop: ## Stop the service (the panel is put to sleep and keeps its image)
	$(SSH) 'sudo systemctl stop $(SERVICE)'

restart: ## Restart the service
	$(SSH) 'sudo systemctl restart $(SERVICE)'

status: ## Show the service status
	$(SSH) 'systemctl status $(SERVICE) --no-pager'

logs: ## Follow the service log (Ctrl-C to stop)
	ssh -t $(HOST) 'journalctl -u $(SERVICE) -f -o cat'

logs-last: ## Show the last 200 log lines
	$(SSH) 'journalctl -u $(SERVICE) -n 200 --no-pager -o cat'

doctor: ## Check SSH, passwordless sudo and the spi/gpio groups on the Pi
	$(SSH) 'echo "ssh ok: $$(id -un)@$$(hostname) $$(uname -m)"; sudo -n true && echo "sudo ok" || echo "sudo asks for a password: deploy will not work"; echo "groups: $$(id -nG)"'
