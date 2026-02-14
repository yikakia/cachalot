package write_back

import (
	"context"
	"time"

	"github.com/sourcegraph/conc/panics"
	"github.com/yikakia/cachalot/core/cache"
	"github.com/yikakia/cachalot/core/multicache"
)

type ErrCallback[T any] func(ctx context.Context, getCtx *multicache.FetchContext[T], caches []cache.Cache[T], err error)

func DefaultBuilder[T any]() Builder[T] {
	return Builder[T]{
		DefaultTTL: time.Minute,
	}
}

// Builder 构建 multicache.WriteBackFn 的简单助手
// 写回策略默认并发写回并同步等待，返回错误
// 写策略 可自定义 可包裹为异步执行
type Builder[T any] struct {
	// 默认配置写回策略中 调用 cache.Set 时传入的 TTL
	DefaultTTL time.Duration

	// 可以自定义传入方法
	CustomWriteBack multicache.WriteBackFn[T]
	// 异步执行
	Async bool

	// 执行出错时回调
	ErrCallback ErrCallback[T]
}

func (w Builder[T]) Build() multicache.WriteBackFn[T] {
	fn := w.buildBasicFn()
	cb := w.buildCallback()

	if w.Async {
		return w.wrapAsync(fn, cb)
	}

	return w.wrapErrCallback(fn, cb)
}

func (w Builder[T]) wrapAsync(fn multicache.WriteBackFn[T], cb ErrCallback[T]) multicache.WriteBackFn[T] {
	return asyncDecorator(fn, cb)
}

func (w Builder[T]) buildBasicFn() multicache.WriteBackFn[T] {
	if w.CustomWriteBack != nil {
		return w.CustomWriteBack
	}

	return multicache.WriteBackParallel[T](w.DefaultTTL)
}

func (w Builder[T]) buildCallback() ErrCallback[T] {
	if w.ErrCallback == nil {
		return nopeCallback
	}

	return w.ErrCallback
}

func (w Builder[T]) wrapErrCallback(fn multicache.WriteBackFn[T], cb ErrCallback[T]) multicache.WriteBackFn[T] {
	return func(ctx context.Context, getCtx *multicache.FetchContext[T], caches []cache.Cache[T]) error {
		err := fn(ctx, getCtx, caches)
		if err != nil {
			cb(ctx, getCtx, caches, err)
			return err
		}

		return nil
	}
}

// 并发装饰
func asyncDecorator[T any](f multicache.WriteBackFn[T], cb ErrCallback[T]) multicache.WriteBackFn[T] {
	return func(ctx context.Context, getCtx *multicache.FetchContext[T], caches []cache.Cache[T]) error {
		go func() {
			panics.Try(func() {
				err := f(ctx, getCtx, caches)
				cb(ctx, getCtx, caches, err)
			})
		}()
		return nil
	}
}

func nopeCallback[T any](ctx context.Context, getCtx *multicache.FetchContext[T], caches []cache.Cache[T], err error) {
}
