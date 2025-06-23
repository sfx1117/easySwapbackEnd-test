package router

import (
	"EasySwapBackend-test/src/svc"
	"github.com/gin-gonic/gin"
)

func initV1Route(router *gin.Engine, serverCtx *svc.ServerCtx) {
	apiV1 := router.Group("/api/v1")

	user := apiV1.Group("/user")
	user.GET("")

}
