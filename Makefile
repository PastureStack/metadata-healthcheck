.PHONY: build test validate package integration

build:
	./scripts/build

test:
	./scripts/test

validate:
	./scripts/validate

package: build
	./scripts/package

integration:
	./scripts/integration
