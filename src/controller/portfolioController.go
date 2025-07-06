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

// 查询用户拥有nft的Item基本信息
func UserMultiChainItemsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.PortfolioMultiChainItemFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、解析封装chain信息
		if len(filter.ChainID) == 0 {
			for _, chain := range serverCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainId)
			}
		}
		//4、根据chainId获取chainName
		var chainNames []string
		for _, chainId := range filter.ChainID {
			chainName, ok := utils.ChainIdToChain[chainId]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chainName)
		}
		//5、调用service
		res, err := service.GetMultiChainUserItems(c.Request.Context(), serverCtx, chainNames, filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}
}

// 查询用户挂单的Listing信息
func UserMultiChainListingsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.PortfolioMultiChainListingFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、解析封装chain信息
		if len(filter.ChainID) == 0 {
			for _, chain := range serverCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainId)
			}
		}
		//4、根据chainId获取chainName
		var chainNames []string
		for _, chainId := range filter.ChainID {
			chainName, ok := utils.ChainIdToChain[chainId]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chainName)
		}
		//5、调用service
		res, err := service.GetMultiChainUserListings(c.Request.Context(), serverCtx, chainNames, filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}
}

// 查询用户挂单的Bids信息
func UserMultiChainBidsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、解析过滤参数
		var filter entity.PortfolioMultiChainBidFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、解析封装chain信息
		if len(filter.ChainID) == 0 {
			for _, chain := range serverCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainId)
			}
		}
		//4、根据chainId获取chainName
		var chainNames []string
		for _, chainId := range filter.ChainID {
			chainName, ok := utils.ChainIdToChain[chainId]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chainName)
		}
		//5、调用service
		res, err := service.GetMultiChainUserBids(c.Request.Context(), serverCtx, chainNames, filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}
}
