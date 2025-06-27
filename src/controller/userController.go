package controller

import (
	"EasySwapBackend-test/src/entity"
	"EasySwapBackend-test/src/service"
	"EasySwapBackend-test/src/svc"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/kit/validator"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

// 生成login签名信息
func GetLoginMessageHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := c.Params.ByName("address")
		if address == "" {
			xhttp.Error(c, errcode.NewCustomErr("user addr is null"))
			return
		}

		res, err := service.GetLoginMessage(c.Request.Context(), serverCtx, address)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}
		xhttp.OkJson(c, res)
	}
}

// 登录
func UserLoginHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		//请求参数对象
		req := entity.LoginReq{}
		//绑定请求参数
		if err := ctx.BindJSON(req); err != nil {
			xhttp.Error(ctx, err)
			return
		}
		//参数校验
		if err := validator.Verify(&req); err != nil {
			xhttp.Error(ctx, errcode.NewCustomErr(err.Error()))
			return
		}
		//调用service，获取返回结果
		res, err := service.UserLogin(ctx, serverCtx, req)
		if err != nil {
			xhttp.Error(ctx, errcode.NewCustomErr(err.Error()))
			return
		}
		//包装返回结果
		xhttp.OkJson(ctx, entity.UserLoginRes{
			Result: res,
		})
	}
}

// 获取用户签名状态
func GetUserSignStatusHandler(serverCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		address := c.Params.ByName("address")
		if address == "" {
			xhttp.Error(c, errcode.NewCustomErr("user addr is null"))
			return
		}
		service.GetUserSignStatus(c.Request.Context(), serverCtx, address)
	}
}
