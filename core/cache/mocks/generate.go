package mocks

// Regenerate mocks for cache interfaces.
// 重新生成缓存接口的 mock 代码。
//go:generate go tool mockgen -source=../cache.go -destination=mock_cache.go -package=mocks
//go:generate go tool mockgen -source=../store.go -destination=mock_store.go -package=mocks
