package main

import (
	"EasySwapBackend-test/src/app"
	"EasySwapBackend-test/src/config"
	"EasySwapBackend-test/src/router"
	"EasySwapBackend-test/src/service/svc"
	"flag"
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

	serverCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}

	r := router.NewRouter(serverCtx)
	app := app.NewPlatform(c, r, serverCtx)
	app.Start()
}
