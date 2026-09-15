# Multi-stage Dockerfile for any FleetFlow Go service.
# Build: docker build --build-arg SERVICE=order-service -f deployments/docker/Dockerfile .
ARG SERVICE=order-service

FROM golang:1.26-alpine AS build
ARG SERVICE
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/service ./cmd/${SERVICE}

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 fleetflow
USER fleetflow
WORKDIR /app
COPY --from=build /out/service /app/service
EXPOSE 8080
ENTRYPOINT ["/app/service"]
