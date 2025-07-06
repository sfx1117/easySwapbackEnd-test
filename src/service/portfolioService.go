package service

import (
	"EasySwapBackend-test/src/dao"
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"EasySwapBackend-test/src/utils"
	"context"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"sort"
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

// 查询用户拥有nft的Item基本信息，list信息和bid信息，从Item表和Activity表中查询
func GetMultiChainUserItems(ctx context.Context, serverCtx *svc.ServerCtx, chainNames []string, filter entity.PortfolioMultiChainItemFilterParams) (*entity.UserItemsResp, error) {
	//解析入参
	chainIds := filter.ChainID
	userAddrs := filter.UserAddresses
	collAddrs := filter.CollectionAddresses
	page := filter.Page
	pageSize := filter.PageSize

	//1、查询用户拥有nft的Item基本信息
	items, total, err := serverCtx.Dao.QueryMultiChainUserItemInfos(ctx, chainNames, userAddrs, collAddrs, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user items info")
	}
	//如果没有total，直接返回空结果
	if total == 0 {
		return &entity.UserItemsResp{Result: items, Count: total}, nil
	}

	//2、构建chainId到chainName的映射
	chainIdToChainNameMap := make(map[int]string)
	for i, chainId := range chainIds {
		chainIdToChainNameMap[chainId] = chainNames[i]
	}

	//3、准备查询参数
	var collectionAddrs [][]string                              // Collection地址和链名称对
	var itemInfos []entity.MultiChainItemInfo                   // Item信息
	var chainCollectionsMap = make(map[string][]string)         // 按链分组的Collection地址
	var multichainItemsMap = make(map[string][]entity.ItemInfo) // 按链分组的Item信息

	// 遍历Item,构建查询参数
	for _, item := range items {
		collectionAddrs = append(collectionAddrs, []string{strings.ToLower(item.CollectionAddress), chainIdToChainNameMap[item.ChainID]})
		itemInfos = append(itemInfos, entity.MultiChainItemInfo{
			ItemInfo: entity.ItemInfo{
				CollectionAddress: item.CollectionAddress,
				TokenID:           item.TokenID,
			},
			ChainName: chainIdToChainNameMap[item.ChainID],
		})
		chainCollectionsMap[strings.ToLower(chainIdToChainNameMap[item.ChainID])] = append(chainCollectionsMap[strings.ToLower(chainIdToChainNameMap[item.ChainID])], item.CollectionAddress)
		multichainItemsMap[strings.ToLower(chainIdToChainNameMap[item.ChainID])] = append(multichainItemsMap[strings.ToLower(chainIdToChainNameMap[item.ChainID])], entity.ItemInfo{
			CollectionAddress: item.CollectionAddress,
			TokenID:           item.TokenID,
		})
	}

	//4、获取用户地址
	var userAddr string
	if len(userAddrs) > 0 {
		userAddr = userAddrs[0]
	} else {
		userAddr = ""
	}

	//5、并发查询collection最高出价信息
	collectionBestBids := make(map[entity.MultichainCollection]multi.Order)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var queryErr error
	for chain, collAddr := range chainCollectionsMap {
		wg.Add(1)
		go func(chain string, collAddrs []string) {
			defer wg.Done()
			bestBids, err := serverCtx.Dao.QueryCollectionsBestBid(ctx, chain, userAddr, collAddrs)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query collections best bids")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, bid := range bestBids {
				collectionBestBids[entity.MultichainCollection{
					CollectionAddress: bid.CollectionAddress,
					Chain:             chain,
				}] = *bid
			}
		}(chain, collAddr)
	}
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(err, "failed on query collection bids")
	}

	//6、并发查询item的最高出价信息
	itemBestBids := make(map[entity.MultiChainItemInfo]multi.Order)
	wg.Add(1)
	for chain, item := range multichainItemsMap {
		go func(chain string, items []entity.ItemInfo) {
			defer wg.Done()
			bestBids, err := serverCtx.Dao.QueryItemsBestBids(ctx, chain, userAddr, items)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query items best bids")
				return
			}

			mu.Lock()
			defer mu.Unlock()
			for _, bid := range bestBids {
				//先查是否map中已存在
				order, ok := itemBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
					CollectionAddress: bid.CollectionAddress,
					TokenID:           bid.TokenId,
				}, ChainName: chain}]
				//不存在，则新增
				if !ok {
					itemBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
						CollectionAddress: bid.CollectionAddress,
						TokenID:           bid.TokenId,
					}, ChainName: chain}] = bid
					continue
				}
				//若存在，并且新的price>map中的price，则更新
				if bid.Price.GreaterThan(order.Price) {
					itemBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
						CollectionAddress: bid.CollectionAddress,
						TokenID:           bid.TokenId,
					}, ChainName: chain}] = bid
				}
			}
		}(chain, item)
	}
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(err, "failed on query item bids")
	}

	//7、查询collection信息
	collections, err := serverCtx.Dao.QueryMultiChainCollectionsInfo(ctx, collectionAddrs)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collections info")
	}
	collectionsMap := make(map[string]multi.Collection)
	for _, coll := range collections {
		collectionsMap[strings.ToLower(coll.Address)] = coll
	}

	//8、查询item挂单信息
	listings, err := serverCtx.Dao.QueryMultiChainUserItemsListInfo(ctx, userAddrs, itemInfos)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query item list info")
	}
	listingsMap := make(map[string]*dao.CollectionItem)
	for _, list := range listings {
		listingsMap[strings.ToLower(list.CollectionAddress+list.TokenId)] = list
	}

	//9、获取挂单价格信息
	var itemPrice []entity.MultiChainItemPriceInfo
	for _, item := range listingsMap {
		if item.Listing {
			itemPrice = append(itemPrice, entity.MultiChainItemPriceInfo{
				ItemPriceInfo: entity.ItemPriceInfo{
					CollectionAddress: item.CollectionAddress,
					TokenId:           item.TokenId,
					Maker:             item.Owner,
					Price:             item.ListPrice,
					OrderStatus:       multi.OrderStatusActive},
				ChainName: chainIdToChainNameMap[item.ChainId],
			})
		}
	}

	//10、获取挂单订单信息
	ordersMap := make(map[string]multi.Order)
	if len(itemPrice) > 0 {
		orders, err := serverCtx.Dao.QueryMultiChainListingInfo(ctx, itemPrice)
		if err != nil {
			return nil, errors.Wrap(err, "failed on query item order id")
		}
		for _, order := range orders {
			ordersMap[strings.ToLower(order.CollectionAddress+order.TokenId)] = order
		}
	}

	//11、查询item图片信息
	itemExternals, err := serverCtx.Dao.QueryMultiChainCollectionsItemsImage(ctx, itemInfos)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query item image info")
	}
	itemExternalMap := make(map[string]multi.ItemExternal)
	for _, item := range itemExternals {
		itemExternalMap[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
	}

	//12、组装最终结果
	for _, item := range items {
		//设置出价信息
		bidOrder, ok := itemBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
			CollectionAddress: item.CollectionAddress,
			TokenID:           item.TokenID,
		}, ChainName: chainIdToChainNameMap[item.ChainID]}]
		if ok {
			if bidOrder.Price.GreaterThan(collectionBestBids[entity.MultichainCollection{
				CollectionAddress: item.CollectionAddress,
				Chain:             chainIdToChainNameMap[item.ChainID],
			}].Price) {
				item.BidOrderID = bidOrder.OrderID
				item.BidExpireTime = bidOrder.ExpireTime
				item.BidPrice = bidOrder.Price
				item.BidTime = bidOrder.EventTime
				item.BidSalt = bidOrder.Salt
				item.BidMaker = bidOrder.Maker
				item.BidType = getBidType(bidOrder.OrderType)
				item.BidSize = bidOrder.Size
				item.BidUnfilled = bidOrder.QuantityRemaining
			}
		} else {
			cBid, ok := collectionBestBids[entity.MultichainCollection{
				CollectionAddress: item.CollectionAddress,
				Chain:             chainIdToChainNameMap[item.ChainID],
			}]
			if ok {
				item.BidOrderID = cBid.OrderID
				item.BidExpireTime = cBid.ExpireTime
				item.BidPrice = cBid.Price
				item.BidTime = cBid.EventTime
				item.BidSalt = cBid.Salt
				item.BidMaker = cBid.Maker
				item.BidType = getBidType(cBid.OrderType)
				item.BidSize = cBid.Size
				item.BidUnfilled = cBid.QuantityRemaining
			}
		}
		//设置collection信息
		collection, ok := collectionsMap[strings.ToLower(item.CollectionAddress)]
		if ok {
			item.CollectionName = collection.Name
			item.FloorPrice = collection.FloorPrice
			item.CollectionImageURI = collection.ImageUri
			if item.Name == "" {
				item.Name = fmt.Sprintf("%s #%s", collection.Name, item.TokenID)
			}
		}
		//设置挂单信息
		listing, ok := listingsMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			item.ListPrice = listing.ListPrice
			item.Listing = listing.Listing
			item.MarketplaceID = listing.MarketID
		}
		//设置挂单订单信息
		order, ok := ordersMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			item.ListOrderID = order.OrderID
			item.ListExpireTime = order.ExpireTime
			item.ListTime = order.EventTime
			item.ListMaker = order.Maker
			item.ListSalt = order.Salt
		}
		//设置item图片信息
		image, ok := itemExternalMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			if image.IsUploadedOss {
				item.ImageURI = image.OssUri
			} else {
				item.ImageURI = image.ImageUri
			}
		}
	}
	//14 包装返回参数
	return &entity.UserItemsResp{
		Result: items,
		Count:  total,
	}, nil
}

// 获取用户在多条链上的挂单信息
func GetMultiChainUserListings(ctx context.Context, serverCtx *svc.ServerCtx, chainNames []string, filter entity.PortfolioMultiChainListingFilterParams) (*entity.UserListingsResp, error) {
	var result []entity.Listing
	//解析入参
	chainIds := filter.ChainID
	userAddrs := filter.UserAddresses
	collAddrs := filter.CollectionAddresses
	page := filter.Page
	pageSize := filter.PageSize

	//1、查询用户挂单的基本信息
	items, count, err := serverCtx.Dao.QueryMultiChainUserListingItemInfos(ctx, chainNames, userAddrs, collAddrs, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get user items info")
	}
	// 如果没有挂单,直接返回空结果
	if count == 0 {
		return &entity.UserListingsResp{
			Result: result,
			Count:  count,
		}, nil
	}
	//2、 构建chainId到chainName的映射关系
	chainIdToChainNameMap := make(map[int]string)
	for i, _ := range chainIds {
		chainIdToChainNameMap[chainIds[i]] = chainNames[i]
	}
	//3、获取用户地址
	var userAddr string
	if len(userAddrs) > 0 {
		userAddr = userAddrs[0]
	} else {
		userAddr = ""
	}
	//4、准备查询参数
	var collectionAddrs [][]string                           // Collection地址和链名称对
	var itemInfos []entity.MultiChainItemInfo                // Item信息
	var chainCollections = make(map[string][]string)         // 按链分组的Collection地址
	var multichainItems = make(map[string][]entity.ItemInfo) // 按链分组的Item信息

	for _, item := range items {
		collectionAddrs = append(collectionAddrs, []string{strings.ToLower(item.CollectionAddress), chainIdToChainNameMap[item.ChainID]})
		itemInfos = append(itemInfos, entity.MultiChainItemInfo{
			ItemInfo: entity.ItemInfo{
				CollectionAddress: item.CollectionAddress,
				TokenID:           item.TokenID,
			},
			ChainName: chainIdToChainNameMap[item.ChainID],
		})

		chainCollections[strings.ToLower(chainIdToChainNameMap[item.ChainID])] = append(chainCollections[strings.ToLower(chainIdToChainNameMap[item.ChainID])], item.CollectionAddress)
		multichainItems[chainIdToChainNameMap[item.ChainID]] = append(multichainItems[chainIdToChainNameMap[item.ChainID]], entity.ItemInfo{
			CollectionAddress: item.CollectionAddress,
			TokenID:           item.TokenID,
		})
	}

	//5、记录item最近成本
	itemLastCost := make(map[entity.MultiChainItemInfo]decimal.Decimal)

	//6、并发查询collection最高出价信息
	collectionBestBids := make(map[entity.MultichainCollection]multi.Order)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var queryErr error

	for chain, collAddrs := range chainCollections {
		wg.Add(1)
		go func(chain string, collAddrs []string) {
			defer wg.Done()
			bestBids, err := serverCtx.Dao.QueryCollectionsBestBid(ctx, chain, userAddr, collAddrs)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query collections best bid")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, bid := range bestBids {
				collectionBestBids[entity.MultichainCollection{
					CollectionAddress: bid.CollectionAddress,
					Chain:             chain,
				}] = *bid
			}
		}(chain, collAddrs)
	}
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(err, "failed on query collection bids")
	}

	//7、并发查询item最高出价信息
	itemsBestBids := make(map[entity.MultiChainItemInfo]multi.Order)
	for chain, items := range multichainItems {
		wg.Add(1)
		go func(chain string, items []entity.ItemInfo) {
			defer wg.Done()
			bids, err := serverCtx.Dao.QueryItemsBestBids(ctx, chain, userAddr, items)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on query items best bid")
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, bid := range bids {
				//先查是否map中已存在
				order, ok := itemsBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
					CollectionAddress: bid.CollectionAddress,
					TokenID:           bid.TokenId,
				}, ChainName: chain}]
				//不存在，则新增
				if !ok {
					itemsBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
						CollectionAddress: bid.CollectionAddress,
						TokenID:           bid.TokenId,
					}, ChainName: chain}] = bid
					continue
				}
				//若存在，并且新的price>map中的price，则更新
				if bid.Price.GreaterThan(order.Price) {
					itemsBestBids[entity.MultiChainItemInfo{ItemInfo: entity.ItemInfo{
						CollectionAddress: bid.CollectionAddress,
						TokenID:           bid.TokenId,
					}, ChainName: chain}] = bid
				}
			}
		}(chain, items)
	}
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(err, "failed on query item bids")
	}

	//8、获取collection基本信息
	collections, err := serverCtx.Dao.QueryMultiChainCollectionsInfo(ctx, collectionAddrs)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collections info")
	}
	collcetionInfoMap := make(map[string]multi.Collection)
	for _, coll := range collections {
		collcetionInfoMap[strings.ToLower(coll.Address)] = coll
	}

	//9、获取用户item挂单信息
	listings, err := serverCtx.Dao.QueryMultiChainUserItemsExpireListInfo(ctx, userAddrs, itemInfos)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query item list info")
	}
	listingInfoMap := make(map[string]*dao.CollectionItem)
	for _, list := range listings {
		listingInfoMap[strings.ToLower(list.CollectionAddress+list.TokenId)] = list
	}

	//10、查询挂单订单信息
	var itemPrice []entity.MultiChainItemPriceInfo
	for _, item := range listingInfoMap {
		if item.Listing {
			itemPrice = append(itemPrice, entity.MultiChainItemPriceInfo{
				ItemPriceInfo: entity.ItemPriceInfo{
					CollectionAddress: item.CollectionAddress,
					TokenId:           item.TokenId,
					Maker:             item.Owner,
					Price:             item.ListPrice,
					OrderStatus:       item.OrderStatus,
				},
				ChainName: chainIdToChainNameMap[item.ChainId],
			})
		}
	}

	orderIdMap := make(map[string]multi.Order)
	if len(itemPrice) > 0 {
		orders, err := serverCtx.Dao.QueryMultiChainListingInfo(ctx, itemPrice)
		if err != nil {
			return nil, errors.Wrap(err, "failed on query item order id")
		}
		for _, order := range orders {
			orderIdMap[strings.ToLower(order.CollectionAddress+order.TokenId)] = order
		}
	}

	//11、查询item图片信息
	itemExternals, err := serverCtx.Dao.QueryMultiChainCollectionsItemsImage(ctx, itemInfos)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query item external info")
	}
	itemExternalMap := make(map[string]multi.ItemExternal)
	for _, item := range itemExternals {
		itemExternalMap[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
	}

	//12、组装最终结果
	for _, item := range items {
		var resultlisting entity.Listing
		listing, ok := listingInfoMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			resultlisting.ListPrice = listing.ListPrice
			resultlisting.MarketplaceID = listing.MarketID
		} else {
			count--
			continue
		}
		resultlisting.ChainID = item.ChainID
		resultlisting.CollectionAddress = item.CollectionAddress
		resultlisting.TokenID = item.TokenID
		resultlisting.LastCostPrice = itemLastCost[entity.MultiChainItemInfo{
			ItemInfo: entity.ItemInfo{
				CollectionAddress: item.CollectionAddress,
				TokenID:           item.TokenID,
			},
			ChainName: chainIdToChainNameMap[item.ChainID],
		}]

		// 设置出价信息 - 优先使用Item出价,如果没有则使用Collection出价
		bidOrder, ok := itemsBestBids[entity.MultiChainItemInfo{
			ItemInfo: entity.ItemInfo{
				CollectionAddress: strings.ToLower(item.CollectionAddress),
				TokenID:           item.TokenID,
			},
			ChainName: chainIdToChainNameMap[item.ChainID],
		}]
		if ok {
			if bidOrder.Price.GreaterThan(collectionBestBids[entity.MultichainCollection{
				CollectionAddress: strings.ToLower(item.CollectionAddress),
				Chain:             chainIdToChainNameMap[item.ChainID],
			}].Price) {
				resultlisting.BidOrderID = bidOrder.OrderID
				resultlisting.BidExpireTime = bidOrder.ExpireTime
				resultlisting.BidPrice = bidOrder.Price
				resultlisting.BidTime = bidOrder.EventTime
				resultlisting.BidSalt = bidOrder.Salt
				resultlisting.BidMaker = bidOrder.Maker
				resultlisting.BidType = getBidType(bidOrder.OrderType)
				resultlisting.BidSize = bidOrder.Size
				resultlisting.BidUnfilled = bidOrder.QuantityRemaining
			}
		} else {
			bidOrder, ok := collectionBestBids[entity.MultichainCollection{
				CollectionAddress: strings.ToLower(item.CollectionAddress),
				Chain:             chainIdToChainNameMap[item.ChainID],
			}]

			if ok {
				resultlisting.BidOrderID = bidOrder.OrderID
				resultlisting.BidExpireTime = bidOrder.ExpireTime
				resultlisting.BidPrice = bidOrder.Price
				resultlisting.BidTime = bidOrder.EventTime
				resultlisting.BidSalt = bidOrder.Salt
				resultlisting.BidMaker = bidOrder.Maker
				resultlisting.BidType = getBidType(bidOrder.OrderType)
				resultlisting.BidSize = bidOrder.Size
				resultlisting.BidUnfilled = bidOrder.QuantityRemaining
			}
		}
		//设置collection信息
		collection, ok := collcetionInfoMap[strings.ToLower(item.CollectionAddress)]
		if ok {
			resultlisting.CollectionName = collection.Name
			resultlisting.FloorPrice = collection.FloorPrice
			if resultlisting.Name == "" {
				resultlisting.Name = fmt.Sprintf("%s #%s", collection.Name, item.TokenID)
			}
		}
		//设置订单信息
		order, ok := orderIdMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			resultlisting.ListOrderID = order.OrderID
			resultlisting.ListExpireTime = order.ExpireTime
			resultlisting.ListMaker = order.Maker
			resultlisting.ListSalt = order.Salt
		}
		//设置图片信息
		itemExternal, ok := itemExternalMap[strings.ToLower(item.CollectionAddress+item.TokenID)]
		if ok {
			if itemExternal.IsUploadedOss {
				resultlisting.ImageURI = itemExternal.OssUri
			} else {
				resultlisting.ImageURI = itemExternal.ImageUri
			}
		}
		result = append(result, resultlisting)
	}
	return &entity.UserListingsResp{
		Result: result,
		Count:  count,
	}, nil
}

// 获取用户在多条链上的出价信息
func GetMultiChainUserBids(ctx context.Context, serverCtx *svc.ServerCtx, chainNames []string, filter entity.PortfolioMultiChainBidFilterParams) (*entity.UserBidsResp, error) {
	//解析入参
	chainIds := filter.ChainID
	userAddrs := filter.UserAddresses
	collAddrs := filter.CollectionAddresses

	// 1. 遍历每条链,查询用户出价信息
	var totalBids []entity.MultiOrder
	for i, chain := range chainNames {
		userBids, err := serverCtx.Dao.QueryUserBids(ctx, chain, userAddrs, collAddrs)
		if err != nil {
			return nil, errors.Wrap(err, "failed on get user bids info")
		}
		// 将每条链的出价信息添加到总出价列表中
		var tmpBids []entity.MultiOrder
		for _, order := range userBids {
			tmpBids = append(tmpBids, entity.MultiOrder{
				Order:     order,
				ChainID:   chainIds[i],
				ChainName: chain,
			})
		}
		totalBids = append(totalBids, tmpBids...)
	}

	// 2. 构建出价信息映射和Collection地址映射
	bidsMap := make(map[string]entity.UserBid)
	bidCollections := make(map[string][]string)
	for _, bid := range totalBids {
		// 按链名称分组Collection地址
		collections, ok := bidCollections[bid.ChainName]
		if ok {
			bidCollections[bid.ChainName] = append(collections, strings.ToLower(bid.CollectionAddress))
		} else {
			bidCollections[bid.ChainName] = []string{strings.ToLower(bid.CollectionAddress)}
		}

		// 构建唯一key,用于合并相同Collection的出价信息
		key := strings.ToLower(bid.CollectionAddress) + bid.TokenId + bid.Price.String() +
			fmt.Sprintf("%d", bid.MarketplaceId) +
			fmt.Sprintf("%d", bid.ExpireTime) +
			fmt.Sprintf("%d", bid.OrderType)
		userBid, ok := bidsMap[key]
		// 如果key不存在,创建新的出价信息
		if !ok {
			bidsMap[key] = entity.UserBid{
				ChainID:           bid.ChainID,
				CollectionAddress: strings.ToLower(bid.CollectionAddress),
				TokenID:           bid.TokenId,
				BidPrice:          bid.Price,
				MarketplaceID:     bid.MarketplaceId,
				ExpireTime:        bid.ExpireTime,
				BidType:           getBidType(bid.OrderType),
				OrderSize:         bid.QuantityRemaining,
				BidInfos: []entity.BidInfo{
					{
						BidOrderID:    bid.OrderID,
						BidTime:       bid.EventTime,
						BidExpireTime: bid.ExpireTime,
						BidPrice:      bid.Price,
						BidSalt:       bid.Salt,
						BidSize:       bid.Size,
						BidUnfilled:   bid.QuantityRemaining,
					},
				},
			}
		} else {
			// 如果key存在,更新出价信息
			userBid.OrderSize += bid.QuantityRemaining
			userBid.BidInfos = append(userBid.BidInfos, entity.BidInfo{
				BidOrderID:    bid.OrderID,
				BidTime:       bid.EventTime,
				BidExpireTime: bid.ExpireTime,
				BidPrice:      bid.Price,
				BidSalt:       bid.Salt,
				BidSize:       bid.Size,
				BidUnfilled:   bid.QuantityRemaining,
			})
			bidsMap[key] = userBid
		}
	}

	// 3. 查询Collection基本信息
	collectionInfos := make(map[string]multi.Collection)
	for chain, collectionAddrs := range bidCollections {
		//collectionAddrs去重
		collections := utils.RemoveRepeatedElement(collectionAddrs)
		collectionsInfo, err := serverCtx.Dao.QueryCollectionsInfo(ctx, chain, collections)
		if err != nil {
			return nil, errors.Wrap(err, "failed on get collections info")
		}
		for _, c := range collectionsInfo {
			collectionInfos[fmt.Sprintf("%d:%s", c.ChainId, strings.ToLower(c.Address))] = c
		}
	}

	// 4. 组装最终结果
	var result []entity.UserBid
	for _, userBid := range bidsMap {
		// 设置Collection名称和图片信息
		collection, ok := collectionInfos[fmt.Sprintf("%d:%s", userBid.ChainID, strings.ToLower(userBid.CollectionAddress))]
		if ok {
			userBid.CollectionName = collection.Name
			userBid.ImageURI = collection.ImageUri
		}
		result = append(result, userBid)
	}

	// 5. 按过期时间降序排序
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ExpireTime > result[j].ExpireTime
	})

	return &entity.UserBidsResp{
		Result: result,
		Count:  len(bidsMap),
	}, nil
}
