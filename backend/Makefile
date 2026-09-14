# Financial Tracker backend — dev orchestration (docs/plan.md Phase 2)
COMPOSE := docker compose

.PHONY: up down logs ps shell migrate dbshell build

up: ## Build images and start db + app (migrations run in app entrypoint, D-014)
	$(COMPOSE) up -d --build

down: ## Stop and remove containers
	$(COMPOSE) down

logs: ## Tail app logs
	$(COMPOSE) logs -f --tail=100 app

ps: ## Show services
	$(COMPOSE) ps

shell: ## Shell inside the app container
	$(COMPOSE) exec app sh

dbshell: ## psql inside the db container (uses container env credentials)
	$(COMPOSE) exec db sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"'

migrate: ## Apply pending migrations only (idempotent)
	$(COMPOSE) run --rm app migrate

build: ## Build the app image
	$(COMPOSE) build app