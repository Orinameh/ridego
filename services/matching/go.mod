module github.com/ridego/services/matching

go 1.26.3

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/nats-io/nats.go v1.54.0
	google.golang.org/grpc v1.83.2
)

require (
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/klauspost/compress v1.20.0 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ridego/pkg v0.0.0
	github.com/ridego/proto v0.0.0
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace github.com/ridego/pkg => ../../pkg

replace github.com/ridego/proto => ../../proto
