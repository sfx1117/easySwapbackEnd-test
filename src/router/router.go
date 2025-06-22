package router

import (
	"EasySwapBackend-test/src/middleware"
	"EasySwapBackend-test/src/service/svc"
	"github.com/gin-gonic/gin"
)

/*
函数负责创建和配置一个 Gin Web 框架的路由器实例，并设置各种中间件和路由规则

	gin.New()和gin.Default()的区别是 ：
	gin.Default()已经内置了两个默认中间件：
		gin.Logger() (日志中间件)
		gin.Recovery() (恢复中间件)
*/
func NewRouter(serverCtx *svc.ServerCtx) *gin.Engine {
	gin.ForceConsoleColor()                    //强制在控制台输出中使用颜色
	gin.SetMode(gin.ReleaseMode)               //设置 Gin 运行模式为发布模式，这会禁用调试信息，提高性能
	router := gin.New()                        //创建一个新的 Gin 引擎实例（不使用默认的 gin.Default()，因为我们需要自定义中间件）
	router.Use(middleware.RecoverMiddleware()) //配置自定义的恢复中间件
	router.Use(middleware.RLog())              //配置自定义的日志中间件
	router.Use(middleware.Cors())              //配置自定义的cors跨域中间件
	initV1Route(router, serverCtx)             //加载业务api路由
	return router
}
