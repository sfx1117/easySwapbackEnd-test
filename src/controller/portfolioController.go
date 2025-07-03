package controller

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service"
	"EasySwapBackend-test/src/svc"
	"encoding/json"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

// 获取用户拥有的collection信息
func UserMultiChainCollectionsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.UserCollectionsParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、解析封装chain信息
		var chainNames []string
		var chainIds []int
		for _, v := range serverCtx.C.ChainSupported {
			chainNames = append(chainNames, v.Name)
			chainIds = append(chainIds, v.ChainId)
		}
		//4、调用service
		res, err := service.GetMultiChainUserCollection(c.Request.Context(), serverCtx, chainIds, chainNames, filter.UserAddresses)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		xhttp.OkJson(c, res)
	}
}
