module github.com/ridego/services/location

go 1.26.3

require (
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.23.0
	github.com/ridego/proto v0.0.0
	google.golang.org/grpc v1.85.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/ridego/pkg v0.0.0
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/net v0.60.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260817212433-ac3dfec99bb1 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/ridego/proto => ../../proto

replace github.com/ridego/pkg => ../../pkg
