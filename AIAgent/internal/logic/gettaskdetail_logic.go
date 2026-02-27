package logic

import (
	"context"
	"errors"
	"strings"

	"aiagent/internal/svc"
	"aiagent/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTaskDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetTaskDetailLogic 创建任务查询逻辑对象。
func NewGetTaskDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTaskDetailLogic {
	return &GetTaskDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetTaskDetail 根据 taskId 返回编排任务的执行细节。
func (l *GetTaskDetailLogic) GetTaskDetail(req *types.TaskDetailRequest) (*types.TaskDetailResponse, error) {
	if strings.TrimSpace(req.TaskId) == "" {
		return nil, errors.New("taskId is required")
	}

	// 从内存任务表读取快照
	task, ok := l.svcCtx.GetTask(req.TaskId)
	if !ok {
		return nil, errors.New("task not found")
	}

	return &types.TaskDetailResponse{
		TaskId:  task.TaskID,
		Status:  task.Status,
		Intent:  task.Intent,
		Plan:    task.Plan,
		Calls:   task.Calls,
		Answer:  task.Answer,
		TraceId: task.TraceID,
	}, nil
}
