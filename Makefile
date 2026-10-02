.PHONY: generate openapi-bundle openapi-bundle-check check test build ci openapi-view openapi-stop hooks-install pre-push

generate openapi-bundle openapi-bundle-check check test build ci openapi-view openapi-stop:
	$(MAKE) -C cicd $@

hooks-install:
	git config core.hooksPath .githooks
	@echo "Installed .githooks (pre-push). Skip once: git push --no-verify"

pre-push:
	./cicd/scripts/pre-push.sh
