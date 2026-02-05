// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"goZero/internal/config"
	"goZero/internal/handler"
	"goZero/internal/svc"
	"goZero/pkg/base62"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/shortener-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	//fmt.Printf("load conf::%v\n", c)

	base62.MustInit(c.BaseStr)

	//server := rest.MustNewServer(c.RestConf)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	fmt.Println("register handlers")
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()

}
