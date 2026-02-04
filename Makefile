.PHONY: rerelease release dev-install dev-restore

# Build and install locally for development
dev-install:
	@echo "Building CLI with prod config..."
	go build -ldflags "-X 'github.com/major-technology/cli/cmd.configFile=configs/prod.json'" -o major .
	@echo "Backing up current CLI..."
	@cp ~/.major/bin/major ~/.major/bin/major.backup 2>/dev/null || true
	@echo "Installing development build..."
	@cp major ~/.major/bin/major
	@rm major
	@echo "Done! Run 'major --version' to verify."

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
