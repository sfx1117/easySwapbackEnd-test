package middleware

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"runtime"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

// RecoverMiddleware 恐慌捕获恢复处理
/**
使用 defer 和 recover 捕获 panic
捕获到 panic 后：
	记录错误日志（包括请求信息和调用栈）
	返回统一的错误响应 (errcode.ErrUnexpected)
*/
func RecoverMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if cause := recover(); cause != nil {
				xzap.WithContext(c.Request.Context()).Errorf("[Recovery] panic recovered, request:%s%v [## stack:]:\n%s", dumpRequest(c.Request), cause, dumpStack(3))
				xhttp.Error(c, errcode.ErrUnexpected)
			}
		}()

		c.Next()
	}
}

// dumpRequest 格式化请求样式
func dumpRequest(req *http.Request) string {
	// 复制请求体以便重复读取
	var dup io.ReadCloser
	req.Body, dup = dupReadCloser(req.Body)

	// 构建请求字符串
	var b bytes.Buffer
	var err error

	reqURI := req.RequestURI
	if reqURI == "" {
		reqURI = req.URL.RequestURI()
	}
	// 写入请求行
	_, _ = fmt.Fprintf(&b, "%s %s HTTP/%d.%d\n", req.Method, reqURI, req.ProtoMajor, req.ProtoMinor)
	// 处理请求体
	chunked := len(req.TransferEncoding) > 0 && req.TransferEncoding[0] == "chunked"
	if req.Body != nil {
		var n int64
		var dest io.Writer = &b
		if chunked {
			dest = httputil.NewChunkedWriter(dest)
		}
		n, err = io.Copy(dest, req.Body)
		if chunked {
			dest.(io.Closer).Close()
		}
		if n > 0 {
			_, _ = io.WriteString(&b, "\n")
		}
	}
	// 恢复原始请求体
	req.Body = dup
	if err != nil {
		return err.Error()
	}

	return b.String()
}

/*
*复制可读关闭器
创建请求体的副本，以便多次读取
*/
func dupReadCloser(reader io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	var buf bytes.Buffer
	tee := io.TeeReader(reader, &buf)
	return ioutil.NopCloser(tee), ioutil.NopCloser(&buf)
}

/*
*
获取调用栈信息
*/
func dumpStack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// 打印基础调用信息
		_, _ = fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		// 读取源代码文件
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		// 打印函数名和源代码行
		_, _ = fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

/*
打印函数名和源代码行
*/
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

/*
*
获取函数名
*/
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	if lastslash := bytes.LastIndex(name, slash); lastslash >= 0 {
		name = name[lastslash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}
