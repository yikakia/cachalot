module github.com/yikakia/cachalot/examples/03_logical_expiry

go 1.25.7

require (
	github.com/dgraph-io/ristretto/v2 v2.4.0
	github.com/yikakia/cachalot v0.0.0
	github.com/yikakia/cachalot/stores/ristretto v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
)

replace github.com/yikakia/cachalot => ../..

replace github.com/yikakia/cachalot/stores/ristretto => ../../stores/ristretto
