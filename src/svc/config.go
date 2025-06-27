package svc

import (
	cached "EasySwapBackend-test/src/cache"
	"EasySwapBackend-test/src/dao"
	"github.com/ProjectsTask/EasySwapBase/evm/erc"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"gorm.io/gorm"
)

/*
*
定义了一个服务上下文(ServerCtx)的构建系统，采用选项模式(Option Pattern)来灵活配置服务上下文
*/
type CtxConfig struct {
	db      *gorm.DB
	dao     *dao.Dao
	KvStore *xkv.Store
	Cached  *cached.Cached
	Evm     erc.Erc
}

// 这是一个函数类型，用于修改 CtxConfig
// 是选项模式的核心，每个选项都是一个可以配置 CtxConfig 的函数
type CtxOption func(conf *CtxConfig)

// 服务上下文的构造函数
func NewServerCtx(options ...CtxOption) *ServerCtx {
	c := &CtxConfig{} //创建一个空的 CtxConfig，然后应用所有选项
	for _, opt := range options {
		opt(c)
	}
	return &ServerCtx{
		DB:      c.db,
		Dao:     c.dao,
		KvStore: c.KvStore,
		Cached:  c.Cached,
	}
}

/**
选项函数
*/

func WithDB(db *gorm.DB) CtxOption {
	return func(conf *CtxConfig) {
		conf.db = db
	}
}
func WithDao(dao *dao.Dao) CtxOption {
	return func(conf *CtxConfig) {
		conf.dao = dao
	}
}

func WithKv(kv *xkv.Store) CtxOption {
	return func(conf *CtxConfig) {
		conf.KvStore = kv
	}
}

func WithCached(cached *cached.Cached) CtxOption {
	return func(conf *CtxConfig) {
		conf.Cached = cached
	}
}
