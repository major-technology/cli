.PHONY: rerelease release dev-install dev-install-local dev-restore sync-schemas

# Build and install locally for development
dev-install:
	@echo "Building CLI with prod config..."
	go build -ldflags "-X 'github.com/major-technology/cli/cmd.configFile=configs/prod.json'" -o major .
	@echo "Backing up current CLI..."
	@cp ~/.major/bin/major ~/.major/bin/major.backup 2>/dev/null || true
	@echo "Installing development build..."
	@cp major ~/.major/bin/major
	@rm major
	@codesign --force --sign - ~/.major/bin/major 2>/dev/null || true
	@mkdir -p ~/.major
	@printf "configs/prod.json" > ~/.major/env
	@echo "Done! Run 'major --version' to verify."

# Build and install locally with local.json config (for testing against local API)
dev-install-local:
	@echo "Building CLI with LOCAL config..."
	go build -ldflags "-X 'github.com/major-technology/cli/cmd.configFile=configs/local.json'" -o major .
	@echo "Backing up current CLI..."
	@cp ~/.major/bin/major ~/.major/bin/major.backup 2>/dev/null || true
	@echo "Installing development build..."
	@cp major ~/.major/bin/major
	@rm major
	@codesign --force --sign - ~/.major/bin/major 2>/dev/null || true
	@mkdir -p ~/.major
	@printf "configs/local.json" > ~/.major/env
	@echo "Done! major CLI now uses configs/local.json"

# Restore original CLI from backup
dev-restore:
	@echo "Restoring original CLI..."
	@cp ~/.major/bin/major.backup ~/.major/bin/major
	@echo "Done!"

rerelease:
	@echo "Releasing version v$(VERSION)..."
	git tag -d v$(VERSION)
	git push origin :refs/tags/v$(VERSION)
	git tag -a v$(VERSION) -m "v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Successfully released v$(VERSION)"

release:
	@echo "Releasing version v$(VERSION)..."
	git tag -a v$(VERSION) -m "v$(VERSION)"
	git push origin v$(VERSION)
	@echo "Successfully released v$(VERSION)"

# Vendor-sync projects/schemas/*.schema.json from the platform API. mono-builder
# is the source of truth: it generates these schemas from zod and serves them at
# GET <base>/schemas/project.json and GET <base>/schemas/agent.json. Refresh the
# vendored copy and its SCHEMAS.sha256 manifest after an upstream schema change.
# Override the source with MAJOR_SCHEMAS_BASE_URL, e.g. against a local dev API:
#   MAJOR_SCHEMAS_BASE_URL=http://localhost:3301 make sync-schemas
sync-schemas:
	@base="$${MAJOR_SCHEMAS_BASE_URL:-https://api.major.tech}"; \
	echo "Syncing schemas from $$base..."; \
	curl -fsSL "$$base/schemas/project.json" -o projects/schemas/project.schema.json; \
	curl -fsSL "$$base/schemas/agent.json" -o projects/schemas/agent.schema.json; \
	{ \
		echo "source: $$base"; \
		echo "fetched: $$(date -u +%Y-%m-%dT%H:%M:%SZ)"; \
		shasum -a 256 projects/schemas/project.schema.json projects/schemas/agent.schema.json; \
	} > projects/schemas/SCHEMAS.sha256; \
	echo "Done. Vendored schemas + projects/schemas/SCHEMAS.sha256 written."
