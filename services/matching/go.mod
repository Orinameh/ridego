module github.com/ridego/services/matching

go 1.26.3

require (
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/nats-io/nats.go v1.52.0
	google.golang.org/grpc v1.81.1
)

require (
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260226221140-a57be14db171 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ridego/pkg v0.0.0
	github.com/ridego/proto v0.0.0
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/ridego/pkg => ../../pkg

replace github.com/ridego/proto => ../../proto
