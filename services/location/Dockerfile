# ---- build stage ----
FROM golang:1.26.10-alpine3.19 AS builder

# 1. Accept the service name as a dynamic build argument
ARG SERVICE_NAME
WORKDIR /app

# Cache dependency layer separately
COPY go.work go.work.sum ./
COPY services/${SERVICE_NAME}/go.mod services/${SERVICE_NAME}/go.sum ./services/${SERVICE_NAME}/
COPY proto/go.mod   proto/go.sum   ./proto/
COPY pkg/go.mod pkg/go.sum ./pkg/
RUN go work sync

COPY . .

# Regenerate protobuf (requires protoc in builder image)
RUN apk add --no-cache protobuf protobuf-dev && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    protoc --go_out=. --go-grpc_out=. proto/location/location.proto

# 2. Build dynamically using the passed service name argument
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /out/service-binary \
    ./services/${SERVICE_NAME}/cmd/api

# ---- runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

ARG SERVICE_NAME

COPY --from=builder /out/service-binary /service-binary

# 3. Dynamically bake in the migrations matching the target service
COPY --from=builder /app/services/${SERVICE_NAME}/migrations /migrations

EXPOSE 6003
EXPOSE 50051
ENTRYPOINT ["/service-binary"]
