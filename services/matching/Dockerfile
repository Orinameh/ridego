# ---- build stage ----
FROM golang:1.26.10-alpine3.19 AS builder

# Accept the service name as a dynamic build argument
ARG SERVICE_NAME
WORKDIR /app

# Cache dependency layer separately
COPY go.work go.work.sum ./
COPY services/${SERVICE_NAME}/go.mod services/${SERVICE_NAME}/go.sum ./services/${SERVICE_NAME}/
COPY pkg/go.mod pkg/go.sum ./pkg/
RUN go work sync

COPY . .

# Payment Service does not use gRPC/proto — no protoc step needed here.
# (Location and Matching Dockerfiles include the protoc regen block instead.)

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

EXPOSE 6004
ENTRYPOINT ["/service-binary"]