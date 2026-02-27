package main

import (
	"context"
	"flag"
	"fmt"

	"trafficservice/internal/config"
	"trafficservice/internal/consumer"
	"trafficservice/internal/handler"
	"trafficservice/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/traffic-service.yaml", "the config file")

// main 是 traffic-service 服务入口：加载配置、启动消费者与 HTTP 服务。
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	// 启动 Redis Stream 消费后台协程
	consumer.Start(context.Background(), ctx)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting traffic-service at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
