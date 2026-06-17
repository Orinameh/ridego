# RideGo

A ride-sharing platform built as a Go microservices learning project — covering service design, distributed systems, Kubernetes, GitOps, and infrastructure as code.

RideGo is not a production product. It's a hands-on refresher for Go and microservices architecture, structured the way a real system would be: independently deployable services, their own databases, async events, service-to-service gRPC, and a Kubernetes-native deployment pipeline.

## Architecture

```
                        ┌─────────────┐
                        │  API Gateway │  :8080
                        └──────┬──────┘
           ┌──────────┬────────┼────────┬──────────┐
           ▼          ▼        ▼        ▼          ▼
       ┌───────┐  ┌───────┐ ┌────────┐ ┌─────────┐ ┌─────────┐
       │ User  │  │ Trip  │ │Location│ │Matching │ │ Payment │
       │ :6001 │  │ :6002 │ │ :6003  │ │  :6004  │ │  :6005  │
       └───┬───┘  └───┬───┘ └───┬────┘ └────┬────┘ └────┬────┘
           │          │         │           │           │
        Postgres   Postgres   Redis      (gRPC to    Postgres
                                          Location)
                          NATS (async events)
```

Each service owns its own database (or no database, where Redis/gRPC fits better). Services talk to each other three ways: synchronous HTTP through the Gateway for client-facing requests, gRPC for internal service-to-service calls (Matching → Location), and NATS for async events (Trip → Matching, Trip → Payment).

## Services

| Service | Port | Responsibility | Storage |
|---|---|---|---|
| Gateway | 8080 | Auth, rate limiting, routing, circuit breaking | Redis (rate limits only) |
| User | 6001 | Registration, login, JWT issuance | PostgreSQL |
| Trip | 6002 | Trip lifecycle state machine | PostgreSQL |
| Location | 6003 / 50051 (gRPC) | Real-time driver positions | Redis GEO |
| Matching | 6004 | Driver–rider pairing | none (gRPC + NATS) |
| Payment | 6005 | Charge capture, ledger | PostgreSQL |

## Stack

- **Language:** Go 1.26, stdlib `net/http` routing (no router library)
- **Databases:** PostgreSQL 16 (per-service), Redis 7 (geo + rate limiting)
- **Messaging:** NATS (async events), gRPC (sync internal calls)
- **Infra:** OpenTofu (VPC, EKS, RDS, ElastiCache on AWS)
- **Deployment:** Kubernetes, ArgoCD (GitOps), Helm
- **Resilience:** circuit breakers, retries with backoff, health probes
- **Service discovery:** native Kubernetes DNS (no Consul)

## Project structure

```
ridego/                                  ← one git repo, root of everything
│
├── go.work                              ← ties all Go modules together
├── go.work.sum
├── Makefile                             ← migrate/run/proto/tidy/docker-build commands
├── README.md                            ← project overview, you have this
├── Dockerfile                           ← (optional shared template — you use per-service ones instead)
├── .gitignore
│
├── services/                            ← THE ACTUAL APPLICATION — 6 independent Go programs
│   │
│   ├── gateway/                         ← entry point for all external traffic, :8080
│   │   ├── go.mod / go.sum
│   │   ├── .env / .env.example
│   │   ├── Dockerfile
│   │   ├── cmd/api/main.go              ← wires everything, starts the HTTP server
│   │   └── internal/
│   │       ├── config/config.go         ← reads .env, loads JWT public key
│   │       ├── handler/                 ← routing table + middleware
│   │       ├── middleware/jwt.go        ← verifies tokens, stamps X-User-* headers
│   │       ├── proxy/                   ← KubeResolver + reverse proxy + circuit breaker
│   │       └── ratelimit/limiter.go     ← Redis sliding window
│   │
│   ├── user/                            ← auth + user data, :6001
│   │   ├── go.mod / go.sum
│   │   ├── .env / .env.example
│   │   ├── Dockerfile
│   │   ├── secrets/jwt_private.pem      ← RSA private key, signs tokens
│   │   ├── secrets/jwt_public.pem       ← same public key copied into gateway's .env
│   │   ├── migrations/000001_init.sql
│   │   ├── cmd/api/main.go
│   │   └── internal/
│   │       ├── config/ handler/ models/ repository/ service/
│   │
│   ├── trip/                            ← trip lifecycle state machine, :6002
│   │   └── (same shape as user/, plus internal/statemachine/)
│   │
│   ├── location/                        ← driver GPS via Redis GEO, :6003 + :50051 (gRPC)
│   │   └── (same shape, plus internal/store/ internal/hub/ internal/grpc/)
│   │
│   ├── matching/                        ← pairs riders with drivers, :6004
│   │   └── (no database — internal/service/ only, calls Location via gRPC, Trip via HTTP)
│   │
│   └── payment/                         ← Stripe charges + ledger, :6005
│       └── (same shape as user/)
│
├── pkg/                                 ← SHARED Go code, imported by multiple services
│   ├── go.mod / go.sum
│   ├── events/events.go                 ← NATS event types + subject constants
│   ├── middleware/middleware.go         ← logging + recovery (shared HTTP middleware)
│   ├── proxy/                           ← KubeResolver (you put it here, not gateway-only)
│   └── resilience/resilience.go         ← circuit breaker + retry helper
│
├── proto/                               ← gRPC schema, its own Go module
│   ├── go.mod / go.sum
│   └── location/
│       ├── location.proto               ← hand-written schema
│       ├── location.pb.go                ← generated
│       └── location_grpc.pb.go           ← generated
│
├── infra/                               ← EVERYTHING needed to deploy to AWS/Kubernetes
│   │
│   ├── tofu/                            ← provisions the CLOUD INFRASTRUCTURE (VPC, EKS, RDS, Redis)
│   │   ├── main.tf / variables.tf / outputs.tf / versions.tf / backend.tf / helm.tf
│   │   ├── environments/{staging,prod}.tfvars
│   │   └── modules/{vpc,eks,rds,elasticache,k8s-secrets}/
│   │
│   └── k8s/                             ← describes WHAT RUNS INSIDE the Kubernetes cluster
│       ├── base/                        ← namespace + NATS (shared infra, not service-specific)
│       │   ├── namespace.yaml
│       │   ├── nats.yaml
│       │   └── kustomization.yaml
│       └── services/                    ← one Deployment+Service+HPA per microservice
│           ├── gateway/deployment.yaml   ← also has an Ingress (external entry point)
│           ├── user/deployment.yaml
│           ├── trip/deployment.yaml
│           ├── location/deployment.yaml
│           ├── matching/deployment.yaml
│           └── payment/deployment.yaml
│
└── .argocd/                             ← tells ArgoCD WHICH Kubernetes manifests to keep in sync
    ├── project.yaml                     ← groups everything under one ArgoCD "project"
    ├── root-app.yaml                    ← apply THIS ONE file to bootstrap all 6 services at once
    ├── gateway-app.yaml                 ← points ArgoCD at infra/k8s/services/gateway/
    ├── user-service-app.yaml            ← points ArgoCD at infra/k8s/services/user/
    ├── trip-service-app.yaml
    ├── location-service-app.yaml
    ├── matching-service-app.yaml
    └── payment-service-app.yaml
```

## Prerequisites

- Go 1.26+
- PostgreSQL 16 (local or Docker)
- Redis 7
- NATS server
- `protoc` + Go protobuf plugins
- `goose` (migrations)
- OpenTofu 1.8+ (only needed for cloud deployment)
- `kubectl`, a Kubernetes cluster (only needed for deployment)

## Local setup

Clone and bootstrap the workspace:

```bash
git clone <repo-url> ridego && cd ridego
go work sync
```

Each service needs its own `.env`, copied from its example and filled in:

```bash
for svc in gateway user trip location matching payment; do
  cp services/$svc/.env.example services/$svc/.env
done
```

The User Service and Gateway share an RSA key pair for JWT signing/verification. Generate one and reference it from both services' `.env` files — the Gateway only needs the public half.

Create the per-service databases:

```bash
psql -U postgres -c "CREATE DATABASE ridego_users_db;"
psql -U postgres -c "CREATE DATABASE ridego_trips_db;"
psql -U postgres -c "CREATE DATABASE ridego_payments_db;"
```

Run migrations and start a service:

```bash
make migrate-up service=user
make run service=user
```

Or do both in one step:

```bash
make dev service=user
```

Location, Matching, and Gateway have no migrations — use `make run` directly for those.

## Make targets

```bash
make help                                     # list everything below

make proto                                    # regenerate Go code from .proto files
make migrate-create service=<svc> name=<n>    # create a new migration
make migrate-up     service=<svc>             # apply pending migrations
make migrate-down   service=<svc>             # roll back last migration
make run  service=<svc>                       # run a service
make dev  service=<svc>                       # migrate then run
make docker-build service=<svc>               # build a Docker image
make tidy                                      # sync workspace, tidy all modules
```

## Deployment

Infrastructure is provisioned with OpenTofu, application delivery is handled by ArgoCD watching this repo's Kubernetes manifests.

```bash
cd infra/tofu
tofu init
tofu apply -var-file=environments/staging.tfvars
aws eks update-kubeconfig --region eu-west-1 --name ridego-staging
```

ArgoCD then takes over — merges to `main` trigger automatic sync of whatever's under `infra/k8s/`.

## Design notes

A few decisions worth knowing if you're reading the code:

- **No Consul.** Service discovery is native Kubernetes DNS (`<service>.ridego.svc.cluster.local`). A `KubeResolver` in `pkg/proxy` falls back to `localhost` for local development via the `RIDEGO_LOCAL_DEV` env var.
- **Optimistic locking on trips.** The Trip Service uses a `version` column rather than row locks, so concurrent updates (e.g. a cancel racing a driver assignment) fail cleanly with a `409` instead of blocking.
- **Idempotency keys on payments.** Every charge capture is keyed by a deterministic hash of the trip ID, so a NATS event redelivery never double-charges.
- **Net/http only, no router library.** Go 1.22+'s `http.ServeMux` supports method + path-pattern routing (`"POST /v1/trips/{id}"`, `r.PathValue("id")`), which covers everything `gorilla/mux` was doing here.
- **One Postgres instance, three databases.** Each service still owns its data exclusively; this is a cost/complexity tradeoff appropriate for this scale, not a hard requirement of the architecture.

## License

Personal learning project — no license applied.
