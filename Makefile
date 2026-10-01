.PHONY: generate openapi-bundle openapi-bundle-check check test build ci openapi-view openapi-stop

generate openapi-bundle openapi-bundle-check check test build ci openapi-view openapi-stop:
	$(MAKE) -C cicd $@
