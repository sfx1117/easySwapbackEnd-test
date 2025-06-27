package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// 查询指定collection详情数据
func GetCollectionDetail(ctx context.Context, serverCtx *svc.ServerCtx, chain, address string) (*entity.CollectionDetailRes, error) {
	//1、查询指定链上的NFT集合信息
	collectionInfo, err := serverCtx.Dao.QueryCollectionInfo(ctx, chain, address)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}

	//2、获取集合24小时内交易信息
	tradeInfos, err := serverCtx.Dao.GetTradeInfoByCollection(chain, address, "1d")
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection trade info", zap.Error(err))
		//return nil, errcode.NewCustomErr("cache error")
	}
	//2.1 获取24小时交易量和销售数量
	var volume24h decimal.Decimal
	var sold int64
	if tradeInfos != nil {
		volume24h = tradeInfos.Volume
		sold = tradeInfos.ItemCount
	}

	//3、查询上架数量
	listedAmount, err := serverCtx.Dao.QueryListedAmount(ctx, chain, address)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get listed count", zap.Error(err))
		//return nil, errcode.NewCustomErr("cache error")
	} else {
		//缓存集合的上架数量
		err := serverCtx.Cached.CacheCollectionsListed(chain, address, int(listedAmount))
		if err != nil {
			xzap.WithContext(ctx).Error("failed on cache collection listed", zap.Error(err))
		}
	}

	//4、查询指定集合地板价
	floorPrice, err := serverCtx.Dao.QueryFloorPrice(ctx, chain, address)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get floor price", zap.Error(err))
	}
	//4.1、如果地板价发生变化，则更新价格事件
	if !floorPrice.Equals(collectionInfo.FloorPrice) {
		err := ordermanager.AddUpdatePriceEvent(serverCtx.KvStore, &ordermanager.TradeEvent{
			EventType:      ordermanager.UpdateCollection,
			CollectionAddr: address,
			Price:          floorPrice,
		}, chain)
		if err != nil {
			xzap.WithContext(ctx).Error("failed on update floor price", zap.Error(err))
		}
	}

	//5、查询指定集合最高卖单价格
	collectionSell, err := serverCtx.Dao.QueryCollectionSellPrice(ctx, chain, address)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get floor price", zap.Error(err))
	}

	//6、查询指定集合的总交易量
	var allVol decimal.Decimal
	collectionVolume, err := serverCtx.Dao.QueryCollectionVolume(ctx, chain, address)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on query collection all volume", zap.Error(err))
	} else {
		allVol = collectionVolume
	}

	//7、构建返回结果
	detail := entity.CollectionDetail{
		ImageUri:    collectionInfo.ImageUri, // svcCtx.ImageMgr.GetFileUrl(collection.ImageUri),
		Name:        collectionInfo.Name,
		Address:     collectionInfo.Address,
		ChainId:     collectionInfo.ChainId,
		FloorPrice:  floorPrice,
		SellPrice:   collectionSell.SalePrice.String(),
		VolumeTotal: allVol,
		Volume24h:   volume24h,
		Sold24h:     sold,
		ListAmount:  listedAmount,
		TotalSupply: collectionInfo.ItemAmount,
		OwnerAmount: collectionInfo.OwnerAmount,
	}

	return &entity.CollectionDetailRes{
		Result: detail,
	}, nil
}

// 分页查询指定Collection的bids信息
func GetBids(ctx context.Context, serverCtx *svc.ServerCtx, chain, collectionAddr string, page, pageSize int) (*entity.CollectionBidsRes, error) {
	bids, count, err := serverCtx.Dao.QueryCollectionBids(ctx, chain, collectionAddr, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}
	return &entity.CollectionBidsRes{
		Result: bids,
		Count:  count,
	}, nil
}
