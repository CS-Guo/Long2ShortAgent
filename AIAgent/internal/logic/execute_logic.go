package logic

import (
	"context"

	"aiagent/internal/agent"
	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ExecuteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewExecuteLogic 创建执行逻辑对象。
func NewExecuteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExecuteLogic {
	return &ExecuteLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Execute 负责调用引擎完成任务，并将执行结果持久化到内存任务表。
func (l *ExecuteLogic) Execute(req *types.ExecuteRequest) (*types.ExecuteResponse, error) {
	// 1) 根据配置选择引擎（rule/eino）
	engine := agent.NewEngine(l.svcCtx)
	result, err := engine.Execute(l.ctx, req)
	if err != nil {
		return nil, err
	}

	// 2) 保存任务快照，供 /agent/tasks/:taskId 查询
	task := svc.OrchestratedTask{
		TaskID:  result.TaskID,
		Status:  result.Status,
		Intent:  result.Intent,
		Plan:    result.Plan,
		Calls:   result.Calls,
		Answer:  result.Answer,
		TraceID: result.TraceID,
	}
	l.svcCtx.SaveTask(task)

	// 3) 返回对外响应
	return &types.ExecuteResponse{
		TaskId:  result.TaskID,
		Intent:  result.Intent,
		Answer:  result.Answer,
		Calls:   result.Calls,
		TraceId: result.TraceID,
	}, nil
}
