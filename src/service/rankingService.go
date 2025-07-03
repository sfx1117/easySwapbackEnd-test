package service

import (
	"EasySwapBackend-test/src/dao"
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
)

const MinuteSeconds = 60
const HourSeconds = 60 * 60
const DaySeconds = 3600 * 24

// 获取指定链上的NFT集合排名信息
func GetTopRanking(ctx context.Context, serverCtx *svc.ServerCtx, chain string, period string, limit int64) ([]*entity.CollectionRankingInfo, error) {
	//1、获取集合交易排行榜信息
	collectionTradeInfos, err := serverCtx.Dao.GetCollectionRankingByActivity(chain, period)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection trade info", zap.Error(err))
	}
	//将数据构建成map结构
	collectionTradeMap := make(map[string]dao.CollectionTrade)
	for _, trade := range collectionTradeInfos {
		collectionTradeMap[strings.ToLower(trade.ContractAddress)] = *trade
	}

	//2、时间段映射
	periodTime := map[string]int64{
		"15m": MinuteSeconds * 15,
		"1h":  HourSeconds,
		"6h":  HourSeconds * 6,
		"1d":  DaySeconds,
		"7d":  DaySeconds * 7,
		"30d": DaySeconds * 30,
	}

	//3、获取地板价变化信息
	collectionFloorChangeMap, err := serverCtx.Dao.QueryCollectionFloorChange(chain, periodTime[period])
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection floor change", zap.Error(err))
	}
	// 并发控制
	var wg sync.WaitGroup
	var queryErr error

	//4.1、并发获取集合销售价格信息
	collectionSellsMap := make(map[string]multi.Collection)
	wg.Add(1)
	go func() {
		defer wg.Done()
		collections, err := serverCtx.Dao.QueryCollectionsSellPrice(ctx, chain)
		if err != nil {
			xzap.WithContext(ctx).Error("failed on get all collections info", zap.Error(err))
			queryErr = errcode.NewCustomErr("failed on get all collections info")
			return
		}
		for _, collection := range collections {
			collectionSellsMap[strings.ToLower(collection.Address)] = collection
		}
	}()
	//4.2、并发获取所有集合的基本信息
	var allCollection []multi.Collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		allCollection, err = serverCtx.Dao.QueryAllCollectionInfo(ctx, chain)
		if err != nil {
			xzap.WithContext(ctx).Error("failed on get all collections info", zap.Error(err))
			queryErr = errcode.NewCustomErr("failed on get all collections info")
			return
		}
	}()
	//等待所有查询结束
	wg.Wait()
	if queryErr != nil {
		return nil, queryErr
	}

	//5、构建返回结果
	var result []*entity.CollectionRankingInfo
	for _, collection := range allCollection {
		var floorPriceChange float64
		var volume decimal.Decimal
		var sellPrice decimal.Decimal
		var sales int64
		//获取集合交易信息
		trade, ok := collectionTradeMap[strings.ToLower(collection.Address)]
		if ok {
			floorPriceChange = collectionFloorChangeMap[strings.ToLower(collection.Address)]
			volume = trade.Volume
			sales = trade.ItemCount
		}
		//获取集合销售价格信息
		sellInfo, ok := collectionSellsMap[strings.ToLower(collection.Address)]
		if ok {
			sellPrice = sellInfo.SalePrice
		}
		//获取上架数量
		var listAmount int
		listed, err := serverCtx.Dao.QueryCollectionsListed(chain, []string{collection.Address})
		if err != nil {
			xzap.WithContext(ctx).Error("failed on query collection listed", zap.Error(err))
		} else {
			listAmount = listed[0].Count
		}
		//构建单个集合的排名信息
		result = append(result, &entity.CollectionRankingInfo{
			Name:        collection.Name,
			Address:     collection.Address,
			ImageUri:    collection.ImageUri,
			FloorPrice:  collection.FloorPrice.String(),
			FloorChange: strconv.FormatFloat(floorPriceChange, 'f', 4, 32),
			SellPrice:   sellPrice.String(),
			Volume:      volume,
			ItemSold:    sales,
			ItemNum:     collection.ItemAmount,
			ItemOwner:   collection.OwnerAmount,
			ListAmount:  listAmount,
			ChainID:     collection.ChainId,
		})
	}
	//6、限制返回数量
	if limit < int64(len(result)) {
		result = result[:limit]
	}
	return result, err
}
