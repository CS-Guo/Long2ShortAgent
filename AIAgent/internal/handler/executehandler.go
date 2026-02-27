package handler

import (
	"net/http"

	"aiagent/internal/logic"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ExecuteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// ExecuteHandler 处理统一执行入口：/agent/execute
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ExecuteRequest
		// 1) 解析请求
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 2) 进入编排逻辑（rule/eino）
		resp, err := logic.NewExecuteLogic(r.Context(), svcCtx).Execute(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 3) 返回结果
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
