package cached

import (
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
)

type Cached struct {
	ctx     context.Context
	KvStore *xkv.Store
}

func NewCache(ctx context.Context, kvStore *xkv.Store) *Cached {
	return &Cached{
		ctx:     ctx,
		KvStore: kvStore,
	}
}
