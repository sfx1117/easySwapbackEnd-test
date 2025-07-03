package mq

import (
	"EasySwapBackend-test/src/entity"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const CacheRefreshPreventReentranceKeyPrefix = "cache:es:item:refresh:prevent:reentrancy:%d:%s:%s"
const CacheRefreshSingleItemMetadataKey = "cache:%s:%s:item:refresh:metadata"
const PreventReentrancyPeriod = 10 //second

func AddSingleItemToRefreshMetadataQueue(kvStore *xkv.Store, project, chain string, chainId int64,
	collectionAddr, tokenId string) error {
	//1、从redis中获取
	reentKey := fmt.Sprintf(CacheRefreshPreventReentranceKeyPrefix, chainId, collectionAddr, chain)
	isRefreshed, err := kvStore.Get(reentKey)
	if err != nil {
		return errors.Wrap(err, "failed on check reentrancy status")
	}
	if isRefreshed != "" {
		xzap.WithContext(context.Background()).Info("refresh within 10s",
			zap.String("collectionAddr", collectionAddr), zap.String("tokenId", tokenId))
		return nil
	}
	//构建参数
	item := entity.RefreshItem{
		ChainId:           chainId,
		CollectionAddress: collectionAddr,
		TokenId:           tokenId,
	}
	rawInfo, err := json.Marshal(&item)
	if err != nil {
		return errors.Wrap(err, "failed on marshal item info")
	}
	//向reids中写入数据
	metadataKey := fmt.Sprintf(CacheRefreshSingleItemMetadataKey, project, chain)
	_, err = kvStore.Sadd(metadataKey, string(rawInfo))
	if err != nil {
		return errors.Wrap(err, "failed on push item to refresh metadata queue")
	}
	_ = kvStore.Setex(reentKey, "true", PreventReentrancyPeriod)
	return nil
}
