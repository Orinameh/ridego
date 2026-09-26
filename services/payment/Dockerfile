# ---- build stage ----
FROM golang:1.26.8-alpine3.24 AS builder

# Accept the service name as a dynamic build argument
ARG SERVICE_NAME
WORKDIR /app

# Cache dependency layer separately
COPY go.work go.work.sum ./
# go.work references every module, so `go work sync` needs ALL go.mod
# files present — not just the target service's. (Sources follow later.)
COPY services/gateway/go.mod services/gateway/go.sum ./services/gateway/
COPY services/user/go.mod services/user/go.sum ./services/user/
COPY services/trip/go.mod services/trip/go.sum ./services/trip/
COPY services/location/go.mod services/location/go.sum ./services/location/
COPY services/matching/go.mod services/matching/go.sum ./services/matching/
COPY services/payment/go.mod services/payment/go.sum ./services/payment/
COPY proto/go.mod proto/go.sum ./proto/
COPY pkg/go.mod pkg/go.sum ./pkg/
RUN go work sync

COPY . .

# Payment does not use gRPC/proto directly, but the workspace still needs
# proto/go.mod present for `go work sync`. Builds against the committed
# generated code — no protoc step.

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /out/service-binary \
    ./services/${SERVICE_NAME}/cmd/api

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

ARG SERVICE_NAME

COPY --from=builder /out/service-binary /service-binary

# Bake in migrations matching the target service
COPY --from=builder /app/services/${SERVICE_NAME}/migrations /migrations

EXPOSE 6005
ENTRYPOINT ["/service-binary"]