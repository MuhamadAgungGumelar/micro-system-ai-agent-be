.PHONY: help migrate-up migrate-down migrate-version migrate-force swagger run-saas run-agent

help:
	@echo "Available commands:"
	@echo "  make migrate-up MODULE=saas       - Run UP migrations for specified module"
	@echo "  make migrate-down MODULE=saas     - Run DOWN migrations for specified module"
	@echo "  make migrate-version MODULE=saas  - Show current migration version"
	@echo "  make migrate-force VERSION=1      - Force migration to specific version"
	@echo "  make swagger                      - Regenerate Swagger docs"
	@echo "  make run-saas                     - Run saas-api server"
	@echo "  make run-agent                    - Run agent-core"

# Migration commands
migrate-up:
	@go run cmd/migrate/main.go -module=$(MODULE) -cmd=up

migrate-down:
	@go run cmd/migrate/main.go -module=$(MODULE) -cmd=down

migrate-version:
	@go run cmd/migrate/main.go -module=$(MODULE) -cmd=version

migrate-force:
	@go run cmd/migrate/main.go -module=$(MODULE) -cmd=force $(VERSION)

# Swagger generation
swagger:
	@swag init -g cmd/saas-api/main.go --output cmd/saas-api/docs

# Run servers
run-saas:
	@go run cmd/saas-api/main.go

run-agent:
	@go run cmd/agent-core/main.go
