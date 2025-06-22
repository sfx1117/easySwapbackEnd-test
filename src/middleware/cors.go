package middleware

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"time"
)

func Cors() gin.HandlerFunc {
	return cors.New(cors.Config{ // 使用cors中间件
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "X-CSRF-Token", "Authorization", "AccessToken", "Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "X-GW-Error-Code", "X-GW-Error-Message"},
		AllowCredentials: true,
		MaxAge:           1 * time.Hour,
	})
}
