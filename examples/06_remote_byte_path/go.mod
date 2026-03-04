module github.com/yikakia/cachalot/examples/06_remote_byte_path

go 1.25.7

require (
	github.com/redis/go-redis/v9 v9.18.0
	github.com/yikakia/cachalot v0.0.0
	github.com/yikakia/cachalot/stores/redis v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
)

replace github.com/yikakia/cachalot => ../..

replace github.com/yikakia/cachalot/stores/redis => ../../stores/redis
