.PHONY: generate check test build ci openapi-view openapi-stop

generate check test build ci openapi-view openapi-stop:
	$(MAKE) -C cicd $@
