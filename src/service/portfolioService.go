package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"strings"
	"sync"
)

const BidTypeOffset = 3

func getBidType(origin int64) int64 {
	if origin >= BidTypeOffset {
		return origin - BidTypeOffset
	} else {
		return origin
	}
}

// 获取用户拥有Collection信息： 拥有item数量、上架数量、floor price
func GetMultiChainUserCollection(ctx context.Context, serverCtx *svc.ServerCtx, chainIds []int, chainNames, userAddrs []string) (*entity.UserCollectionsResp, error) {
	//1、查询用户在多条链上的collection基本信息
	collections, err := serverCtx.Dao.QueryMultiChainUserCollectionInfos(ctx, chainIds, chainNames, userAddrs)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}
	//2、构建chainID到chainName的映射
	chainIdToChainNameMap := make(map[int]string)
	for _, chain := range serverCtx.C.ChainSupported {
		chainIdToChainNameMap[chain.ChainId] = chain.Name
	}
	//3、构建chainId到collection的映射
	chainIdToCollAddrMap := make(map[int][]string)
	for _, coll := range collections {
		_, ok := chainIdToCollAddrMap[coll.ChainID]
		if !ok { //不存在，则新增
			chainIdToCollAddrMap[coll.ChainID] = []string{coll.Address}
		} else { //存在，则追加
			chainIdToCollAddrMap[coll.ChainID] = append(chainIdToCollAddrMap[coll.ChainID], coll.Address)
		}
	}
	//4、并发查询每个collection的挂单数量
	var listed []entity.CollectionInfo
	var wg sync.WaitGroup
	var mu sync.Mutex
	for chainId, collectionAddrs := range chainIdToCollAddrMap {
		chain := chainIdToChainNameMap[chainId]
		wg.Add(1)
		go func(chain string, collectionAddrs []string) {
			defer wg.Done()
			//查询多个集合中已上架NFT的数量
			list, err := serverCtx.Dao.QueryListedAmountEachCollection(ctx, chain, collectionAddrs, userAddrs)
			if err != nil {
				return
			}

			//加锁
			mu.Lock()
			listed = append(listed, list...)
			mu.Unlock()

		}(chain, collectionAddrs)
	}
	//等待查询完成
	wg.Wait()

	//5、构建collection地址到挂单数量的映射
	collectionsListedMap := make(map[string]int)
	for _, li := range listed {
		collectionsListedMap[strings.ToLower(li.Address)] = li.ListAmount
	}

	//6、组装最终结果
	var result entity.UserCollectionsData
	chainInfoMap := make(map[int]entity.ChainInfo)
	for _, collection := range collections {
		//6.1 添加collection信息
		result.CollectionInfos = append(result.CollectionInfos, entity.CollectionInfo{
			ChainID:    collection.ChainID,
			Name:       collection.Name,
			Address:    collection.Address,
			Symbol:     collection.Symbol,
			ImageURI:   collection.ImageURI,
			ListAmount: collectionsListedMap[strings.ToLower(collection.Address)],
			ItemAmount: collection.ItemAmount,
			FloorPrice: collection.FloorPrice,
		})
		//6.2 添加chain信息
		chainInfo, ok := chainInfoMap[collection.ChainID]
		if ok {
			chainInfo.ItemOwned += collection.ItemCount
			chainInfo.ItemValue = chainInfo.ItemValue.Add(decimal.New(collection.ItemCount, 0).Mul(collection.FloorPrice))
			chainInfoMap[collection.ChainID] = chainInfo
		} else {
			chainInfoMap[collection.ChainID] = entity.ChainInfo{
				ChainID:   collection.ChainID,
				ItemOwned: collection.ItemCount,
				ItemValue: decimal.New(collection.ItemCount, 0).Mul(collection.FloorPrice),
			}
		}
		result.ChainInfos = append(result.ChainInfos, chainInfoMap[collection.ChainID])
	}

	return &entity.UserCollectionsResp{Result: result}, nil
}
