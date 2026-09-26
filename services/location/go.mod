module github.com/ridego/services/location

go 1.26.3

require (
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/redis/go-redis/v9 v9.20.0
	github.com/ridego/proto v0.0.0
	google.golang.org/grpc v1.84.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/ridego/pkg v0.0.0
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/ridego/proto => ../../proto

replace github.com/ridego/pkg => ../../pkg
