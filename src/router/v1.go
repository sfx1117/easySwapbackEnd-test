package router

import (
	"EasySwapBackend-test/src/controller"
	"EasySwapBackend-test/src/middleware"
	"EasySwapBackend-test/src/svc"
	"github.com/gin-gonic/gin"
)

func initV1Route(router *gin.Engine, serverCtx *svc.ServerCtx) {
	apiV1 := router.Group("/api/v1")

	user := apiV1.Group("/user")
	user.GET("/:address/login-message", controller.GetLoginMessageHandler(serverCtx)) // 生成login签名信息
	user.GET("/login", controller.UserLoginHandler(serverCtx))                        // 登录
	user.GET("/:address/sig-status", controller.GetUserSignStatusHandler(serverCtx))  // 获取用户签名状态

	collections := apiV1.Group("/collections")
	collections.GET("/:address", controller.CollectionDetailHandler(serverCtx))                 //指定Collection详情
	collections.GET("/:address/bids", controller.CollectionBidsHandler(serverCtx))              //指定Collection的bids信息
	collections.GET("/:address/:token_id/bids", controller.CollectionItemBidHandler(serverCtx)) //指定collection指定item的出价信息
	collections.GET("/:address/items", controller.CollectionItemsHandler(serverCtx))            //指定Collection的items信息
	collections.GET("/:address/:token_id", controller.ItemDetailHandler(serverCtx))             //item详情
	collections.GET("/:address/:token_id/trait", controller.ItemTraitsHandler(serverCtx))       //查询item特性信息
	collections.GET("/:address/top-trait", controller.ItemTopTraitPriceHandler(serverCtx))      //获取指定 item的Trait的最高价格信息
	collections.GET("/:address/:token_id/image", middleware.CacheApi(serverCtx.KvStore, 60),
		controller.ItemImageHandler(serverCtx)) // 获取NFT Item的图片信息
	collections.GET("/:address/history-sales", controller.HistorySalesHandler(serverCtx))             //查询指定时间段 NFT的历史销售价格
	collections.GET("/:address/:token_id/owner", controller.ItemOwnerHandler(serverCtx))              //获取NFT所有者信息
	collections.GET("/:address/:token_id/metadata", controller.RefreshItemMetadataHandler(serverCtx)) //刷新NFT的元数据信息
	collections.GET("/ranking", controller.TopRankingHandler(serverCtx))                              // 获取NFT集合排名信息

	activities := apiV1.Group("/activities")
	activities.GET("", controller.ActivityMultiChainHandler(serverCtx)) //批量获取activity信息

	portfolio := apiV1.Group("/portfolio")
	portfolio.GET("/collections", controller.UserMultiChainCollectionsHandler(serverCtx)) //获取用户拥有Collection信息
}
