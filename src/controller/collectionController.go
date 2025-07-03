package controller

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service"
	"EasySwapBackend-test/src/svc"
	"EasySwapBackend-test/src/utils"
	"encoding/json"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

// 指定Collection的items信息
func CollectionItemsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//2、获取查询入参filters
		filtersParam := c.Query("filters")
		if filtersParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、将filters转换为查询对象
		var filter entity.CollectionItemFilterParam
		err := json.Unmarshal([]byte(filtersParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、将chainId转换为chain
		chain, ok := utils.ChainIdToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//5、调用service
		res, err := service.GetCollectionItems(c.Request.Context(), serverCtx, chain, collectionAddr, filter)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}

}

// item的详情
func ItemDetailHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenid
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetItemDetail(c.Request.Context(), serverCtx, chain, int(chainId), collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回结果
		xhttp.OkJson(c, res)
	}
}

// 查询item特性信息
func ItemTraitsHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenid
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetItemTraits(c.Request.Context(), serverCtx, chain, collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回结果
		xhttp.OkJson(c, res)
	}
}

// 获取指定 item的Trait的最高价格信息
func ItemTopTraitPriceHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参 集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参 过滤条件
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//3、将过滤条件 解析为对象
		var filter entity.TopTraitFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}
		//4、通过chainId转换chain
		chain, ok := utils.ChainIdToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//5、调用service
		res, err := service.GetItemTopTraitPrice(c.Request.Context(), serverCtx, chain, collectionAddr, filter.TokenIds)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}
}

// 获取NFT Item的图片信息
func ItemImageHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenid
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetItemImage(c.Request.Context(), serverCtx, chain, collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回结果
		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{Result: res})
	}
}

// NFT销售历史价格信息
func HistorySalesHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参 集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取过滤条件 chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、将chainId转换为chain
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、获取过滤条件 duration(时间段、期间、时长)
		duration := c.Query("duration")
		if duration == "" {
			validParamsMap := map[string]bool{
				"24h": true,
				"7d":  true,
				"30d": true,
			}
			if ok := validParamsMap[duration]; !ok {
				xzap.WithContext(c).Error("duration parse error: ", zap.String("duration", duration))
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
		} else {
			duration = "7d"
		}
		//5、调用service
		res, err := service.GetHistorySalesPrice(c.Request.Context(), serverCtx, chain, collectionAddr, duration)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//6、包装返回参数
		xhttp.OkJson(c, res)
	}
}

// 获取NFT的所有者信息
func ItemOwnerHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenid
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		res, err := service.GetItemOwner(c.Request.Context(), serverCtx, chainId, chain, collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回参数
		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{
			Result: res,
		})
	}
}

// 刷新NFT的元数据信息
func RefreshItemMetadataHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取入参集合address
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//2、获取入参tokenid
		tokenId := c.Params.ByName("token_id")
		if tokenId == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//3、获取入参chain_id
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		chain, ok := utils.ChainIdToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		//4、调用service
		err = service.RefreshItemMetadata(c.Request.Context(), serverCtx, chainId, chain, collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		//5、包装返回参数
		successStr := "Success to joined the refresh queue and waiting for refresh."
		xhttp.OkJson(c, entity.CommonResp{Result: successStr})
	}
}
