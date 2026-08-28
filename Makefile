.PHONY: license wire swagger meowpick-run meowpick-clean

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
		--v3.1 \
		--parseDependency \
		--parseInternal \
		--packagePrefix github.com/Boyuan-IT-Club/Meowpick-Backend \
		--output docs
	perl -0pi -e 's/"bearerauth":/"Bearer":/g; s/\bbearerauth:/Bearer:/g; s/"security": \[\s*\{\s*"": \[\]\s*\}\s*\]/"security": []/g; s/security:\n([ \t]*)- "": \[\]/security: []/g' docs/docs.go docs/swagger.json docs/swagger.yaml
