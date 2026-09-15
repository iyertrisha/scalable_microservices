.PHONY: deps infra up down test build run-order run-driver run-dispatch docker-build kind-up kind-deploy

COMPOSE=docker compose -f deployments/compose/docker-compose.yml

deps:
	go mod tidy

infra:
	$(COMPOSE) up -d postgres redis kafka

down:
	$(COMPOSE) --profile apps down -v

test:
	go test ./...

build:
	go build -o bin/order-service ./cmd/order-service
	go build -o bin/driver-service ./cmd/driver-service
	go build -o bin/dispatch-service ./cmd/dispatch-service
	go build -o bin/tracking-service ./cmd/tracking-service
	go build -o bin/location-service ./cmd/location-service
	go build -o bin/pricing-service ./cmd/pricing-service
	go build -o bin/notification-service ./cmd/notification-service
	go build -o bin/event-logger ./cmd/event-logger

run-order:
	ORDER_DATABASE_URL='postgres://fleetflow:fleetflow@localhost:5433/orders?sslmode=disable' \
	KAFKA_BROKERS=localhost:9094 \
	go run ./cmd/order-service

run-driver:
	DRIVER_DATABASE_URL='postgres://fleetflow:fleetflow@localhost:5433/drivers?sslmode=disable' \
	SAFE_ASSIGN=true \
	go run ./cmd/driver-service

run-dispatch:
	KAFKA_BROKERS=localhost:9094 \
	DRIVER_SERVICE_URL=http://localhost:8081 \
	ORDER_SERVICE_URL=http://localhost:8080 \
	REDIS_ADDR=localhost:6379 \
	go run ./cmd/dispatch-service

up:
	$(COMPOSE) --profile apps up -d --build

docker-build:
	docker build --build-arg SERVICE=order-service -t fleetflow/order-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=driver-service -t fleetflow/driver-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=dispatch-service -t fleetflow/dispatch-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=tracking-service -t fleetflow/tracking-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=location-service -t fleetflow/location-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=pricing-service -t fleetflow/pricing-service:local -f deployments/docker/Dockerfile .
	docker build --build-arg SERVICE=notification-service -t fleetflow/notification-service:local -f deployments/docker/Dockerfile .

kind-up:
	kind create cluster --name fleetflow --config deployments/k8s/kind-config.yaml || true
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

kind-load: docker-build
	kind load docker-image fleetflow/order-service:local --name fleetflow
	kind load docker-image fleetflow/driver-service:local --name fleetflow
	kind load docker-image fleetflow/dispatch-service:local --name fleetflow
	kind load docker-image fleetflow/tracking-service:local --name fleetflow
	kind load docker-image fleetflow/location-service:local --name fleetflow
	kind load docker-image fleetflow/pricing-service:local --name fleetflow
	kind load docker-image fleetflow/notification-service:local --name fleetflow

kind-deploy:
	kubectl apply -k deployments/k8s/overlays/local
