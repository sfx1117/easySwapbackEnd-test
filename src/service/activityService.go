package service

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/pkg/errors"
)

// 获取链上活动信息
func GetMultiChainActivity(ctx context.Context, serverCtx *svc.ServerCtx, chains []string, filter entity.ActivityMultiChainFilterParams) (*entity.ActivityResp, error) {
	// 查询多链上的活动信息
	activities, total, err := serverCtx.Dao.QueryMultiChainActivities(ctx, chains, filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query multi-chain activity")
	}
	if len(activities) == 0 || total == 0 {
		return &entity.ActivityResp{
			Result: nil,
			Count:  0,
		}, nil
	}
	//查询多链活动的外部信息
	results, err := serverCtx.Dao.QueryMultiChainActivityExternalInfo(ctx, filter.ChainID, chains, activities)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query activity external info")
	}
	return &entity.ActivityResp{
		Result: results,
		Count:  total,
	}, nil
}
