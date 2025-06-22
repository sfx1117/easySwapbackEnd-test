package middleware

import (
	"bytes"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"io/ioutil"
	"time"
)

type BodyLogWrite struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *BodyLogWrite) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *BodyLogWrite) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// RLog 请求响应日志打印处理
// RLog() 是一个中间件函数,用于记录HTTP请求和响应的详细日志
// 主要功能包括:
// 1. 记录请求的URL路径、查询参数
// 2. 记录请求体内容
// 3. 记录响应体内容
// 4. 记录请求处理时间
// 5. 记录请求/响应的各种元数据(状态码、方法、IP等)
// 6. 使用zap日志库将信息写入日志
func RLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		//1、获取原始请求路径和参数
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		//2、获取并保存请求体
		var buf bytes.Buffer
		reader := io.TeeReader(c.Request.Body, &buf)
		requestBody, _ := ioutil.ReadAll(reader)
		c.Request.Body = ioutil.NopCloser(&buf)
		bodyLogWriter := &BodyLogWrite{ResponseWriter: c.Writer, body: bytes.NewBufferString("")}
		c.Writer = bodyLogWriter

		//3、开始时间
		start := time.Now()

		//4、调用下一个处理器
		c.Next()

		//5、获取响应体
		responseBody := bodyLogWriter.body.Bytes()
		logger := xzap.WithContext(c.Request.Context())

		if len(c.Errors) > 0 {
			// 如果有错误,记录错误信息
			for _, e := range c.Errors.Errors() {
				logger.Error(e)
			}
		} else {
			// 计算处理时间
			latency := float64(time.Now().Sub(start).Nanoseconds() / 1000000.0)
			// 记录请求和响应的详细信息
			fields := []zapcore.Field{
				zap.Int("status", c.Writer.Status()),
				zap.String("method", c.Request.Method),
				zap.String("function", c.HandlerName()),
				zap.String("path", path),
				zap.String("query", query),
				zap.String("ip", c.ClientIP()),
				zap.String("user-agent", c.Request.UserAgent()),
				zap.String("token", c.Request.Header.Get("session_id")),
				zap.String("content-type", c.Request.Header.Get("Content-Type")),
				zap.Float64("latency", latency),
				zap.String("request", string(requestBody)),
				zap.String("response", string(responseBody)),
			}
			logger.Info("Go-End", fields...)
		}
	}
}
