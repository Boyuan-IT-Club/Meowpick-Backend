.PHONY: license wire swagger

license:
	@ROOT=$$(git rev-parse --show-toplevel); \
	echo "Generating LICENSE..."; \
	cd $$ROOT && addlicense -c "Boyuan-IT-Club" -l apache .

wire:
	@echo "Running wire code generation..."
	wire gen ./provider

swagger:
	@echo "Generating swagger..."
	swag init \
		--parseDependency \
		--parseInternal \
		--output docs

