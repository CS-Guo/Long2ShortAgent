package handler

import (
	"net/http"

	"aiagent/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	// RegisterHandlers 统一注册 ai-agent 对外路由。
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/agent/execute",
				Handler: ExecuteHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/agent/chat",
				Handler: ChatHandler(serverCtx),
			},
			{
				Method:  http.MethodPost,
				Path:    "/agent/chat/stream",
				Handler: ChatStreamHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/agent/tasks/:taskId",
				Handler: GetTaskDetailHandler(serverCtx),
			},
		},
	)
}
