package service

import (
	"EasySwapBackend-test/src/dao"
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service/mq"
	"EasySwapBackend-test/src/svc"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/evm/eip"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"strings"
	"sync"
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

// GetItems 获取NFT Item列表信息：Item基本信息、订单信息、图片信息、用户持有数量、最近成交价格、最高出价信息
func GetCollectionItems(ctx context.Context, serverCtx *svc.ServerCtx, chain, collectionAddr string, filter entity.CollectionItemFilterParam) (*entity.NFTListInfoRes, error) {
	//1、查询item基础信息和订单信息
	items, count, err := serverCtx.Dao.QueryCollectionItemOrder(ctx, chain, collectionAddr, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}
	//2、提取需要查询的itemID和所有者地址
	var itemIds []string
	var itemOwners []string
	var itemPrice []entity.ItemPriceInfo
	for _, item := range items {
		if item.TokenId != "" {
			itemIds = append(itemIds, item.TokenId)
		}
		if item.Owner != "" {
			itemOwners = append(itemOwners, item.Owner)
		}
		// 记录已上架Item的价格信息
		if item.Listing {
			itemPrice = append(itemPrice, entity.ItemPriceInfo{
				CollectionAddress: item.CollectionAddress,
				TokenId:           item.TokenId,
				Maker:             item.Owner,
				Price:             item.ListPrice,
				OrderStatus:       multi.OrderStatusActive,
			})
		}
	}
	//3、并发查询各种扩展信息
	var queryErr error
	var wg sync.WaitGroup

	//3.1、查询订单详情
	ordersInfo := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemPrice) > 0 {
			orders, err := serverCtx.Dao.QueryListingInfo(ctx, chain, itemPrice)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get orders time info")
				return
			}
			for _, order := range orders {
				ordersInfo[strings.ToLower(order.CollectionAddress+order.TokenId)] = order
			}
		}
	}()
	//3.2、查询item图片信息和视频信息
	itemsExternal := make(map[string]multi.ItemExternal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemIds) > 0 {
			images, err := serverCtx.Dao.QueryCollectionItemImage(ctx, chain, collectionAddr, itemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items image info")
				return
			}
			for _, image := range images {
				itemsExternal[strings.ToLower(image.TokenId)] = image
			}
		}
	}()
	//3.3、查询用户持有数量
	userItemCount := make(map[string]int64)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemOwners) > 0 {
			userCount, err := serverCtx.Dao.QueryUserItemCount(ctx, chain, collectionAddr, itemOwners)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get user item count")
				return
			}
			for _, v := range userCount {
				userItemCount[strings.ToLower(v.Owner)] = v.Counts
			}
		}
	}()
	//3.4、查询最近成交价格
	lastSales := make(map[string]decimal.Decimal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemIds) > 0 {
			lastSalePrice, err := serverCtx.Dao.QueryLastSalePrice(ctx, chain, collectionAddr, itemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items last sale info")
				return
			}
			for _, v := range lastSalePrice {
				lastSales[v.TokenId] = v.Price
			}
		}
	}()
	//3.5、查询item级别的最高出价
	bestBids := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemIds) > 0 {
			bids, err := serverCtx.Dao.QueryBestBids(ctx, chain, collectionAddr, filter.UserAddress, itemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "")
				return
			}
			for _, bid := range bids {
				//1、先从map中获取一下，判断是否已存在
				order, ok := bestBids[strings.ToLower(bid.TokenId)]
				//若不存在，则写入map
				if !ok {
					bestBids[strings.ToLower(bid.TokenId)] = bid
					continue
				}
				//若存在，并且新的出价比map中的出价高，则更新
				if bid.Price.GreaterThan(order.Price) {
					bestBids[strings.ToLower(bid.TokenId)] = bid
				}
			}
		}
	}()
	//3.6、查询集合级别的最高出价
	var collectionBestBid multi.Order
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectionBestBid, err = serverCtx.Dao.QueryCollectionBestBid(ctx, chain, collectionAddr, filter.UserAddress)
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}
	}()
	//4、等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on get items info")
	}

	//5、整合所有信息
	var resItems []*entity.NFTListInfo
	for _, item := range items {
		//设置item名称
		itemName := item.Name
		if itemName == "" {
			itemName = fmt.Sprintf("#%s", item.TokenId)
		}
		//构建返回体
		resItem := &entity.NFTListInfo{
			Name:              itemName,
			CollectionAddress: item.CollectionAddress,
			TokenID:           item.TokenId,
			OwnerAddress:      item.Owner,
			ListPrice:         item.ListPrice,
			MarketID:          item.MarketID,
			BidExpireTime:     collectionBestBid.ExpireTime,
			BidMaker:          collectionBestBid.Maker,
			BidOrderID:        collectionBestBid.OrderID,
			BidPrice:          collectionBestBid.Price,
			BidSalt:           collectionBestBid.Salt,
			BidTime:           collectionBestBid.EventTime,
			BidType:           getBidType(collectionBestBid.OrderType),
			BidSize:           collectionBestBid.Size,
			BidUnfilled:       collectionBestBid.QuantityRemaining,
		}
		//添加订单信息
		order, ok := ordersInfo[strings.ToLower(item.CollectionAddress+item.TokenId)]
		if ok {
			resItem.ListExpireTime = order.ExpireTime
			resItem.ListOrderID = order.OrderID
			resItem.ListSalt = order.Salt
			resItem.ListTime = order.EventTime
		}
		//添加图片信息和视频信息
		itemExternal, ok := itemsExternal[strings.ToLower(item.TokenId)]
		if ok {
			//图片信息
			if itemExternal.IsUploadedOss {
				resItem.ImageURI = itemExternal.OssUri
			} else {
				resItem.ImageURI = itemExternal.ImageUri
			}
			//视频信息
			if len(itemExternal.VideoUri) > 0 {
				resItem.VideoType = itemExternal.VideoType
				if itemExternal.IsVideoUploaded {
					resItem.VideoURI = itemExternal.VideoOssUri
				} else {
					resItem.VideoURI = itemExternal.VideoUri
				}
			}
		}
		//添加用户持有数量
		ownerCount, ok := userItemCount[strings.ToLower(item.Owner)]
		if ok {
			resItem.OwnerOwnedAmount = ownerCount
		}
		//添加最近成交价格
		lastPrice, ok := lastSales[strings.ToLower(item.TokenId)]
		if ok {
			resItem.LastSellPrice = lastPrice
		}
		//添加最高出价信息
		bidOrder, ok := bestBids[strings.ToLower(item.TokenId)]
		if ok {
			if bidOrder.Price.GreaterThan(collectionBestBid.Price) {
				resItem.BidTime = bidOrder.EventTime
				resItem.BidType = getBidType(bidOrder.OrderType)
				resItem.BidUnfilled = bidOrder.QuantityRemaining
				resItem.BidSalt = bidOrder.Salt
				resItem.BidPrice = bidOrder.Price
				resItem.BidOrderID = bidOrder.OrderID
				resItem.BidMaker = bidOrder.Maker
				resItem.BidExpireTime = bidOrder.ExpireTime
				resItem.BidSize = bidOrder.Size
			}
		}
		resItems = append(resItems, resItem)
	}
	//6、包装返回结果
	return &entity.NFTListInfoRes{
		Result: resItems,
		Count:  count,
	}, nil
}

// GetItemDetail 获取单个NFT的详细信息
func GetItemDetail(ctx context.Context, serverCtx *svc.ServerCtx, chain string, chainId int, collectionAddr, tokenId string) (*entity.ItemDetailInfoResp, error) {
	var queryErr error
	var wg sync.WaitGroup

	//并发查询以下信息
	//1、查询collection信息
	var collection *multi.Collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		collection, queryErr = serverCtx.Dao.QueryCollectionInfo(ctx, chain, collectionAddr)
		if queryErr != nil {
			return
		}
	}()
	//2、查询item基本信息
	var item *multi.Item
	wg.Add(1)
	go func() {
		defer wg.Done()
		item, queryErr = serverCtx.Dao.QueryItemInfo(ctx, chain, collectionAddr, tokenId)
		if queryErr != nil {
			return
		}
	}()
	//3、查询item挂单信息
	var itemListInfo *dao.CollectionItem
	wg.Add(1)
	go func() {
		defer wg.Done()
		itemListInfo, queryErr = serverCtx.Dao.QueryItemListInfo(ctx, chain, collectionAddr, tokenId)
		if queryErr != nil {
			return
		}
	}()
	//4、查询item的图片和视频信息
	itemExternal := make(map[string]multi.ItemExternal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		images, err := serverCtx.Dao.QueryCollectionItemImage(ctx, chain, collectionAddr, []string{tokenId})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items image info")
			return
		}
		for _, image := range images {
			itemExternal[strings.ToLower(image.TokenId)] = image
		}
	}()
	//5、查询最近成交价格
	lastSales := make(map[string]decimal.Decimal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		activitys, err := serverCtx.Dao.QueryLastSalePrice(ctx, chain, collectionAddr, []string{tokenId})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get last price info")
			return
		}
		for _, v := range activitys {
			lastSales[strings.ToLower(v.TokenId)] = v.Price
		}
	}()
	//6、查询item维度最高出价
	bestBids := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		bids, err := serverCtx.Dao.QueryBestBids(ctx, chain, collectionAddr, "", []string{tokenId})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}
		for _, v := range bids {
			order, ok := bestBids[strings.ToLower(v.TokenId)]
			if !ok {
				bestBids[strings.ToLower(v.TokenId)] = order
				continue
			}
			if v.Price.GreaterThan(order.Price) {
				bestBids[strings.ToLower(v.TokenId)] = order
			}
		}
	}()
	//7、查询collection维度的最高出价
	var collectionBestBid multi.Order
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectionBestBid, queryErr = serverCtx.Dao.QueryCollectionBestBid(ctx, chain, collectionAddr, "")
		if queryErr != nil {
			return
		}
	}()
	//8、等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on get items info")
	}
	//9、组装返回数据
	var itemDetail entity.ItemDetailInfo
	itemDetail.ChainID = chainId
	if item != nil {
		itemDetail.Name = item.Name
		itemDetail.CollectionAddress = item.CollectionAddress
		itemDetail.TokenID = item.TokenId
		itemDetail.OwnerAddress = item.Owner
		//设置collection级别的最高出价信息
		itemDetail.BidMaker = collectionBestBid.Maker
		itemDetail.BidSize = collectionBestBid.Size
		itemDetail.BidExpireTime = collectionBestBid.ExpireTime
		itemDetail.BidOrderID = collectionBestBid.OrderID
		itemDetail.BidPrice = collectionBestBid.Price
		itemDetail.BidSalt = collectionBestBid.Salt
		itemDetail.BidUnfilled = collectionBestBid.QuantityRemaining
		itemDetail.BidType = getBidType(collectionBestBid.OrderType)
		itemDetail.BidTime = collectionBestBid.EventTime
	}
	// 如果item级别的最高出价大于collection级别的最高出价,则使用item级别的出价信息
	bidOrder, ok := bestBids[strings.ToLower(item.TokenId)]
	if ok {
		if bidOrder.Price.GreaterThan(collectionBestBid.Price) {
			itemDetail.BidMaker = bidOrder.Maker
			itemDetail.BidSize = bidOrder.Size
			itemDetail.BidExpireTime = bidOrder.ExpireTime
			itemDetail.BidOrderID = bidOrder.OrderID
			itemDetail.BidPrice = bidOrder.Price
			itemDetail.BidSalt = bidOrder.Salt
			itemDetail.BidUnfilled = bidOrder.QuantityRemaining
			itemDetail.BidType = getBidType(bidOrder.OrderType)
			itemDetail.BidTime = bidOrder.EventTime
		}
	}
	//设置挂单信息
	if itemListInfo != nil {
		itemDetail.ListSalt = itemListInfo.ListSalt
		itemDetail.ListMaker = itemListInfo.ListMaker
		itemDetail.ListTime = itemListInfo.ListTime
		itemDetail.ListExpireTime = itemListInfo.ListExpireTime
		itemDetail.ListOrderID = itemListInfo.OrderID
		itemDetail.ListPrice = itemListInfo.ListPrice
		itemDetail.MarketplaceID = itemListInfo.MarketID
	}
	//设置collection信息
	if collection != nil {
		itemDetail.CollectionName = collection.Name
		itemDetail.CollectionImageURI = collection.ImageUri
		itemDetail.FloorPrice = collection.FloorPrice
		if itemDetail.Name == "" {
			itemDetail.Name = fmt.Sprintf("%s #%s", collection.Name, tokenId)
		}
	}
	//设置最近成交价格
	price, ok := lastSales[strings.ToLower(tokenId)]
	if ok {
		itemDetail.LastSellPrice = price
	}
	//设置图片和视频信息
	external, ok := itemExternal[strings.ToLower(tokenId)]
	if ok {
		//图片信息
		if external.IsUploadedOss {
			itemDetail.ImageURI = external.OssUri
		} else {
			itemDetail.ImageURI = external.ImageUri
		}
		//视频信息
		if len(external.VideoUri) > 0 {
			itemDetail.VideoType = external.VideoType
			if external.IsVideoUploaded {
				itemDetail.VideoURI = external.VideoOssUri
			} else {
				itemDetail.VideoURI = external.VideoUri
			}
		}
	}
	return &entity.ItemDetailInfoResp{
		Result: itemDetail,
	}, nil
}

// GetItemTraits 获取NFT的 Trait信息
// 主要功能:
// 1. 并发查询三个信息:
//   - NFT的 Trait信息
//   - 集合中每个 Trait的数量统计
//   - 集合基本信息
//
// 2. 计算每个 Trait的百分比
// 3. 组装返回数据
func GetItemTraits(ctx context.Context, serverCtx *svc.ServerCtx, chain string, collectionAddr, tokenId string) (*entity.ItemTraitsResp, error) {
	var traitInfos []entity.TraitInfo
	var queryErr error
	var wg sync.WaitGroup

	//1、查询item的trait信息
	var itemTraits []multi.ItemTrait
	wg.Add(1)
	go func() {
		defer wg.Done()
		itemTraits, queryErr = serverCtx.Dao.QueryItemTraits(ctx, chain, collectionAddr, tokenId)
		if queryErr != nil {
			return
		}
	}()
	//2、查询collection维度的trait统计信息
	var traitCounts []entity.TraitCount
	wg.Add(1)
	go func() {
		defer wg.Done()
		traitCounts, queryErr = serverCtx.Dao.QueryCollectionTraitCount(ctx, chain, collectionAddr)
		if queryErr != nil {
			return
		}
	}()
	//3、查询集合信息
	var collection *multi.Collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		collection, queryErr = serverCtx.Dao.QueryCollectionInfo(ctx, chain, collectionAddr)
		if queryErr != nil {
			return
		}
	}()
	//4、等待查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, queryErr
	}
	//5、如果item没有trait信息，则返回空
	if len(itemTraits) == 0 {
		return &entity.ItemTraitsResp{
			Result: traitInfos,
		}, nil
	}
	//6、构建trait数量映射
	traitCountMap := make(map[string]int64)
	for _, trait := range traitCounts {
		traitCountMap[fmt.Sprintf("%s-%s", trait.Trait, trait.TraitValue)] = trait.Count
	}
	//7、计算每个 Trait的百分比并组装返回数据
	for _, trait := range itemTraits {
		key := fmt.Sprintf("%s-%s", trait.Trait, trait.TraitValue)
		count, ok := traitCountMap[key]
		if ok {
			traitPercent := 0.0
			if collection.ItemAmount != 0 {
				traitPercent = decimal.NewFromInt(count).
					DivRound(decimal.NewFromInt(collection.ItemAmount), 4).
					Mul(decimal.NewFromInt(100)).
					InexactFloat64()
			}
			traitInfos = append(traitInfos, entity.TraitInfo{
				Trait:        trait.Trait,
				TraitValue:   trait.TraitValue,
				TraitAmount:  count,
				TraitPercent: traitPercent,
			})
		}
	}
	return &entity.ItemTraitsResp{
		Result: traitInfos,
	}, nil
}

// 获取指定 token ids的Trait的最高价格信息
func GetItemTopTraitPrice(ctx context.Context, serverCtx *svc.ServerCtx, chain, collectionAddr string, tokenIds []string) (*entity.ItemTopTraitResp, error) {
	var res []entity.TraitPrice

	//1、查询trait对应的最低挂单价格列表
	traitPrice, err := serverCtx.Dao.QueryTraitPrice(ctx, chain, collectionAddr, tokenIds)
	if err != nil {
		return nil, errors.Wrap(err, "failed on calc top trait")
	}
	if len(traitPrice) == 0 {
		return &entity.ItemTopTraitResp{
			Result: res,
		}, nil
	}
	//2、构建trait ->最低挂单价格映射
	traitPriceMap := make(map[string]decimal.Decimal)
	for _, v := range traitPrice {
		traitPriceMap[strings.ToLower(fmt.Sprintf("%s-%s", v.Trait, v.TraitValue))] = v.Price
	}
	//3、查询指定tokenids的所有trait
	itemsTraits, err := serverCtx.Dao.QueryItemsTraits(ctx, chain, collectionAddr, tokenIds)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items trait")
	}
	//4、计算指定 token ids的 最高价值 Trait
	topTraitsMap := make(map[string]entity.TraitPrice)
	for _, trait := range itemsTraits {
		key := strings.ToLower(fmt.Sprintf("%s-%s", trait.Trait, trait.TraitValue))
		price, ok := traitPriceMap[key]
		if ok {
			topPrice, ok := topTraitsMap[trait.TokenId]
			// 如果已有最高价且当前价格不高于最高价,跳过
			if ok && price.LessThanOrEqual(topPrice.Price) {
				continue
			}
			//否则，更新最高价值 Trait
			topTraitsMap[trait.TokenId] = entity.TraitPrice{
				CollectionAddress: collectionAddr,
				TokenID:           trait.TokenId,
				Trait:             trait.Trait,
				TraitValue:        trait.TraitValue,
				Price:             price,
			}
		}
	}
	//5、包装返回结果
	for _, topTrait := range topTraitsMap {
		res = append(res, topTrait)
	}
	return &entity.ItemTopTraitResp{
		Result: res,
	}, nil
}

// 获取NFT Item的图片信息
func GetItemImage(ctx context.Context, serverCtx *svc.ServerCtx, chain string, collectionAddr, tokenId string) (*entity.ItemImage, error) {
	items, err := serverCtx.Dao.QueryCollectionItemImage(ctx, chain, collectionAddr, []string{tokenId})
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item image")
	}
	var imageUri string
	if items[0].IsUploadedOss {
		imageUri = items[0].OssUri
	} else {
		imageUri = items[0].ImageUri
	}
	return &entity.ItemImage{
		CollectionAddress: collectionAddr,
		TokenID:           tokenId,
		ImageUri:          imageUri,
	}, nil
}

// NFT历史销售价格列表
func GetHistorySalesPrice(ctx context.Context, serverCtx *svc.ServerCtx, chain string, collectionAddr, duration string) (*entity.HistorySalesPriceRes, error) {
	var res []entity.HistorySalesPriceInfo
	var durationTimeStamp int64
	if duration == "24h" {
		durationTimeStamp = 24 * 60 * 60
	} else if duration == "7d" {
		durationTimeStamp = 7 * 24 * 60 * 60
	} else if duration == "30d" {
		durationTimeStamp = 30 * 24 * 60 * 60
	} else {
		return nil, errors.New("only support 24h/7d/30d")
	}
	//查询数据库
	historySalesPriceInfo, err := serverCtx.Dao.QueryHistorySalesPriceInfo(ctx, chain, collectionAddr, durationTimeStamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get history sales price info")
	}
	for _, v := range historySalesPriceInfo {
		res = append(res, entity.HistorySalesPriceInfo{
			Price:     v.Price,
			TokenID:   v.TokenId,
			TimeStamp: v.EventTime,
		})
	}
	return &entity.HistorySalesPriceRes{
		Result: res,
	}, nil
}

// 获取NFT所有者信息
func GetItemOwner(ctx context.Context, serverCtx *svc.ServerCtx, chainId int64, chain, collectionAddr, tokenId string) (*entity.ItemOwner, error) {
	//1、从链上获取NFT所有者地址
	ownerAddr, err := serverCtx.NodeSrvs[chainId].FetchNftOwner(collectionAddr, tokenId)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on fetch nft owner onchain", zap.Error(err))
		return nil, errors.Wrap(err, "failed on fetch nft owner onchain")
	}
	//2、校验地址并转换格式
	owner, err := eip.ToCheckSumAddress(ownerAddr.String())
	if err != nil {
		xzap.WithContext(ctx).Error("invalid address", zap.Error(err), zap.String("address", ownerAddr.String()))
		return nil, errors.Wrap(err, "invalid address")
	}
	//3、更新数据库中NFT所有者信息
	err = serverCtx.Dao.UpdateItemOwner(ctx, chain, collectionAddr, tokenId, owner)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on update item owner", zap.Error(err), zap.String("address", ownerAddr.String()))
	}
	//4、返回NFT所有者信息
	return &entity.ItemOwner{
		CollectionAddress: collectionAddr,
		TokenID:           tokenId,
		Owner:             owner,
	}, nil
}

// 刷新NFT的元数据信息
func RefreshItemMetadata(ctx context.Context, serverCtx *svc.ServerCtx, chainId int64, chain, collectionAddr, tokenId string) error {
	err := mq.AddSingleItemToRefreshMetadataQueue(serverCtx.KvStore, serverCtx.C.ProjectCfg.Name, chain, chainId, collectionAddr, tokenId)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on add item to refresh queue", zap.Error(err),
			zap.String("collectionAddress", collectionAddr), zap.String("tokenId", tokenId))
		return errors.Wrap(err, "failed on add item to refresh queue")
	}
	return nil
}
