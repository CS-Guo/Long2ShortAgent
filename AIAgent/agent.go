package main

import (
	"flag"
	"fmt"
	"os"

	"aiagent/internal/cli"
	"aiagent/internal/config"
	"aiagent/internal/handler"
	"aiagent/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/ai-agent.yaml", "the config file")
var runCLI = flag.Bool("cli", false, "run in interactive command line chat mode")
var cliEnableTools = flag.Bool("cli-tools", false, "enable tool orchestration in cli mode")
var cliSessionID = flag.String("cli-session", "", "session id used in cli mode")
var cliOperatorID = flag.String("cli-operator", "cli-user", "operator id used in cli mode")
var cliTenantID = flag.String("cli-tenant", "cli-tenant", "tenant id used in cli mode")
var cliSystemPrompt = flag.String("cli-system", "", "system prompt used in cli mode")
var cliMaxHistory = flag.Int("cli-history", 12, "max rounds of chat history to keep in cli mode")

// main 是 ai-agent 服务入口：加载配置、初始化上下文、注册路由并启动 HTTP 服务。
func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	if *runCLI {
		// CLI 对话模式下关闭 go-zero 每分钟资源统计日志，避免打断输入体验。
		stat.DisableLog()

		err := cli.RunChatREPL(c, cli.Options{
			EnableTools:  *cliEnableTools,
			SessionID:    *cliSessionID,
			OperatorID:   *cliOperatorID,
			TenantID:     *cliTenantID,
			SystemPrompt: *cliSystemPrompt,
			MaxHistory:   *cliMaxHistory,
		})
		if err != nil {
			fmt.Printf("cli chat failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting ai-agent at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
