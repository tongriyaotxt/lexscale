// Package engine 测试 — 覆盖事件驱动引擎核心逻辑
package engine

import (
	"context"
	"testing"
	"time"

	"lexscale/internal/graph"
	"lexscale/internal/guard"
)

// MockCounsel 测试用 Counsel
type MockCounsel struct {
	intents []*Intent
}

func (m *MockCounsel) Plan(ctx context.Context, g *graph.LegalGraph, history []*Intent) ([]*Intent, error) {
	return m.intents, nil
}

// MockWorker 测试用 Worker
type MockWorker struct {
	id          string
	handleTypes []IntentType
	executeFn   func(ctx context.Context, intent *Intent, g *graph.LegalGraph) (*IntentResult, error)
}

func (m *MockWorker) ID() string { return m.id }

func (m *MockWorker) CanHandle(t IntentType) bool {
	for _, ht := range m.handleTypes {
		if ht == t {
			return true
		}
	}
	return false
}

func (m *MockWorker) Execute(ctx context.Context, intent *Intent, g *graph.LegalGraph) (*IntentResult, error) {
	if m.executeFn != nil {
		return m.executeFn(ctx, intent, g)
	}
	return &IntentResult{Summary: "mock executed"}, nil
}

// ─── 测试 ─────────────────────────────────────────────

func TestEngineCreation(t *testing.T) {
	eng := New(Config{
		MatterID: "test-matter",
		Counsel:  &MockCounsel{},
		Workers:  []Worker{},
		Guard:    guard.NewGate(),
	})
	
	if eng == nil {
		t.Fatal("engine should not be nil")
	}
	if eng.Graph() == nil {
		t.Fatal("graph should not be nil")
	}
	if eng.Graph().MatterID != "test-matter" {
		t.Errorf("expected matter ID 'test-matter', got '%s'", eng.Graph().MatterID)
	}
}

func TestIntentLifecycle(t *testing.T) {
	completed := make(chan *Intent, 1)
	
	eng := New(Config{
		MatterID: "test-lifecycle",
		Counsel: &MockCounsel{
			intents: []*Intent{
				{
					ID:       "intent-001",
					Type:     IntentContractReview,
					Goal:     "审查合同",
					Priority: 10,
				},
			},
		},
		Workers: []Worker{
			&MockWorker{
				id:          "worker-1",
				handleTypes: []IntentType{IntentContractReview},
				executeFn: func(ctx context.Context, intent *Intent, g *graph.LegalGraph) (*IntentResult, error) {
					// 模拟分析结果（不直接写图，由 engine recordIntent 统一写入）
					finding := &graph.RiskFinding{
						Title:      "测试风险",
						Severity:   3,
						Likelihood: 2,
						Tier:       graph.TierInference,
					}
					
					return &IntentResult{
						Summary:  "审查完成",
						Findings: []*graph.RiskFinding{finding},
					}, nil
				},
			},
		},
		Guard:            guard.NewGate(),
		OnIntentComplete: func(intent *Intent) { completed <- intent },
	})
	
	// 启动引擎
	if err := eng.Start(); err != nil {
		t.Fatalf("failed to start engine: %v", err)
	}
	
	// 触发规划
	eng.NotifyGraphChanged()
	
	// 等待完成
	select {
	case intent := <-completed:
		if intent.Status != IntentCompleted {
			t.Errorf("expected status COMPLETED, got %s", intent.Status)
		}
		if intent.Result == nil {
			t.Error("result should not be nil")
		}
		if intent.Result.Summary != "审查完成" {
			t.Errorf("expected summary '审查完成', got '%s'", intent.Result.Summary)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for intent completion")
	}
	
	// 验证图更新
	stats := eng.Graph().Stats()
	if stats.FindingCount != 1 {
		t.Errorf("expected 1 finding, got %d", stats.FindingCount)
	}
	
	// 停止引擎
	if err := eng.Stop(); err != nil {
		t.Errorf("failed to stop engine: %v", err)
	}
}

func TestNoAvailableWorker(t *testing.T) {
	completed := make(chan *Intent, 1)
	
	eng := New(Config{
		MatterID: "test-no-worker",
		Counsel: &MockCounsel{
			intents: []*Intent{
				{
					ID:   "intent-001",
					Type: IntentStatuteResearch, // 无 worker 处理此类型
					Goal: "检索法条",
				},
			},
		},
		Workers:          []Worker{}, // 无 worker
		Guard:            guard.NewGate(),
		OnIntentComplete: func(intent *Intent) { completed <- intent },
	})
	
	eng.Start()
	eng.NotifyGraphChanged()
	
	select {
	case intent := <-completed:
		if intent.Status != IntentFailed {
			t.Errorf("expected status FAILED, got %s", intent.Status)
		}
		if intent.Result == nil || intent.Result.Error == "" {
			t.Error("expected error message")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	
	eng.Stop()
}

func TestWaitIdle(t *testing.T) {
	eng := New(Config{
		MatterID: "test-idle",
		Counsel:  &MockCounsel{},
		Workers:  []Worker{},
		Guard:    guard.NewGate(),
	})
	
	eng.Start()
	
	// 空队列应该立即 idle
	if !eng.WaitIdle(1 * time.Second) {
		t.Error("expected idle on empty queue")
	}
	
	eng.Stop()
}

func TestHistory(t *testing.T) {
	completed := make(chan *Intent, 2)
	
	eng := New(Config{
		MatterID: "test-history",
		Counsel: &MockCounsel{
			intents: []*Intent{
				{ID: "i1", Type: IntentContractReview, Goal: "任务1"},
				{ID: "i2", Type: IntentRiskAssess, Goal: "任务2"},
			},
		},
		Workers: []Worker{
			&MockWorker{id: "w1", handleTypes: []IntentType{IntentContractReview, IntentRiskAssess}},
		},
		Guard:            guard.NewGate(),
		OnIntentComplete: func(intent *Intent) { completed <- intent },
	})
	
	eng.Start()
	eng.NotifyGraphChanged()
	
	// 等待两个完成
	for i := 0; i < 2; i++ {
		select {
		case <-completed:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout")
		}
	}
	
	history := eng.History()
	if len(history) < 2 {
		t.Errorf("expected at least 2 history entries, got %d", len(history))
	}
	
	eng.Stop()
}

func BenchmarkEngineNotify(b *testing.B) {
	eng := New(Config{
		MatterID: "benchmark",
		Counsel:  &MockCounsel{},
		Workers:  []Worker{},
		Guard:    guard.NewGate(),
	})
	defer eng.Stop()
	
	eng.Start()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		eng.NotifyGraphChanged()
	}
}
