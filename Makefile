# Load .env file
# Dynamically include the .env file of the requested service
ifneq ($(service),)
    ifneq (,$(wildcard ./services/$(service)/.env))
        include ./services/$(service)/.env
        export
    endif
endif

.PHONY: migrate-create migrate-up migrate-down run dev docker-build proto tidy wire-pkg wire-proto jwt-dev-keys k8s-images k8s-load k8s-secrets k8s-dev k8s-migrate k8s-dev-down k8s-status help

SERVICES := gateway user trip location matching payment
KIND_CLUSTER := ridego

# Throwaway RSA keypair for local JWT signing (User signs, Gateway verifies).
# Lives in gitignored services/user/secrets/ — never committed, never prod.
JWT_DIR := services/user/secrets
JWT_PRIV := $(JWT_DIR)/dev-private.pem
JWT_PUB := $(JWT_DIR)/dev-public.pem

jwt-dev-keys:
	@if [ -f "$(JWT_PRIV)" ]; then echo "Reusing existing dev keypair in $(JWT_DIR)/"; else \
		mkdir -p $(JWT_DIR) && \
		openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out $(JWT_PRIV) && \
		openssl pkey -in $(JWT_PRIV) -pubout -out $(JWT_PUB) && \
		echo "Generated throwaway dev keypair in $(JWT_DIR)/ (gitignored — never commit)"; \
	fi

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
	docker build --build-arg SERVICE_NAME=$(service) -f services/$(service)/Dockerfile -t ridego-$(service):latest .

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

# --- Local Kubernetes (kind / minikube) -------------------------------
# make k8s-dev        Build :dev images, load them into the local cluster,
#                     provision dev secrets, and apply the dev overlay
#                     (app + NATS + Postgres + Redis).
# make k8s-secrets    (Re)provision dev-only Secrets (throwaway JWT keypair,
#                     in-cluster DATABASE_URLs). Never committed to git.
# make k8s-migrate    Run pending SQL migrations via a Postgres port-forward.
# make k8s-dev-down   Tear the dev overlay back down.
k8s-images:
	@for svc in $(SERVICES); do \
		echo "Building ridego-$$svc:dev..."; \
		docker build --build-arg SERVICE_NAME=$$svc -f services/$$svc/Dockerfile -t ridego-$$svc:dev . || exit 1; \
	done

k8s-load:
	@if kind get clusters 2>/dev/null | grep -qx "$(KIND_CLUSTER)"; then \
		echo "Loading images into kind cluster '$(KIND_CLUSTER)'..."; \
		for svc in $(SERVICES); do kind load docker-image ridego-$$svc:dev --name $(KIND_CLUSTER) || exit 1; done; \
	elif minikube status -f '{{.Host}}' 2>/dev/null | grep -qx Running; then \
		echo "Loading images into minikube..."; \
		for svc in $(SERVICES); do minikube image load ridego-$$svc:dev || exit 1; done; \
	else \
		echo "Error: no kind cluster named '$(KIND_CLUSTER)' and minikube is not running."; exit 1; \
	fi

k8s-dev: k8s-images k8s-load k8s-secrets
	kubectl apply -k infra/k8s/overlays/dev
	@echo "Waiting for rollouts (up to 3m per deployment)..."
	@for d in nats postgres redis gateway user-service trip-service location-service matching-service payment-service; do \
		kubectl -n ridego rollout status deploy/$$d --timeout=180s || exit 1; \
	done
	@echo ""
	@echo "RideGo is up. Next steps:"
	@echo "  make k8s-migrate              # run pending SQL migrations (user/trip/payment)"
	@echo "  kubectl -n ridego get pods    # inspect"

k8s-migrate:
	@if ! command -v goose >/dev/null 2>&1; then echo "Error: goose is not installed (go install github.com/pressly/goose/v3/cmd/goose@latest)"; exit 1; fi
	@kubectl -n ridego wait --for=condition=ready pod -l app=postgres --timeout=120s
	@kubectl -n ridego port-forward svc/postgres 5432:5432 >/dev/null 2>&1 & \
	PF_PID=$$!; \
	trap 'kill $$PF_PID 2>/dev/null' EXIT INT TERM; \
	sleep 2; \
	$(MAKE) migrate-up service=user DATABASE_URL="postgres://postgres:postgres@localhost:5432/ridego_users_db?sslmode=disable" && \
	$(MAKE) migrate-up service=trip DATABASE_URL="postgres://postgres:postgres@localhost:5432/ridego_trips_db?sslmode=disable" && \
	$(MAKE) migrate-up service=payment DATABASE_URL="postgres://postgres:postgres@localhost:5432/ridego_payment_db?sslmode=disable"

k8s-dev-down:
	kubectl delete -k infra/k8s/overlays/dev

# Provision dev-only Secrets. The JWT keypair is generated on first run into
# gitignored services/user/secrets/; DATABASE_URLs point at the in-cluster
# Postgres. Idempotent — safe to re-run any time.
k8s-secrets: jwt-dev-keys
	kubectl create namespace ridego --dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic gateway-secrets \
		--from-file=JWT_PUBLIC_KEY=$(JWT_PUB) \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic user-service-secrets \
		--from-literal=DATABASE_URL="postgres://postgres:postgres@postgres.ridego.svc.cluster.local:5432/ridego_users_db?sslmode=disable" \
		--from-file=JWT_PRIVATE_KEY=$(JWT_PRIV) \
		--from-file=JWT_PUBLIC_KEY=$(JWT_PUB) \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic trip-service-secrets \
		--from-literal=DATABASE_URL="postgres://postgres:postgres@postgres.ridego.svc.cluster.local:5432/ridego_trips_db?sslmode=disable" \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic location-service-secrets \
		--from-literal=REDIS_PASSWORD="" \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic matching-service-secrets \
		--from-literal=NATS_URL="nats://nats.ridego.svc.cluster.local:4222" \
		--dry-run=client -o yaml | kubectl apply -f -
	kubectl -n ridego create secret generic payment-service-secrets \
		--from-literal=DATABASE_URL="postgres://postgres:postgres@postgres.ridego.svc.cluster.local:5432/ridego_payment_db?sslmode=disable" \
		--from-literal=STRIPE_KEY="sk_test_dev_only_do_not_use" \
		--dry-run=client -o yaml | kubectl apply -f -

k8s-status:
	kubectl -n ridego get pods,svc,ingress

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
	@echo "Local Kubernetes (kind/minikube)"
	@echo "  make k8s-dev                                 Build :dev images, load, apply dev overlay"
	@echo "  make k8s-secrets                             (Re)provision dev-only Secrets (never committed)"
	@echo "  make jwt-dev-keys                            Generate throwaway local JWT keypair"
	@echo "  make k8s-migrate                             Run migrations via Postgres port-forward"
	@echo "  make k8s-status                              Show pods/services/ingress in ridego ns"
	@echo "  make k8s-dev-down                            Delete the dev overlay"
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