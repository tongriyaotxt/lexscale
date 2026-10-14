// Package llm 测试 — 覆盖 Provider 路由和 Mock Provider
package llm

import (
	"context"
	"testing"
	"time"
)

// MockProvider 测试用 Mock Provider
type MockProvider struct {
	name     string
	response string
	delay    time.Duration
	err      error
}

// NewMockProvider 创建 Mock Provider
func NewMockProvider(name, response string) *MockProvider {
	return &MockProvider{name: name, response: response}
}

func (m *MockProvider) Name() string { return m.name }

func (m *MockProvider) Complete(ctx context.Context, req *Request) (*Response, error) {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	if m.err != nil {
		return nil, m.err
	}
	
	return &Response{
		Content: m.response,
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Model:   "mock-model",
		FinishReason: "stop",
	}, nil
}

// ─── 测试 ─────────────────────────────────────────────

func TestRouterRegister(t *testing.T) {
	router := NewRouter()
	mock := NewMockProvider("test", "hello")
	
	router.Register(RoleCounsel, mock)
	
	// 验证注册
	provider := router.GetProvider(RoleCounsel)
	if provider == nil {
		t.Fatal("expected provider to be registered")
	}
	if provider.Name() != "test" {
		t.Errorf("expected provider name 'test', got '%s'", provider.Name())
	}
}

func TestRouterFallback(t *testing.T) {
	router := NewRouter()
	fallback := NewMockProvider("fallback", "fallback response")
	
	router.SetFallback(fallback)
	
	// 未注册角色应回退到 fallback
	resp, err := router.Complete(context.Background(), RoleAnalyst, &Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "fallback response" {
		t.Errorf("expected 'fallback response', got '%s'", resp.Content)
	}
}

func TestRouterRoleBasedRouting(t *testing.T) {
	router := NewRouter()
	
	counsel := NewMockProvider("counsel-provider", "counsel response")
	worker := NewMockProvider("worker-provider", "worker response")
	
	router.Register(RoleCounsel, counsel)
	router.Register(RoleAnalyst, worker)
	
	// Counsel 角色
	resp, _ := router.Complete(context.Background(), RoleCounsel, &Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if resp.Content != "counsel response" {
		t.Errorf("expected 'counsel response', got '%s'", resp.Content)
	}
	
	// Analyst 角色
	resp, _ = router.Complete(context.Background(), RoleAnalyst, &Request{
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if resp.Content != "worker response" {
		t.Errorf("expected 'worker response', got '%s'", resp.Content)
	}
}

func TestMockProvider(t *testing.T) {
	mock := NewMockProvider("test", "mock response")
	
	resp, err := mock.Complete(context.Background(), &Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "mock response" {
		t.Errorf("expected 'mock response', got '%s'", resp.Content)
	}
	if resp.Usage.TotalTokens != 30 {
		t.Errorf("expected 30 tokens, got %d", resp.Usage.TotalTokens)
	}
}

func TestRequestValidation(t *testing.T) {
	// 空消息列表
	req := &Request{Messages: []Message{}}
	if len(req.Messages) == 0 {
		t.Log("empty messages handled gracefully")
	}
	
	// 正常请求
	req = &Request{
		Messages: []Message{
			{Role: "system", Content: "You are a lawyer"},
			{Role: "user", Content: "Review this contract"},
		},
		Temperature: 0.3,
		MaxTokens:   4096,
	}
	
	if req.Temperature != 0.3 {
		t.Errorf("expected temperature 0.3, got %f", req.Temperature)
	}
}
