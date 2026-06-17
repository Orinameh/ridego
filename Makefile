# Load .env file
# Dynamically include the .env file of the requested service
ifneq ($(service),)
    ifneq (,$(wildcard ./services/$(service)/.env))
        include ./services/$(service)/.env
        export
    endif
endif

.PHONY: migrate-create migrate-up migrate-down run dev docker-build proto tidy wire-pkg wire-proto help

# 1. Generic rule to CREATE a new migration file
# make migrate-create service=user name=add_users_table
migrate-create:
	@if [ -z "$(service)" ] || [ -z "$(name)" ]; then echo "Error: 'service' and 'name' are required. Example: make migrate-create service=user name=add_users_table"; exit 1; fi
	@echo "Creating migration '$(name)' for services/$(service)..."
	goose -dir services/$(service)/migrations create $(name) sql

# 2. Generic rule to APPLY migrations for a service
# make migrate-up service=user
migrate-up:
	@if [ -z "$(service)" ]; then echo "Error: 'service' is required. Example: make migrate-up service=user"; exit 1; fi
	@if [ -z "$(DATABASE_URL)" ]; then echo "Error: DATABASE_URL is not set in services/$(service)/.env"; exit 1; fi
	@echo "Running migrations for services/$(service)..."
	goose -dir services/$(service)/migrations postgres "$(DATABASE_URL)" up

# 3. Generic rule to ROLLBACK migrations for a service
# make migrate-down service=user
migrate-down:
	@if [ -z "$(service)" ]; then echo "Error: 'service' is required. Example: make migrate-down service=user"; exit 1; fi
	@echo "Rolling back migrations for services/$(service)..."
	goose -dir services/$(service)/migrations postgres "$(DATABASE_URL)" down

# 4. Generic rule to RUN a specific service
# make run service=user
# make dev service=user
run:
	@if [ -z "$(service)" ]; then echo "Error: 'service' is required. Example: make run service=user"; exit 1; fi
	@echo "Starting services/$(service)..."
	cd services/$(service) && go run cmd/api/main.go

# 5. FIXED: Generic rule to MIGRATE AND RUN sequentially
dev:
	@if [ -z "$(service)" ]; then echo "Error: 'service' is required. Example: make dev service=user"; exit 1; fi
	@$(MAKE) migrate-up service=$(service)
	@$(MAKE) run service=$(service)

# 6. Generic rule to BUILD a Docker image
# make docker-build service=user
docker-build:
	@if [ -z "$(service)" ]; then echo "Error: 'service' is required. Example: make docker-build service=user"; exit 1; fi
	@echo "Building Docker image for ridego-$(service)..."
	docker build --build-arg SERVICE_NAME=$(service) -t ridego-$(service):latest .

# Add this target
# make proto
proto:
	@echo "Generating Go code from proto files..."
	protoc \
		--proto_path=proto \
		--go_out=proto \
		--go-grpc_out=proto \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		location/location.proto
	@echo "Proto generation complete"

# make tidy
tidy:
	@echo "Syncing workspace and tidying all modules..."
	go work sync
	@for dir in services/user services/trip services/location \
	             services/matching services/payment services/gateway \
	             pkg proto; do \
		echo "Tidying $$dir..."; \
		(cd $$dir && go mod tidy && cd ..); \
	done
	@echo "Done"

# make wire-pkg service=newservice
wire-pkg:
	@if [ -z "$(service)" ]; then echo "Error: service required"; exit 1; fi
	cd services/$(service) && \
	go mod edit -require=github.com/ridego/pkg@v0.0.0 && \
	go mod edit -replace github.com/ridego/pkg=../../pkg
	@echo "Wired services/$(service) to local pkg module"

# make wire-proto service=newservice
wire-proto:
	@if [ -z "$(service)" ]; then echo "Error: service required"; exit 1; fi
	cd services/$(service) && \
	go mod edit -require=github.com/ridego/proto@v0.0.0 && \
	go mod edit -replace github.com/ridego/proto=../../proto
	@echo "Wired services/$(service) to local proto module"

# make help
help:
	@echo ""
	@echo "RideGo — available commands"
	@echo ""
	@echo "Proto"
	@echo "  make proto                                    Generate Go code from .proto files"
	@echo ""
	@echo "Migrations"
	@echo "  make migrate-create service=<svc> name=<n>   Create a new migration file"
	@echo "  make migrate-up     service=<svc>             Apply all pending migrations"
	@echo "  make migrate-down   service=<svc>             Roll back the last migration"
	@echo ""
	@echo "Running"
	@echo "  make run  service=<svc>                       Run a service"
	@echo "  make dev  service=<svc>                       Migrate then run a service"
	@echo ""
	@echo "Docker"
	@echo "  make docker-build service=<svc>               Build a Docker image for a service"
	@echo ""
	@echo "Workspace"
	@echo "  make tidy                                     Sync workspace and tidy all modules"
	@echo ""
	@echo "  make wire-pkg service=<svc>                               Wire local pkg"
	@echo ""
	@echo "  make wire-proto service=<svc>                             Wire local proto"
	@echo ""
	@echo "Services: user | trip | location | matching | payment | gateway"
	@echo ""