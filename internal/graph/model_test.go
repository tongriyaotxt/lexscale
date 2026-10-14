// Package graph 测试 — 覆盖双图模型和三层证据
package graph

import (
	"testing"
)

func TestNewLegalGraph(t *testing.T) {
	g := NewLegalGraph("matter-test-001")
	
	if g.MatterID != "matter-test-001" {
		t.Errorf("expected matter ID 'matter-test-001', got '%s'", g.MatterID)
	}
	if g.Entities == nil {
		t.Error("entities map not initialized")
	}
	if g.Claims == nil {
		t.Error("claims map not initialized")
	}
	if g.Findings == nil {
		t.Error("findings map not initialized")
	}
}

func TestAddEntity(t *testing.T) {
	g := NewLegalGraph("matter-test")
	
	entity := g.AddEntity(EntityParty, "甲方公司", map[string]string{
		"统一社会信用代码": "91110000MA01XXXX",
		"法定代表人":     "张三",
	})
	
	if entity.ID == "" {
		t.Error("entity ID should not be empty")
	}
	if entity.Type != EntityParty {
		t.Errorf("expected type PARTY, got %s", entity.Type)
	}
	if entity.Name != "甲方公司" {
		t.Errorf("expected name '甲方公司', got '%s'", entity.Name)
	}
	if entity.Attributes["法定代表人"] != "张三" {
		t.Error("attribute not set correctly")
	}
	
	// 验证实体已存入图
	if len(g.Entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(g.Entities))
	}
}

func TestAddFinding(t *testing.T) {
	g := NewLegalGraph("matter-test")
	
	finding := &RiskFinding{
		Title:       "合同违约风险",
		Description: "付款条款存在重大法律瑕疵",
		Severity:    4,
		Likelihood:  3,
		Tier:        TierInference,
	}
	
	g.AddFinding(finding)
	
	if finding.ID == "" {
		t.Error("finding ID should not be empty")
	}
	if finding.Score != 12 {
		t.Errorf("expected score 12 (4×3), got %d", finding.Score)
	}
	if finding.Level != RiskOrange {
		t.Errorf("expected level ORANGE, got %s", finding.Level)
	}
	
	// 验证已存入图
	if len(g.Findings) != 1 {
		t.Errorf("expected 1 finding, got %d", len(g.Findings))
	}
}

func TestRiskLevelCalculation(t *testing.T) {
	tests := []struct {
		name       string
		severity   int
		likelihood int
		wantScore  int
		wantLevel  RiskLevel
	}{
		{"GREEN low", 1, 2, 2, RiskGreen},
		{"GREEN high", 2, 2, 4, RiskGreen},
		{"YELLOW low", 1, 5, 5, RiskYellow},
		{"YELLOW mid", 2, 3, 6, RiskYellow},
		{"YELLOW high", 3, 3, 9, RiskYellow},
		{"ORANGE low", 2, 5, 10, RiskOrange},
		{"ORANGE mid", 3, 4, 12, RiskOrange},
		{"ORANGE high", 5, 3, 15, RiskOrange},
		{"RED low", 4, 4, 16, RiskRed},
		{"RED high", 5, 5, 25, RiskRed},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &RiskFinding{
				Severity:   tt.severity,
				Likelihood: tt.likelihood,
			}
			f.CalculateScore()
			
			if f.Score != tt.wantScore {
				t.Errorf("score: got %d, want %d", f.Score, tt.wantScore)
			}
			if f.Level != tt.wantLevel {
				t.Errorf("level: got %s, want %s", f.Level, tt.wantLevel)
			}
		})
	}
}

func TestEvidenceTier(t *testing.T) {
	// FACT 级主张必须有引用
	factClaim := ClaimNode{
		ID:        "claim-001",
		Statement: "根据民法典第577条，当事人一方不履行合同义务...",
		Status:    ClaimSupported,
		Support: []SupportEdge{
			{
				ID:     "support-001",
				Type:   SupportStatute,
				Target: "statute-001",
				Tier:   TierFact,
				Citations: []Citation{
					{
						SourceType: "statute",
						SourceRef:  "民法典第577条",
						Locator:    "第577条",
						Excerpt:    "当事人一方不履行合同义务或者履行合同义务不符合约定的...",
					},
				},
			},
		},
	}
	
	// 验证 FACT 级有引用
	for _, s := range factClaim.Support {
		if s.Tier == TierFact && len(s.Citations) == 0 {
			t.Error("FACT tier claim must have at least one citation")
		}
	}
	
	// INFERENCE 级可以无引用但必须有推理
	inferenceClaim := ClaimNode{
		ID:        "claim-002",
		Statement: "根据合同条款推断，甲方意图保留知识产权",
		Status:    ClaimOpen,
		Support: []SupportEdge{
			{
				ID:        "support-002",
				Type:      SupportLogic,
				Target:    "clause-8",
				Tier:      TierInference,
				Reasoning: "合同第8.2条明确约定'乙方保留全部知识产权'，结合上下文可推断...",
			},
		},
	}
	
	for _, s := range inferenceClaim.Support {
		if s.Tier == TierInference && s.Reasoning == "" {
			t.Error("INFERENCE tier claim must have reasoning")
		}
	}
}

func TestStats(t *testing.T) {
	g := NewLegalGraph("matter-stats-test")
	
	// 添加实体
	g.AddEntity(EntityParty, "甲方", nil)
	g.AddEntity(EntityParty, "乙方", nil)
	g.AddEntity(EntityContract, "服务协议", nil)
	
	// 添加风险发现
	g.AddFinding(&RiskFinding{Severity: 5, Likelihood: 4})  // RED (20)
	g.AddFinding(&RiskFinding{Severity: 4, Likelihood: 3})  // ORANGE (12)
	g.AddFinding(&RiskFinding{Severity: 3, Likelihood: 2})  // YELLOW (6)
	g.AddFinding(&RiskFinding{Severity: 1, Likelihood: 2})  // GREEN (2)
	
	stats := g.Stats()
	
	if stats.EntityCount != 3 {
		t.Errorf("expected 3 entities, got %d", stats.EntityCount)
	}
	if stats.FindingCount != 4 {
		t.Errorf("expected 4 findings, got %d", stats.FindingCount)
	}
	if stats.RedCount != 1 {
		t.Errorf("expected 1 red, got %d", stats.RedCount)
	}
	if stats.OrangeCount != 1 {
		t.Errorf("expected 1 orange, got %d", stats.OrangeCount)
	}
}

func TestNodeIDGeneration(t *testing.T) {
	id1 := newNodeID("TEST")
	id2 := newNodeID("TEST")
	
	if id1 == id2 {
		t.Error("node IDs should be unique")
	}
	
	if len(id1) < 10 {
		t.Errorf("node ID too short: %s", id1)
	}
}

func TestLinkFindingToEntity(t *testing.T) {
	g := NewLegalGraph("matter-link-test")
	
	entity := g.AddEntity(EntityClause, "第8.2条", nil)
	finding := &RiskFinding{
		Title:    "IP归属风险",
		Severity: 4,
		Likelihood: 3,
	}
	g.AddFinding(finding)
	
	g.LinkFindingToEntity(finding.ID, entity.ID)
	
	// 验证关联
	updated := g.Findings[finding.ID]
	if len(updated.EntityRefs) != 1 {
		t.Errorf("expected 1 entity ref, got %d", len(updated.EntityRefs))
	}
	if updated.EntityRefs[0] != entity.ID {
		t.Errorf("expected ref to %s, got %s", entity.ID, updated.EntityRefs[0])
	}
}

func BenchmarkAddEntity(b *testing.B) {
	g := NewLegalGraph("benchmark")
	for i := 0; i < b.N; i++ {
		g.AddEntity(EntityParty, "Party", map[string]string{"index": string(rune(i))})
	}
}

func BenchmarkCalculateScore(b *testing.B) {
	f := &RiskFinding{Severity: 4, Likelihood: 3}
	for i := 0; i < b.N; i++ {
		f.CalculateScore()
	}
}
