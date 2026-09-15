# Load test results

**Date:** 2026-09-14  
**Environment:** local Go services + Docker Compose (Postgres :5433, Kafka :9094, Redis :6379)  
**Command:** `k6 run --vus 15 --duration 20s loadtests/orders.js`

## Summary

| Metric | Value |
|--------|-------|
| VUs | 15 |
| Duration | 20s |
| Iterations | 135 |
| Order checks | 100% (270/270) |
| Order error rate | **0.00%** |
| Order p95 latency | **1.08s** (threshold `<1.5s` passed) |
| Overall RPS | ~26.7 req/s |

Thresholds on `{name:order}` **passed**. Overall `http_req_failed` includes optional tracking/location probes and is not the gate.

## Kubernetes evidence (same day)

- kind cluster `fleetflow` with Ingress nginx, HPA on order/driver, KEDA ScaledObject on dispatch (**Ready=True, Active=True** after Kafka advertised FQDN fix)
- Port-forward smoke: create driver → create order → **assigned in 1 poll**
- Observability pods Running: Grafana, Jaeger, Prometheus (`observability` namespace)

## Notes

- Host Postgres already binds `:5432`; Compose maps FleetFlow Postgres to **`:5433`**
- Kafka from host uses **`:9094`** (`PLAINTEXT_HOST`); in-cluster uses `kafka:9092`
