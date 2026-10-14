// Package llm LLM 提供商抽象层。
//
// 仿 PentAGI 的 providers/ 设计：每种 msgchain 角色可独立配置模型。
// 法律场景的角色分流建议：
//   - 检索类（searcher/researcher）：小模型，省钱
//   - 起草类（drafter/counsel）：大模型，质量优先
//   - 验证类（verifier）：中等模型 + 强制结构化输出
package llm

import (
	"context"
	"fmt"
)

// Message 对话消息
type Message struct {
	Role    string `json:"role"` // system / user / assistant / tool
	Content string `json:"content"`
}

// ToolDef 工具定义（function calling）
type ToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// Request 请求
type Request struct {
	Model       string      `json:"model"`
	Messages    []Message   `json:"messages"`
	Tools       []ToolDef   `json:"tools,omitempty"`
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

// Response 响应
type Response struct {
	Content      string `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	FinishReason string `json:"finish_reason"`
	Model        string `json:"model,omitempty"`
	Usage        Usage  `json:"usage"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Usage token 用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Provider LLM 提供商接口
type Provider interface {
	Name() string
	Complete(ctx context.Context, req *Request) (*Response, error)
}

// ChainRole 消息链角色（对应 PentAGI 的 msgchain_type）
type ChainRole string

const (
	RoleCounsel    ChainRole = "counsel"     // 主办律师——策略规划
	RoleResearcher ChainRole = "researcher"  // 法律检索员
	RoleAnalyst    ChainRole = "analyst"     // 合同分析师
	RoleDrafter    ChainRole = "drafter"     // 文书起草师
	RoleVerifier   ChainRole = "verifier"    // 论证验证师
	RoleAssessor   ChainRole = "assessor"    // 风险评级师
)

// Router 按角色路由到不同模型
type Router struct {
	providers map[ChainRole]Provider
	fallback  Provider
}

// NewRouter 创建路由器
func NewRouter() *Router {
	return &Router{
		providers: make(map[ChainRole]Provider),
	}
}

// Register 为角色注册提供商
func (r *Router) Register(role ChainRole, p Provider) {
	r.providers[role] = p
}

// SetFallback 设置兜底提供商
func (r *Router) SetFallback(p Provider) {
	r.fallback = p
}

// Complete 按角色路由完成请求
func (r *Router) Complete(ctx context.Context, role ChainRole, req *Request) (*Response, error) {
	p, ok := r.providers[role]
	if !ok {
		p = r.fallback
	}
	if p == nil {
		return nil, fmt.Errorf("no provider for role %s and no fallback", role)
	}
	return p.Complete(ctx, req)
}

// GetProvider 返回指定角色的 Provider（外部访问用）
func (r *Router) GetProvider(role ChainRole) Provider {
	p, ok := r.providers[role]
	if !ok {
		return r.fallback
	}
	return p
}
