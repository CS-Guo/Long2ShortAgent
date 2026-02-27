package handler

import (
	"net/http"
	"strings"

	"goZero/internal/logic"
	"goZero/internal/svc"
	"goZero/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func InternalCreateShortLinkHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// InternalCreateShortLinkHandler 提供内部创建短链接口（供 AI 编排服务调用）。
	return func(w http.ResponseWriter, r *http.Request) {
		// 1) 校验内部调用 token
		token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if !svcCtx.IsInternalAuthorized(token) {
			httpx.WriteJsonCtx(r.Context(), w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// 2) 解析请求
		var req types.InternalCreateShortLinkRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 3) 执行业务逻辑
		resp, err := logic.NewInternalCreateShortLinkLogic(r.Context(), svcCtx).Create(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 4) 返回结果
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
