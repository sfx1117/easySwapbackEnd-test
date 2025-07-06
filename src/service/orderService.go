package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"sort"
)

/*
*
获取订单信息
该函数主要用于获取指定NFT的出价信息,包括单个NFT的最高出价和整个Collection的最高出价
*/
func GetOrderInfos(ctx context.Context, serverCtx *svc.ServerCtx, chain string, filter entity.OrderInfosParam) ([]entity.ItemBid, error) {
	//解析参数
	userAddr := filter.UserAddress
	collectionAddr := filter.CollectionAddress
	tokenIds := filter.TokenIds

	// 1. 构建NFT信息列表
	var items []entity.ItemInfo
	for _, tokenId := range tokenIds {
		items = append(items, entity.ItemInfo{
			CollectionAddress: collectionAddr,
			TokenID:           tokenId,
		})
	}

	// 2. 查询每个NFT的最高出价信息
	bestBids, err := serverCtx.Dao.QueryItemsBestBids(ctx, chain, userAddr, items)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items best bids")
	}

	// 3. 处理每个NFT的最高出价,如果有多个出价选择最高的
	itemBestBids := make(map[string]multi.Order)
	for _, bid := range bestBids {
		order, ok := itemBestBids[bid.TokenId]
		if !ok {
			itemBestBids[bid.TokenId] = bid
			continue
		}
		if bid.Price.GreaterThan(order.Price) {
			itemBestBids[bid.TokenId] = bid
		}
	}

	// 4. 查询整个Collection的最高出价信息
	collectionBids, err := serverCtx.Dao.QueryCollectionTopNBid(ctx, chain, userAddr, collectionAddr, len(tokenIds))
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collection top n bid")
	}
	// 5. 处理并返回最终的出价信息
	return processBids(tokenIds, itemBestBids, collectionBids, collectionAddr), nil
}

// 处理NFT的出价信息,返回每个NFT的最高出价
// 参数说明:
// - tokenIds: NFT的token ID列表
// - itemsBestBids: 每个NFT的最高出价信息,key为tokenId
// - collectionBids: 整个Collection的最高出价列表
// - collectionAddr: Collection地址
//
// 处理逻辑:
// 1. 将itemsBestBids按价格升序排序
// 2. 遍历tokenIds,对每个tokenId:
//   - 如果该tokenId没有单独的出价信息,使用Collection级别的出价(如果有)
//   - 如果该tokenId有单独的出价信息:
//   - 如果Collection级别没有更高的出价,使用该NFT的出价
//   - 如果Collection级别有更高的出价,使用Collection的出价
//
// 3. 返回每个NFT的最终出价信息
func processBids(tokenIds []string, itemBestBids map[string]multi.Order, collectionBid []multi.Order, collectionAddr string) []entity.ItemBid {
	// 1、将itemsBestBids转换为切片并按价格升序排序
	var itemsSortedBids []multi.Order
	for _, bid := range itemBestBids {
		itemsSortedBids = append(itemsSortedBids, bid)
	}
	sort.SliceStable(itemsSortedBids, func(i, j int) bool {
		return itemsSortedBids[i].Price.LessThan(itemsSortedBids[j].Price)
	})

	var resultBids []entity.ItemBid
	var cBidIndex int

	//2、处理没有单独出价的NFT
	for _, tokenId := range tokenIds {
		_, ok := itemBestBids[tokenId]
		if !ok && cBidIndex < len(collectionBid) {
			resultBids = append(resultBids, entity.ItemBid{
				MarketplaceId:     collectionBid[cBidIndex].MarketplaceId,
				CollectionAddress: collectionAddr,
				TokenId:           tokenId,
				OrderId:           collectionBid[cBidIndex].OrderID,
				EventTime:         collectionBid[cBidIndex].EventTime,
				ExpireTime:        collectionBid[cBidIndex].ExpireTime,
				Price:             collectionBid[cBidIndex].Price,
				Salt:              collectionBid[cBidIndex].Salt,
				BidSize:           collectionBid[cBidIndex].Size,
				BidUnfilled:       collectionBid[cBidIndex].QuantityRemaining,
				Bidder:            collectionBid[cBidIndex].Maker,
				OrderType:         getBidType(collectionBid[cBidIndex].OrderType),
			})
			cBidIndex++
		}
	}

	//3、处理没有单独出价的NFT
	for _, itemBid := range itemsSortedBids {
		// 如果没有更多Collection级别的出价,使用NFT自己的出价
		if cBidIndex >= len(collectionBid) {
			resultBids = append(resultBids, entity.ItemBid{
				MarketplaceId:     itemBid.MarketplaceId,
				CollectionAddress: collectionAddr,
				TokenId:           itemBid.TokenId,
				OrderId:           itemBid.OrderID,
				EventTime:         itemBid.EventTime,
				ExpireTime:        itemBid.ExpireTime,
				Price:             itemBid.Price,
				Salt:              itemBid.Salt,
				BidSize:           itemBid.Size,
				BidUnfilled:       itemBid.QuantityRemaining,
				Bidder:            itemBid.Maker,
				OrderType:         getBidType(itemBid.OrderType),
			})
		} else {
			// 比较Collection级别的出价和NFT自己的出价
			cBid := collectionBid[cBidIndex]
			if cBid.Price.GreaterThan(itemBid.Price) {
				// 如果Collection的出价更高,使用Collection的出价
				resultBids = append(resultBids, entity.ItemBid{
					MarketplaceId:     cBid.MarketplaceId,
					CollectionAddress: collectionAddr,
					TokenId:           itemBid.TokenId,
					OrderId:           cBid.OrderID,
					EventTime:         cBid.EventTime,
					ExpireTime:        cBid.ExpireTime,
					Price:             cBid.Price,
					Salt:              cBid.Salt,
					BidSize:           cBid.Size,
					BidUnfilled:       cBid.QuantityRemaining,
					Bidder:            cBid.Maker,
					OrderType:         getBidType(cBid.OrderType),
				})
				cBidIndex++
			} else {
				// 如果NFT自己的出价更高,使用NFT的出价
				resultBids = append(resultBids, entity.ItemBid{
					MarketplaceId:     itemBid.MarketplaceId,
					CollectionAddress: collectionAddr,
					TokenId:           itemBid.TokenId,
					OrderId:           itemBid.OrderID,
					EventTime:         itemBid.EventTime,
					ExpireTime:        itemBid.ExpireTime,
					Price:             itemBid.Price,
					Salt:              itemBid.Salt,
					BidSize:           itemBid.Size,
					BidUnfilled:       itemBid.QuantityRemaining,
					Bidder:            itemBid.Maker,
					OrderType:         getBidType(itemBid.OrderType),
				})
			}
		}
	}
	return resultBids
}
