package svc

import (
	"net/http"
	"sync"
	"time"

	"aiagent/internal/config"
	"aiagent/internal/types"
)

// OrchestratedTask 表示一次编排任务的快照信息。
type OrchestratedTask struct {
	TaskID  string
	Status  string
	Intent  string
	Plan    []string
	Calls   []types.ToolCall
	Answer  string
	TraceID string
}

// DialogueSession 表示会话式对话中的记忆槽位。
type DialogueSession struct {
	SessionID     string
	LastShortCode string
	LastShortURL  string
	LastLongURL   string
	LastExpireAt  string
	LastFrom      string
	LastTo        string
	LastTaskID    string
	LastIntent    string
	UpdatedAtUnix int64
}

// ServiceContext 是全局依赖容器。
type ServiceContext struct {
	Config config.Config

	HTTPClient *http.Client

	mu       sync.RWMutex
	tasks    map[string]OrchestratedTask
	sessions map[string]DialogueSession
}

// NewServiceContext 初始化服务上下文。
func NewServiceContext(c config.Config) *ServiceContext {
	timeout := c.InternalServices.TimeoutMs
	if timeout <= 0 {
		timeout = 3000
	}

	return &ServiceContext{
		Config: c,
		HTTPClient: &http.Client{
			Timeout: durationMs(timeout),
		},
		tasks:    make(map[string]OrchestratedTask),
		sessions: make(map[string]DialogueSession),
	}
}

// SaveTask 保存任务快照。
func (s *ServiceContext) SaveTask(task OrchestratedTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.TaskID] = task
}

// GetTask 查询任务快照。
func (s *ServiceContext) GetTask(taskID string) (OrchestratedTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[taskID]
	return t, ok
}

// SaveSession 保存对话会话记忆。
func (s *ServiceContext) SaveSession(session DialogueSession) {
	if session.SessionID == "" {
		return
	}
	if session.UpdatedAtUnix <= 0 {
		session.UpdatedAtUnix = time.Now().Unix()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.SessionID] = session
}

// GetSession 获取对话会话记忆。
func (s *ServiceContext) GetSession(sessionID string) (DialogueSession, bool) {
	if sessionID == "" {
		return DialogueSession{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionID]
	return session, ok
}
