# FleetFlow learning ladder



Build and understand one layer at a time.


| Step | What you build                 | What you should understand              |
| ---- | ------------------------------ | --------------------------------------- |
| 1    | In-memory Order API            | Go module, handlers, JSON               |
| 2    | Postgres for orders            | Persistence, Compose for deps           |
| 3    | Driver service + DB            | Process boundary, DB-per-service        |
| 4    | Kafka `order.created`          | Events vs RPC; backlog if consumer down |
| 5    | Naive dispatch                 | Event-driven workflow; **race**         |
| 6    | Atomic assign + Redis lock     | Why two pods can both see AVAILABLE     |
| 7    | Docker images                  | Image vs container                      |
| 8    | kind + one Deployment          | Pod / Deployment / Service              |
| 9    | All services + Ingress         | Discovery, gateway                      |
| 10   | HPA then KEDA                  | CPU vs Kafka lag scaling                |
| 11   | Location + Tracking            | Redis hot path                          |
| 12   | Pricing + Notifications        | Async side effects                      |
| 13   | OTel / Prom / Grafana / Jaeger | Metrics vs traces                       |
| 14   | k6 + Jenkins + JWT + AWS path  | Evidence + delivery                     |


Commands live in the root `Makefile` and `README.md`.