package middleware

import (
	"bytes"
	"crypto/sha512"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
)

const CacheApiPrefix = "apicache:"

type responseCache struct {
	Status int
	Header http.Header
	Data   []byte
}

// 一个缓存中间件函数,用于缓存API响应数据
// 1. 接收一个 xkv.Store 存储实例和过期时间作为参数
// 2. 检查请求是否有缓存,如果有且状态码为200则直接返回缓存数据
// 3. 如果没有缓存,则继续处理请求
// 4. 请求处理完成后,如果响应状态码为200,则将响应数据缓存起来
func CacheApi(store *xkv.Store, expireSeconds int) gin.HandlerFunc {
	return func(c *gin.Context) {
		var data xhttp.Response
		//创建响应体写入器，用于获取相应内容
		bodyLogWrite := &BodyLogWrite{ResponseWriter: c.Writer, body: bytes.NewBufferString("")}
		c.Writer = bodyLogWrite

		//生成缓存key
		cacheKey := CreateKey(c)
		if cacheKey == "" {
			xhttp.Error(c, errcode.NewCustomErr("cache error:no cache"))
			c.Abort() //中断请求
		}
		//尝试获取缓存数据
		cacheData, err := (*store).Get(cacheKey)
		if err == nil && cacheData != "" {
			//将缓存数据解析成对象
			cache := unserialize(cacheData)
			//如果有缓存，则直接返回缓存的响应
			if cache != nil {
				//设置响应状态码
				bodyLogWrite.ResponseWriter.WriteHeader(cache.Status)
				//设置响应头
				for k, vals := range cache.Header {
					for _, v := range vals {
						bodyLogWrite.ResponseWriter.Header().Set(k, v)
					}
				}
				//解析并返回响应体
				//将缓存数据解析成json格式
				err := json.Unmarshal(cache.Data, &data)
				if err == nil {
					//检查状态码是否为200 OK
					if data.Code == http.StatusOK {
						//写入响应体
						bodyLogWrite.ResponseWriter.Write(cache.Data)
						//终止后续处理
						c.Abort()
					}
				}
			}
		}
		//若缓存中没有，则继续处理请求
		c.Next()
		//获取响应数据
		responseBody := bodyLogWrite.body.Bytes()
		// 如果响应状态码为200,则缓存响应数据
		err = json.Unmarshal(responseBody, &data)
		if err == nil {
			if data.Code == http.StatusOK {
				storeCache := responseCache{
					Header: bodyLogWrite.Header().Clone(),
					Status: bodyLogWrite.ResponseWriter.Status(),
					Data:   responseBody,
				}
				store.SetnxEx(cacheKey, serialize(storeCache), expireSeconds)
			}
		}
	}
}

// 生成缓存的key
// 1. 将路径、查询参数和请求体组合成缓存key
// 2. 如果key长度超过128,使用SHA512进行哈希
// 3. 添加缓存前缀并返回最终的key
func CreateKey(c *gin.Context) string {
	var buf bytes.Buffer
	reader := io.TeeReader(c.Request.Body, &buf)
	reqBody, _ := ioutil.ReadAll(reader)
	c.Request.Body = ioutil.NopCloser(&buf)

	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery
	// 组合缓存key
	cacheKey := path + "," + query + string(reqBody)
	//如果key太长则进行哈希
	if len(cacheKey) > 128 {
		hash := sha512.New()                    // 创建SHA-512哈希对象
		hash.Write([]byte(cacheKey))            //将cacheKey写入hash计算器
		cacheKey = string(hash.Sum([]byte(""))) //计算hash值（空参数表示不追加额外数据）
		cacheKey = fmt.Sprintf("%x", cacheKey)  //将哈希结果转换为十六进制字符串
	}
	// 添加缓存前缀
	cacheKey = CacheApiPrefix + cacheKey
	return cacheKey
}

// 将结构体数据序列化
func serialize(cache responseCache) string {
	//创建缓冲区
	buf := new(bytes.Buffer)
	//创建新的gob编码器
	enc := gob.NewEncoder(buf)
	//将cache结构体数据编码到缓冲区中
	if err := enc.Encode(cache); err != nil {
		return "" // 编码失败返回空字符串
	} else {
		return buf.String() // 成功返回序列化后的字符串
	}
}

// 反序列化成结构体数据
func unserialize(data string) *responseCache {
	// 1. 创建一个空的responseCache对象
	var g1 = responseCache{}
	// 2. 从data创建字节缓冲区，并创建gob解码器
	dec := gob.NewDecoder(bytes.NewBuffer([]byte(data)))
	// 3. 解码数据到g1对象
	if err := dec.Decode(&g1); err != nil {
		return nil // 解码失败返回nil
	} else {
		return &g1 // 成功返回反序列化后的对象指针
	}
}
