package mocks

// Regenerate mocks for cache interfaces.
// 重新生成缓存接口的 mock 代码。
//go:generate go tool mockgen -source=../../core/cache/cache.go -destination=mock_cache.go -package=mocks
//go:generate go tool mockgen -source=../../core/cache/store.go -destination=mock_store.go -package=mocks
