package controller

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service"
	"EasySwapBackend-test/src/svc"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"sort"
	"strconv"
	"sync"
)

// 获取NFT集合排名信息
func TopRankingHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤条件limit，需要返回的数量
		limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取过滤条件range，指定时间段
		period := c.Query("range")
		if period != "" {
			validParamMap := map[string]bool{
				"15m": true, // 15分钟
				"1h":  true, // 1小时
				"6h":  true, // 6小时
				"1d":  true, // 1天
				"7d":  true, // 7天
				"30d": true, // 30天
			}
			if ok := validParamMap[period]; !ok {
				xzap.WithContext(c).Error("range parse error:", zap.String("range", period))
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
		} else {
			period = "1d"
		}
		//3、使用WaitGroup和Mutex来保证并发安全
		var allResult []*entity.CollectionRankingInfo
		var wg sync.WaitGroup
		var mu sync.Mutex
		// 并发获取每条链的排名数据
		for _, chain := range serverCtx.C.ChainSupported {
			wg.Add(1)
			go func(chain string) {
				defer wg.Done()
				//调用service,获取指定链上的排名信息
				result, err := service.GetTopRanking(c.Request.Context(), serverCtx, chain, period, limit)
				if err != nil {
					xhttp.Error(c, err)
					return
				}
				//将结果追加到总结果中
				mu.Lock()
				allResult = append(allResult, result...)
				mu.Unlock()
			}(chain.Name)
		}
		//等待所有查询完成
		wg.Wait()
		//根据交易量对集合进行降序排序
		sort.SliceStable(allResult, func(i, j int) bool {
			return allResult[i].Volume.GreaterThan(allResult[j].Volume)
		})
		//包装返回参数
		xhttp.OkJson(c, entity.CommonResp{
			Result: allResult,
		})
	}
}
