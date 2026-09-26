module github.com/ridego/services/gateway

go 1.26.3

require (
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/redis/go-redis/v9 v9.20.0
)

require github.com/sony/gobreaker/v2 v2.4.0 // indirect

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/ridego/pkg v0.0.0
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/ridego/pkg => ../../pkg
