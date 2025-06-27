package router

import (
	"EasySwapBackend-test/src/controller"
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
}
