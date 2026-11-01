module github.com/ridego/services/matching

go 1.26.3

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/nats-io/nats.go v1.55.0
	google.golang.org/grpc v1.85.0
)

require (
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260817212433-ac3dfec99bb1 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
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
