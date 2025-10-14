# include make-*.mk

# Makes it possible to run "make instll user/repo" instead of "make instll ARGS=user/repo"
ARGS = $(filter-out $@,$(MAKECMDGOALS))

.PHONY: push build-docker bump-version test release instll instll


instll:
	curl -fsSL instll.sh/$(ARGS) | bash

build:
	go build -o para main.go

docker-build:
	docker run --rm -v $(PWD):/app -w /app golang:1.24-alpine sh -c "./scripts/build.sh"

push!: build

bump-version:
	@if [ -z "$(v)" ]; then \
		echo "Usage: make bump-version v=x.y.z"; \
		exit 1; \
	fi; \
	sed -i 's/Version[ ]*=[ ]*".*"/Version = "$(v)"/' main.go; \
	go fmt main.go; \
	git add main.go; \
	git commit -m "bump version to $(v)"; \
	git tag $(v); \
	echo "Version updated to $(v) and git tag created."

push-version:
	git push
	git push --tags

test:
	@echo "Running tests..."
	go test -v

release:
	@if [ -z "$(v)" ]; then \
		echo "Usage: make release v=x.y.z"; \
		echo "Example: make release v=0.6.6"; \
		exit 1; \
	fi; \
	echo "🧪 Running tests..."; \
	if ! go test -v; then \
		echo "❌ Tests failed! Release aborted."; \
		exit 1; \
	fi; \
	echo "✅ Tests passed!"; \
	echo "📝 Bumping version to $(v)..."; \
	sed -i 's/Version[ ]*=[ ]*".*"/Version = "$(v)"/' main.go; \
	go fmt main.go; \
	echo "📦 Creating commit and tag..."; \
	git add main.go; \
	git commit -m "🚀 Release $(v)"; \
	git tag $(v); \
	echo "🚀 Pushing to repository..."; \
	git push origin main --tags; \
	echo "✨ Release $(v) completed successfully!"

show-versions:
	@git tag -l

install:
	scripts/install.sh

dev-install:
	@echo "Detecting platform and architecture..."
	@PLATFORM=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	ARCH=$$(uname -m); \
	if [ "$$ARCH" = "x86_64" ]; then \
		ARCH="amd64"; \
	elif [ "$$ARCH" = "aarch64" ]; then \
		ARCH="arm64"; \
	fi; \
	BINARY="para-$$PLATFORM-$$ARCH"; \
	echo "Installing $$BINARY as para-dev..."; \
	sudo ln -sf $(PWD)/dist/$$BINARY /usr/local/bin/para-dev; \
	echo "Development version installed successfully!"

dev-uninstall:
	sudo rm -f /usr/local/bin/para-dev

dev-install!: dev-uninstall dev-install

uninstall:
	scripts/uninstall.sh

reinstall: uninstall install

.DEFAULT_GOAL := instll

# Catch-all rule to prevent Make from trying to find targets with argument names
%:
	@:

