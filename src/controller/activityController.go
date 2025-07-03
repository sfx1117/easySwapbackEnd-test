package controller

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service"
	"EasySwapBackend-test/src/svc"
	"EasySwapBackend-test/src/utils"
	"encoding/json"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

// 批量获取activity信息
// 1. 解析过滤参数
// 2. 根据是否指定链ID执行不同的查询逻辑:
//   - 未指定链ID: 查询所有链上的活动
//   - 指定链ID: 只查询指定链上的活动
func ActivityMultiChainHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.ActivityMultiChainFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//解析chain
		var chains []string
		for _, chainId := range filter.ChainID {
			chains = append(chains, utils.ChainIdToChain[chainId])
		}
		//3、调用service，获取活动信息
		res, err := service.GetMultiChainActivity(c.Request.Context(), serverCtx, chains, filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Get multi-chain activities failed."))
			return
		}
		//4、包装返回参数
		xhttp.OkJson(c, res)
	}
}
