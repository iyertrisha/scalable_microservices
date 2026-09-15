# FleetFlow

Real-time package delivery / fleet orchestration backend — built step by step to learn **Go**, **Docker**, **Kubernetes**, and **microservices scalability**

## Architecture

```text
Client
  │
  ▼
Order Service ──► Kafka (order.created) ──► Dispatch Service
  │                                            │
  ▼                                            ▼
PostgreSQL (orders)                     Driver Service (+ Redis lock)
                                               │
                                               ▼
                                        PostgreSQL (drivers)

Location ──► Kafka ──► Tracking (Redis latest GPS)
Pricing / Notifications consume the same event stream asynchronously
```

## Quick start

```bash
# 1) Start infra (Postgres, Redis, Kafka)
make infra

# 2) Run core services (separate terminals)
make run-order
make run-driver
make run-dispatch
```

> Host Postgres often owns `:5432`. Compose maps FleetFlow Postgres to **`:5433`**.  
> Kafka from the host is on **`:9094`**.

### Smoke test

```bash
# create driver
curl -s -X POST http://localhost:8081/drivers \
  -H 'Content-Type: application/json' \
  -d '{"name":"Asha","vehicle_type":"bike"}'

# make available (replace <id>)
curl -s -X PATCH http://localhost:8081/drivers/<id>/status \
  -H 'Content-Type: application/json' \
  -d '{"status":"AVAILABLE"}'

# create order → dispatch assigns a driver
curl -s -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"pickup_location":{"lat":12.97,"lng":77.59},"delivery_location":{"lat":12.98,"lng":77.60},"package_size":"M","priority":"high"}'

# poll until assigned_driver is set
curl -s http://localhost:8080/orders/<order-id>
```

### Full stack in Docker

```bash
make up          # builds and runs all app services + infra
make down        # tear down
```

## Learning path

See [docs/LEARNING.md](docs/LEARNING.md) for the step-by-step ladder.

## Ports

| Service | Port |
|---------|------|
| order | 8080 |
| driver | 8081 |
| dispatch | 8082 |
| tracking | 8083 |
| location | 8084 |
| pricing | 8085 |
| notification | 8086 |
| Kafka (host) | 9094 |
| Postgres (compose) | 5433 |
| Redis | 6379 |

## Kubernetes (kind)

```bash
brew install kind helm k6   # once
make kind-up
make kind-load
make kind-deploy

# optional: KEDA lag-based scaling + observability
helm repo add kedacore https://kedacore.github.io/charts
helm install keda kedacore/keda -n keda --create-namespace
kubectl apply -f deployments/k8s/keda/dispatch-scaledobject.yaml
kubectl apply -f deployments/observability/stack.yaml
```

## Load test

```bash
k6 run loadtests/orders.js
```

Results notes: [loadtests/RESULTS.md](loadtests/RESULTS.md)

## CI (Jenkins)

```bash
docker compose -f deployments/jenkins/docker-compose.yml up -d
# open http://localhost:8088 — pipeline is Jenkinsfile at repo root
```

## JWT

```bash
go run ./cmd/token-issuer
```

Set `AUTH_ENABLED=true` and `JWT_SECRET=...` on order-service to enforce Bearer tokens.

## AWS

See [docs/aws-eks.md](docs/aws-eks.md). Same manifests; swap images to ECR.

## Useful Make targets

| Target | What it does |
|--------|----------------|
| `make infra` | Postgres + Redis + Kafka |
| `make test` | `go test ./...` |
| `make build` | Build all service binaries into `bin/` |
| `make up` | Full Compose app stack |
| `make docker-build` | Multi-stage images for every service |
| `make kind-up` / `kind-load` / `kind-deploy` | Local Kubernetes path |
