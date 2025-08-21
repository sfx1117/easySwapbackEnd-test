package main

import (
	"EasySwapBackend-test/src/app"
	"EasySwapBackend-test/src/config"
	"EasySwapBackend-test/src/router"
	"EasySwapBackend-test/src/svc"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
)

const (
	defaultConfigPath = "./config/config.toml"
)

func main() {
	conf := flag.String("conf", defaultConfigPath, "conf file path")
	flag.Parse()
	c, err := config.UnmarshalConfig(*conf)
	if err != nil {
		panic(err)
	}

	for _, chain := range c.ChainSupported {
		if chain.ChainId == 0 || chain.Name == "" {
			panic("invalid chain_suffix config")
		}
	}
	// 启动 pprof 服务器
	go func() {
		log.Println(http.ListenAndServe(":6060", nil)) // 默认pprof地址
	}()

	serverCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}

	r := router.NewRouter(serverCtx)
	app := app.NewPlatform(c, r, serverCtx)
	app.Start()
}
