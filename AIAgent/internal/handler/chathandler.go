package handler

import (
	"net/http"

	"aiagent/internal/logic"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ChatHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// ChatHandler 处理会话对话入口：/agent/chat（可直聊或触发工具）
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatRequest
		// 1) 解析请求
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 2) 调用会话对话逻辑（自动判断是否走工具编排）
		resp, err := logic.NewChatLogic(r.Context(), svcCtx).Chat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 3) 返回对话结果
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
