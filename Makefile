.PHONY: rerelease release

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
