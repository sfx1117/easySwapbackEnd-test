package svc

import (
	"EasySwapBackend-test/src/config"
	"EasySwapBackend-test/src/dao"
	"context"
	"github.com/ProjectsTask/EasySwapBase/chain/nftchainservice"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

type ServerCtx struct {
	C        *config.Config
	DB       *gorm.DB
	Dao      *dao.Dao
	KvStore  *xkv.Store
	RankKey  string
	NodeSrvs map[int64]*nftchainservice.Service
}

func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	//1、使用zap初始化日志
	_, err := xzap.SetUp(c.Log)
	if err != nil {
		return nil, err
	}
	//2、初始化redis/kv存储
	var kvConf kv.KvConf
	for _, con := range c.Kv.Redis {
		kvConf = append(kvConf, cache.NodeConf{
			RedisConf: redis.RedisConf{
				Host: con.Host,
				Type: con.Type,
				Pass: con.Pass,
			},
			Weight: 1, // 权重（可能用于负载均衡）
		})
	}
	store := xkv.NewStore(kvConf) // 初始化 Redis 客户端,创建 Redis 存储

	//3、初始化数据库
	db, err := gdb.NewDB(&c.DB)
	if err != nil {
		return nil, err
	}

	//4、区块链节点服务初始化
	nodeSrvs := make(map[int64]*nftchainservice.Service)
	for _, supported := range c.ChainSupported {
		nodeSrvs[int64(supported.ChainId)], err = nftchainservice.New(
			context.Background(),
			supported.Endpoint,
			supported.Name,
			supported.ChainId,
			c.MetadataParse.NameTags,
			c.MetadataParse.ImageTags,
			c.MetadataParse.AttributesTags,
			c.MetadataParse.TraitNameTags,
			c.MetadataParse.TraitValueTags,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed on start onchain sync service")
		}
	}

	//5、dao层初始化
	dao := dao.New(context.Background(), db, store)

	//6、创建服务上下文
	serverCtx := NewServerCtx(WithDao(dao), WithDB(db), WithKv(store))
	serverCtx.C = c
	serverCtx.NodeSrvs = nodeSrvs
	return serverCtx, nil
}
