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
- **Local Kubernetes:** kind or minikube, Kustomize (`infra/k8s/overlays/dev`)
- **Infra (optional, AWS):** OpenTofu (VPC, EKS, RDS, ElastiCache)
- **Release delivery (optional, AWS):** GHCR images + ArgoCD (GitOps)
- **Resilience:** circuit breakers, retries with backoff, health probes
- **Service discovery:** native Kubernetes DNS (no Consul)

## Project structure

```
ridego/                                  ← one git repo, root of everything
│
├── go.work                              ← ties all Go modules together
├── go.work.sum
├── Makefile                             ← migrate/run/proto/tidy/docker/k8s commands
├── README.md                            ← project overview, you have this
├── .dockerignore                        ← keeps .env/*.pem/.git out of image builds
├── .gitignore
│
├── .github/
│   ├── workflows/ci.yaml                ← PR + main: go vet/build/test, docker builds,
│   │                                       kustomize+kubeconform, tofu validate
│   ├── workflows/publish.yaml           ← on `rel-*` tags: push GHCR images (`sha-*`
│   │                                       tags) and pin them into infra/k8s
│   └── dependabot.yaml                  ← weekly bumps for actions, base images, Go mods
│
├── services/                            ← THE ACTUAL APPLICATION — 6 independent Go programs
│   │
│   ├── gateway/                         ← entry point for all external traffic, :8080
│   │   ├── go.mod / go.sum
│   │   ├── .env / .env.example
│   │   ├── Dockerfile
│   │   ├── cmd/api/main.go              ← wires everything, starts the HTTP server
│   │   └── internal/
│   │       ├── config/config.go         ← reads env, loads JWT public key
│   │       ├── handler/                 ← routing table + middleware
│   │       ├── middleware/jwt.go        ← verifies tokens, stamps X-User-* headers
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
├── infra/                               ← EVERYTHING needed to deploy to Kubernetes/AWS
│   │
│   ├── tofu/                            ← provisions CLOUD INFRASTRUCTURE (optional —
│   │                                       only needed for AWS: VPC, EKS, RDS, Redis)
│   │   ├── main.tf / variables.tf / outputs.tf / versions.tf / backend.tf / helm.tf
│   │   ├── environments/{staging,prod}.tfvars
│   │   └── modules/{vpc,eks,rds,elasticache,k8s-secrets}/
│   │
│   └── k8s/                             ← describes WHAT RUNS INSIDE Kubernetes (Kustomize)
│       ├── base/                        ← namespace + NATS (shared, environment-agnostic)
│       │   ├── namespace.yaml
│       │   ├── nats.yaml
│       │   └── kustomization.yaml
│       ├── services/                    ← one Deployment+Service+HPA per microservice
│       │   ├── kustomization.yaml       ← aggregates all six services
│       │   ├── gateway/deployment.yaml  ← also has an Ingress (external entry point)
│       │   ├── user/deployment.yaml
│       │   ├── trip/deployment.yaml
│       │   ├── location/deployment.yaml
│       │   ├── matching/deployment.yaml
│       │   └── payment/deployment.yaml
│       └── overlays/
│           ├── dev/                     ← kind/minikube: :dev images, 1 replica,
│           │                               nginx ingress, in-cluster Postgres+Redis,
│           │                               non-sensitive ConfigMap (secrets are
│           │                               provisioned by `make k8s-secrets`)
│           └── prod/                    ← AWS: inherits base as-is (prod replicas,
│                                           resources, ALB ingress); image tags are
│                                           pinned to `sha-*` by the release workflow
│
└── .argocd/                             ← tells ArgoCD WHICH Kubernetes manifests to keep in sync (AWS path)
    ├── project.yaml                     ← groups everything under one ArgoCD "project"
    ├── root-app.yaml                    ← apply THIS ONE file to bootstrap all services at once
    ├── gateway-service-app.yaml
    ├── user-service-app.yaml
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
- `protoc` + Go protobuf plugins (only to regenerate gRPC code via `make proto`)
- `goose` (migrations)
- `kubectl` + [kind](https://kind.sigs.k8s.io/) or [minikube](https://minikube.sigs.k8s.io/) (local Kubernetes)
- OpenTofu 1.8+ and AWS access (only for the optional cloud path)

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
psql -U postgres -c "CREATE DATABASE ridego_payment_db;"
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
make jwt-dev-keys                              # generate throwaway local JWT keypair
make k8s-dev                                   # build :dev images, load, apply dev overlay
make k8s-secrets                               # (re)provision dev-only Secrets (never committed)
make k8s-migrate                               # run migrations against in-cluster Postgres
make k8s-status                                # show pods/services/ingress
make k8s-dev-down                              # delete the dev overlay
```

## Local Kubernetes (kind / minikube)

The default deployment target. Prerequisites: Docker running, `kubectl`,
and either [kind](https://kind.sigs.k8s.io/) or
[minikube](https://minikube.sigs.k8s.io/).

**Step 0 — start a cluster** (pick one):

```bash
# kind (named "ridego", which is the Makefile default)
kind create cluster --name ridego

# ...or minikube
minikube start
```

**Step 1 — deploy everything.** One command builds all six `:dev` images,
loads them into your cluster, provisions dev-only Secrets (throwaway JWT
keypair, in-cluster database URLs — never committed to git), and applies
the dev overlay (app + NATS + Postgres + Redis):

```bash
make k8s-dev
```

Using a differently-named kind cluster? `KIND_CLUSTER=my-cluster make k8s-dev`.

**Step 2 — create the tables** (user/trip/payment only):

```bash
make k8s-migrate
```

**Step 3 — verify:**

```bash
make k8s-status                  # pods, services, ingress in the ridego namespace
kubectl -n ridego get pods       # all should reach Running, including postgres/redis/nats
```

**Step 4 — talk to the gateway.** The quickest path is a port-forward:

```bash
kubectl -n ridego port-forward svc/gateway 8080:8080
curl localhost:8080/healthz
```

To go through the Ingress instead (the dev overlay already retargeted it
from the AWS ALB to nginx):

```bash
# minikube
minikube addons enable ingress
minikube tunnel   # in a separate terminal — exposes the ingress on localhost

# kind — install the nginx controller once per cluster
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl -n ingress-nginx port-forward svc/ingress-nginx-controller 8080:80
```

Then `curl localhost:8080/healthz`.

**Step 5 — tear it down:**

```bash
make k8s-dev-down
# ...and optionally: kind delete cluster --name ridego  /  minikube stop
```

What the dev overlay changes versus prod: local `:dev` images instead of
GHCR, 1 replica per service, relaxed autoscaling (no metrics-server
locally), an nginx Ingress instead of the AWS ALB, and in-cluster Postgres
(`postgres:16-alpine`, three databases) and Redis (`redis:7-alpine`) since
there is no RDS/ElastiCache locally.

## Seeing GitOps locally (ArgoCD on kind / minikube)

ArgoCD itself runs fine on a local cluster — it's just pods. This is the
fastest way to learn how the GitOps loop works before touching AWS.

**Step 0 — push your work.** ArgoCD reads from GitHub, not your laptop:

```bash
git add -A && git commit -m "..." && git push origin main
```

**Step 1 — install ArgoCD:**

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl -n argocd rollout status deploy/argocd-server --timeout=300s
```

**Step 2 — prep images and secrets, but don't apply anything yourself.**
Applying is ArgoCD's job — that's the whole point:

```bash
make k8s-images k8s-load k8s-secrets
```

**Step 3 — open the UI:**

```bash
kubectl -n argocd port-forward svc/argocd-server 8443:443 &
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo
```

Log in at `https://localhost:8443` as `admin` with that password.

**Step 4 — create a demo app pointing at the dev overlay:**

```bash
kubectl apply -f - <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: ridego-dev
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/orinameh/ridego.git
    targetRevision: main
    path: infra/k8s/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: ridego
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
EOF
```

Watch the UI go green as it deploys everything from Git. Note this
intentionally does **not** use `.argocd/root-app.yaml`: those apps reference
`:latest` GHCR images that don't exist until your first `rel-*` release, so
locally they'd sit in `ImagePullBackOff`. The dev overlay with local `:dev`
images is the correct demo target.

**Step 5 — witness the loop.** Change something visible — e.g. gateway
replicas `1` → `2` in `infra/k8s/overlays/dev/kustomization.yaml` — then:

```bash
git commit -am "demo: scale gateway to 2" && git push origin main
```

ArgoCD auto-syncs within ~3 minutes (or press **SYNC**) and the second pod
appears. Then try fighting it:

```bash
kubectl -n ridego scale deploy/gateway --replicas=5
```

`selfHeal` reverts your manual change — Git (the desired state) always wins
over `kubectl` (the live state). That's GitOps.

**Step 6 — clean up.** Delete the Application **first**, otherwise self-heal
resurrects everything `make k8s-dev-down` removes:

```bash
kubectl -n argocd delete app ridego-dev
make k8s-dev-down
```

## CI / CD

- **CI** (`.github/workflows/ci.yaml`) runs on every PR and push to `main`:
  `gofmt`, `go vet` / `go build` / `go test` per module, a no-push Docker
  build of all six images, `kustomize build` + `kubeconform` on both
  overlays, ArgoCD YAML lint, and `tofu fmt` + `validate` (no AWS
  credentials needed).
- **Publish** (`.github/workflows/publish.yaml`) runs only when you push a
  `rel-*` tag (e.g. `git tag rel-2026.09.1 && git push origin rel-2026.09.1`).
  It builds and pushes all six images to GHCR tagged immutably as
  `sha-<short-commit>` (never `:latest`), then commits the pinned
  tags back into `infra/k8s/` on `main` for ArgoCD to sync. Packages pushed
  with `GITHUB_TOKEN` default to private on GHCR — flip them public or add
  an `imagePullSecret`.
- **Dependabot** (`.github/dependabot.yaml`) proposes one grouped Go-module
  bump PR per week (a single PR keeps CI cost flat). Docker base images and
  action pins are bumped by hand when needed — CI validates both.

## Cloud deployment (optional, AWS)

Infrastructure is provisioned with OpenTofu; application delivery is handled
by ArgoCD watching this repo's Kubernetes manifests.

```bash
cd infra/tofu
tofu init
tofu apply -var-file=environments/staging.tfvars
aws eks update-kubeconfig --region eu-west-1 --name ridego-staging
```

ArgoCD then takes over — cut a `rel-*` tag and the Publish workflow pins
the new image SHAs into `infra/k8s/`, which ArgoCD syncs automatically.
Bootstrap ArgoCD itself with:

```bash
kubectl apply -f .argocd/root-app.yaml
```

## Design notes

A few decisions worth knowing if you're reading the code:

- **No Consul.** Service discovery is native Kubernetes DNS (`<service>.ridego.svc.cluster.local`). A `KubeResolver` in `pkg/proxy` falls back to `localhost` for local development via the `RIDEGO_LOCAL_DEV` env var.
- **Optimistic locking on trips.** The Trip Service uses a `version` column rather than row locks, so concurrent updates (e.g. a cancel racing a driver assignment) fail cleanly with a `409` instead of blocking.
- **Idempotency keys on payments.** Every charge capture is keyed by a deterministic hash of the trip ID, so a NATS event redelivery never double-charges.
- **Net/http only, no router library.** Go 1.22+'s `http.ServeMux` supports method + path-pattern routing (`"POST /v1/trips/{id}"`, `r.PathValue("id")`), which covers everything `gorilla/mux` was doing here.
- **One Postgres instance, three databases.** Each service still owns its data exclusively; this is a cost/complexity tradeoff appropriate for this scale, not a hard requirement of the architecture.

## License

Personal learning project — no license applied.
