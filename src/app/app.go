package app

import (
	"EasySwapBackend-test/src/config"
	"EasySwapBackend-test/src/svc"
	"context"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Platform struct {
	config    *config.Config
	router    *gin.Engine
	serverCtx *svc.ServerCtx
}

func NewPlatform(config *config.Config, router *gin.Engine, serverCtx *svc.ServerCtx) *Platform {
	return &Platform{
		config:    config,
		router:    router,
		serverCtx: serverCtx,
	}
}

func (p *Platform) Start() {
	xzap.WithContext(context.Background()).Info("EasySwap-End run", zap.String("port", p.config.Api.Port))
	err := p.router.Run(p.config.Api.Port)
	if err != nil {
		panic(err)
	}
}
