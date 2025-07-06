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

// 批量查询出价信息
func OrderInfosHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.OrderInfosParam
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、将chainId转换为chain
		chain, ok := utils.ChainIdToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetOrderInfos(c.Request.Context(), serverCtx, chain, filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{Result: res})
	}
}
