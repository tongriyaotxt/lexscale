// Package guard 测试 — 覆盖审批门核心逻辑
package guard

import (
	"fmt"
	"testing"
	"time"
)

// mockIntent 模拟意图（避免循环依赖）
type mockIntent struct {
	ID   string
	Type string
	Goal string
}

func TestNewGate(t *testing.T) {
	gate := NewGate()
	
	if gate == nil {
		t.Fatal("gate should not be nil")
	}
	
	// 添加默认规则
	rules := DefaultLegalRules()
	for _, r := range rules {
		gate.AddRule(r)
	}
	
	if len(gate.rules) < 4 {
		t.Errorf("expected at least 4 rules, got %d", len(gate.rules))
	}
}

func TestCheckIntentAllow(t *testing.T) {
	gate := NewGate()
	
	// 添加一条 allow 规则
	gate.AddRule(Rule{
		ID:       "allow-all",
		Name:     "允许所有",
		Decision: DecisionAllow,
		Match:    func(intent interface{}) bool { return true },
		Reason:   "测试",
	})
	
	intent := &mockIntent{ID: "test-001", Type: "CONTRACT_REVIEW"}
	decision := gate.CheckIntent(intent)
	
	if decision != DecisionAllow {
		t.Errorf("expected ALLOW, got %s", decision)
	}
}

func TestCheckIntentDeny(t *testing.T) {
	gate := NewGate()
	
	// 添加 deny 规则
	gate.AddRule(Rule{
		ID:       "deny-all",
		Name:     "拒绝所有",
		Decision: DecisionDeny,
		Match:    func(intent interface{}) bool { return true },
		Reason:   "测试 deny",
	})
	
	intent := &mockIntent{ID: "test-deny", Type: "DRAFT"}
	decision := gate.CheckIntent(intent)
	
	if decision != DecisionDeny {
		t.Errorf("expected DENY, got %s", decision)
	}
}

func TestCheckIntentAsk(t *testing.T) {
	gate := NewGate()
	
	// 添加 ask 规则
	gate.AddRule(Rule{
		ID:       "ask-review",
		Name:     "需要审批",
		Decision: DecisionAsk,
		Match: func(intent interface{}) bool {
			if mi, ok := intent.(*mockIntent); ok {
				return mi.Type == "DRAFT_DOCUMENT"
			}
			return false
		},
		Reason: "起草需要审批",
	})
	
	// ASK 类型
	draftIntent := &mockIntent{ID: "test-ask", Type: "DRAFT_DOCUMENT"}
	decision := gate.CheckIntent(draftIntent)
	if decision != DecisionAsk {
		t.Errorf("expected ASK for draft, got %s", decision)
	}
	
	// 其他类型应 ALLOW
	reviewIntent := &mockIntent{ID: "test-allow", Type: "CONTRACT_REVIEW"}
	decision = gate.CheckIntent(reviewIntent)
	if decision != DecisionAllow {
		t.Errorf("expected ALLOW for review, got %s", decision)
	}
}

func TestDenyPrecedence(t *testing.T) {
	gate := NewGate()
	
	// 先加 allow，再加 deny — deny 应该优先生效
	gate.AddRule(Rule{
		ID:       "allow-all",
		Name:     "允许所有",
		Decision: DecisionAllow,
		Match:    func(intent interface{}) bool { return true },
	})
	
	gate.AddRule(Rule{
		ID:       "deny-specific",
		Name:     "拒绝特定类型",
		Decision: DecisionDeny,
		Match: func(intent interface{}) bool {
			if mi, ok := intent.(*mockIntent); ok {
				return mi.Type == "DRAFT_DOCUMENT"
			}
			return false
		},
	})
	
	draftIntent := &mockIntent{Type: "DRAFT_DOCUMENT"}
	decision := gate.CheckIntent(draftIntent)
	if decision != DecisionDeny {
		t.Errorf("deny should take precedence, got %s", decision)
	}
	
	allowIntent := &mockIntent{Type: "CONTRACT_REVIEW"}
	decision = gate.CheckIntent(allowIntent)
	if decision != DecisionAllow {
		t.Errorf("non-matching intent should be allowed, got %s", decision)
	}
}

func TestAutoAllow(t *testing.T) {
	gate := NewGate()
	
	ruleID := "ask-review"
	gate.AddRule(Rule{
		ID:       ruleID,
		Name:     "需要审批",
		Decision: DecisionAsk,
		Match:    func(intent interface{}) bool { return true },
	})
	
	intent := &mockIntent{Type: "ANY"}
	
	// 无 auto-allow 时应 ASK
	decision := gate.CheckIntent(intent)
	if decision != DecisionAsk {
		t.Errorf("expected ASK before auto-allow, got %s", decision)
	}
	
	// 授予 auto-allow
	gate.GrantAutoAllow(ruleID, "test-user", 1*time.Hour)
	
	// 现在有 auto-allow 应 ALLOW
	decision = gate.CheckIntent(intent)
	if decision != DecisionAllow {
		t.Errorf("expected ALLOW after auto-allow granted, got %s", decision)
	}
	
	// 撤销 auto-allow
	gate.RevokeAutoAllow(ruleID)
	decision = gate.CheckIntent(intent)
	if decision != DecisionAsk {
		t.Errorf("expected ASK after auto-allow revoked, got %s", decision)
	}
}

func TestAutoAllowExpiry(t *testing.T) {
	gate := NewGate()
	
	ruleID := "expiring-rule"
	gate.AddRule(Rule{
		ID:       ruleID,
		Name:     "过期规则",
		Decision: DecisionAsk,
		Match:    func(intent interface{}) bool { return true },
	})
	
	// 授予极短的 auto-allow
	gate.GrantAutoAllow(ruleID, "test-user", 1*time.Nanosecond)
	
	// 等待过期
	time.Sleep(10 * time.Millisecond)
	
	intent := &mockIntent{Type: "ANY"}
	decision := gate.CheckIntent(intent)
	if decision != DecisionAsk {
		t.Errorf("expected ASK after auto-allow expired, got %s", decision)
	}
}

func TestDenyNotOverriddenByAutoAllow(t *testing.T) {
	gate := NewGate()
	
	ruleID := "deny-rule"
	gate.AddRule(Rule{
		ID:       ruleID,
		Name:     "不可豁免",
		Decision: DecisionDeny,
		Match:    func(intent interface{}) bool { return true },
	})
	
	// 即使授予 auto-allow，deny 仍然生效
	gate.GrantAutoAllow(ruleID, "admin", 24*time.Hour)
	
	intent := &mockIntent{Type: "ANY"}
	decision := gate.CheckIntent(intent)
	if decision != DecisionDeny {
		t.Errorf("DENY should not be overridable by auto-allow, got %s", decision)
	}
}

func TestDefaultLegalRules(t *testing.T) {
	rules := DefaultLegalRules()
	
	if len(rules) < 4 {
		t.Errorf("expected at least 4 default rules, got %d", len(rules))
	}
	
	// 检查关键规则存在
	ruleIDs := make(map[string]bool)
	for _, r := range rules {
		ruleIDs[r.ID] = true
	}
	
	expected := []string{
		"no-client-contact-without-review",
		"no-fact-without-citation",
		"red-risk-mandatory-review",
		"conflict-check-required",
	}
	
	for _, id := range expected {
		if !ruleIDs[id] {
			t.Errorf("expected default rule %s not found", id)
		}
	}
}

func TestStats(t *testing.T) {
	gate := NewGate()
	
	gate.AddRule(Rule{
		ID:       "test-allow",
		Decision: DecisionAllow,
		Match:    func(intent interface{}) bool { return true },
	})
	
	intent := &mockIntent{Type: "TEST"}
	
	// 执行多次
	for i := 0; i < 5; i++ {
		gate.CheckIntent(intent)
	}
	
	gate.statsMu.Lock()
	allowCount := gate.AllowCount
	gate.statsMu.Unlock()
	
	if allowCount != 5 {
		t.Errorf("expected 5 allows, got %d", allowCount)
	}
}

func TestPendingQueue(t *testing.T) {
	gate := NewGate()
	
	if gate.PendingCount() != 0 {
		t.Errorf("expected 0 pending initially, got %d", gate.PendingCount())
	}
}

// BenchmarkCheckIntent 性能测试
func BenchmarkCheckIntent(b *testing.B) {
	gate := NewGate()
	
	// 添加 10 条规则
	for i := 0; i < 10; i++ {
		gate.AddRule(Rule{
			ID:       fmt.Sprintf("rule-%d", i),
			Decision: DecisionAllow,
			Match:    func(intent interface{}) bool { return true },
		})
	}
	
	intent := &mockIntent{Type: "BENCHMARK"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gate.CheckIntent(intent)
	}
}
