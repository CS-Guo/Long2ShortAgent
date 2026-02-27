package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"aiagent/internal/logic"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ChatStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	// ChatStreamHandler 处理流式对话入口：/agent/chat/stream（SSE）
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		resp, err := logic.NewChatLogic(r.Context(), svcCtx).Chat(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, errors.New("streaming unsupported"))
			return
		}

		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		meta := map[string]any{
			"requestId": resp.RequestId,
			"sessionId": resp.SessionId,
			"model":     resp.Model,
			"mode":      resp.Mode,
			"taskId":    resp.TaskId,
			"intent":    resp.Intent,
			"traceId":   resp.TraceId,
			"memory":    resp.Memory,
		}
		if err := writeSSEJSON(w, flusher, "meta", meta); err != nil {
			return
		}

		for _, delta := range splitByRune(resp.Reply, 20) {
			select {
			case <-r.Context().Done():
				return
			default:
			}

			if err := writeSSEJSON(w, flusher, "delta", map[string]string{"content": delta}); err != nil {
				return
			}
		}

		_ = writeSSEJSON(w, flusher, "done", resp)
	}
}

// writeSSEJSON 输出 JSON 格式的 SSE 事件。
func writeSSEJSON(w http.ResponseWriter, flusher http.Flusher, event string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(w, "event: %s\n", event)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(raw), "\n") {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

// splitByRune 按 rune 数切分字符串，避免中文被截断。
func splitByRune(text string, size int) []string {
	if size <= 0 {
		size = 20
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return []string{""}
	}

	chunks := make([]string, 0, (len(runes)+size-1)/size)
	for i := 0; i < len(runes); i += size {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}
