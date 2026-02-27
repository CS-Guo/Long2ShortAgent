package handler

import (
	"net/http"

	"aiagent/internal/logic"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetTaskDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// GetTaskDetailHandler 查询任务执行明细：/agent/tasks/:taskId
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TaskDetailRequest
		// 1) 解析 path 参数
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 2) 查询任务详情
		resp, err := logic.NewGetTaskDetailLogic(r.Context(), svcCtx).GetTaskDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 3) 返回结果
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
