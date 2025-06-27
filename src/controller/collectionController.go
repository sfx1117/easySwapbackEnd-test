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
	"strconv"
)

/*
*
指定Collection详情
*/
func CollectionDetailHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 32)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、根据chianId转换为chain
		chain := utils.ChainIdToChain[int(chainId)]
		if chain == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取参数address
		address := c.Params.ByName("address")
		if address == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetCollectionDetail(c.Request.Context(), serverCtx, chain, address)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回参数
		xhttp.OkJson(c, res)
	}
}

/*
*
指定Collection的bids信息
*/
func CollectionBidsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参filters
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、将filterParam转换为对象
		var filter entity.CollectionBidFilterParam
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//4、根据chainId转换chain
		chain, ok := utils.ChainIdToChain[filter.ChainId]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//5、调用service
		bidsRes, err := service.GetBids(c.Request.Context(), serverCtx, chain, collectionAddr, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回结果
		xhttp.OkJson(c, bidsRes)
	}
}

// 指定collection指定item的出价信息
func CollectionItemBidHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenId
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参查询条件filters
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//4、将filters转换为对象
		var filter entity.CollectionBidFilterParam
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//5、将chainId转换为chain
		chain, ok := utils.ChainIdToChain[filter.ChainId]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//6、调用service
		itemBidsInfo, err := service.GetItemBidsInfo(c.Request.Context(), serverCtx, chain, collectionAddr, tokenId, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//7、保证返回结果
		xhttp.OkJson(c, itemBidsInfo)
	}
}
