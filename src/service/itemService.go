package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/pkg/errors"
)

// 分页查询指定collection指定item的出价信息
func GetItemBidsInfo(ctx context.Context, serverCtx *svc.ServerCtx, chain, collectionAddr, tokenId string,
	page, pageSize int) (*entity.CollectionBidsRes, error) {

	itemBids, count, err := serverCtx.Dao.QueryItemBids(ctx, chain, collectionAddr, tokenId, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}
	for i := 0; i < len(itemBids); i++ {
		itemBids[i].OrderType = getBidType(itemBids[i].OrderType)
	}
	return &entity.CollectionBidsRes{
		Result: itemBids,
		Count:  count,
	}, nil
}
