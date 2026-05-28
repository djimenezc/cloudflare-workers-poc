.PHONY: help install dev deploy logs list images clean tidy

WORKER_NAME := vega-ephemeral-dev

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

install: ## Install npm dependencies
	npm install

tidy: ## Refresh Go module deps for the workspace agent
	cd container_src && go mod tidy

dev: ## Run locally with wrangler dev (requires Docker)
	npx wrangler dev

deploy: ## Build container image, push to Cloudflare Registry, and deploy worker
	npx wrangler deploy

logs: ## Tail live worker logs
	npx wrangler tail $(WORKER_NAME)

list: ## List running container instances
	npx wrangler containers list

images: ## List container images in Cloudflare Registry
	npx wrangler containers images list

clean: ## Remove node_modules and wrangler cache
	rm -rf node_modules .wrangler
