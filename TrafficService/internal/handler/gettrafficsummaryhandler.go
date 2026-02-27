package handler

import (
	"net/http"

	"trafficservice/internal/logic"
	"trafficservice/internal/svc"
	"trafficservice/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetTrafficSummaryHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.TrafficSummaryRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		resp, err := logic.NewGetTrafficSummaryLogic(r.Context(), svcCtx).GetTrafficSummary(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
