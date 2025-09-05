# Color definitions
COLOR_RESET = \033[0m
COLOR_RED = \033[31m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m

help: ## Show avaliable commands
	@echo "$(COLOR_GREEN)Usage:$(COLOR_RESET)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | sed 's/.*Makefile://' | awk 'BEGIN {FS = ":.*?## "}; {printf "$(COLOR_YELLOW)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'

up: ## Start all services
	docker-compose down && docker-compose up --build -d

logs: ## Show logs
	docker-compose logs --tail=30   