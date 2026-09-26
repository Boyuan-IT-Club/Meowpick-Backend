.PHONY: license license-check wire swagger meowpick-run meowpick-clean

license:
	@ROOT=$$(git rev-parse --show-toplevel); \
	echo "Generating LICENSE..."; \
	cd $$ROOT && addlicense -c "Boyuan-IT-Club" -l apache .

license-check:
	@test -s LICENSE
	@grep -q 'Apache License' LICENSE
	@grep -q 'Version 2.0' LICENSE
	@git ls-files -z | xargs -0 addlicense -check -c "Boyuan-IT-Club" -l apache

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
		--outputTypes go \
		--output docs
	perl -0pi -e 's/"bearerauth":/"Bearer":/g; s/"security": \[\s*\{\s*"": \[\]\s*\}\s*\]/"security": []/g' docs/docs.go
