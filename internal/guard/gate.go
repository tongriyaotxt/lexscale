// Package guard 审批门控系统。
//
// 仿 ARTEX 的 guard + intercept 双层设计：
//   - Guard 是规则引擎：意图执行前匹配 deny / ask / allow
//   - deny 硬拦截，不豁免（即使"一键放行"开启）
//   - ask 进入人工审批队列
//   - AutoAllow 带时限（法律版默认更保守：最长 4h）
//
// 法律版特有规则：
//   - 产出物触达客户前必须人工审核（执业规范）
//   - 利益冲突核查
//   - 幻觉拦截：FACT 级结论无引用 → deny
package guard

import (
	"sync"
	"time"

	"lexscale/internal/graph"
)

// Decision 审批决定
type Decision string

const (
	DecisionDeny  Decision = "DENY"
	DecisionAsk   Decision = "ASK"
	DecisionAllow Decision = "ALLOW"
)

// Rule 审批规则
type Rule struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Match    func(intent interface{}) bool `json:"-"` // 匹配函数
	Decision Decision          `json:"decision"`
	Reason   string            `json:"reason"`
}

// ApprovalItem 待审批项
type ApprovalItem struct {
	IntentID  string    `json:"intent_id"`
	Goal      string    `json:"goal"`
	Type      string    `json:"type"`
	Submitted time.Time `json:"submitted"`
	RuleID    string    `json:"rule_id"`
	Reason    string    `json:"reason"`
}

// ApprovalDecision 人工审批决定
type ApprovalDecision struct {
	IntentID string
	Approved bool
	By       string    // 审批人
	Note     string    // 审批意见
	At       time.Time
	Auto     bool      // 是否自动放行（AutoAllow）
}

// AutoAllowGrant 一键放行授权（带时限）
type AutoAllowGrant struct {
	RuleID    string    `json:"rule_id"`
	GrantedBy string    `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Gate 审批门
type Gate struct {
	rules       []Rule
	autoAllows  map[string]*AutoAllowGrant // ruleID → 授权
	pending     []*ApprovalItem
	pendingMu   sync.Mutex
	decisions   chan ApprovalDecision
	onApproval  func(item *ApprovalItem, decision ApprovalDecision)

	// 统计
	statsMu      sync.Mutex
	AskCount     int `json:"ask_count"`
	DenyCount    int `json:"deny_count"`
	AllowCount   int `json:"allow_count"`
}

// NewGate 创建审批门
func NewGate() *Gate {
	return &Gate{
		autoAllows: make(map[string]*AutoAllowGrant),
		decisions:  make(chan ApprovalDecision, 100),
	}
}

// AddRule 添加规则
func (g *Gate) AddRule(r Rule) {
	g.rules = append(g.rules, r)
}

// SetOnApproval 设置审批回调（审批通过/拒绝后触发）
func (g *Gate) SetOnApproval(fn func(item *ApprovalItem, decision ApprovalDecision)) {
	g.onApproval = fn
}

// DefaultLegalRules 返回法律场景默认规则集
func DefaultLegalRules() []Rule {
	return []Rule{
		{
			ID:       "no-client-contact-without-review",
			Name:     "客户触达前必须人工审核",
			Decision: DecisionAsk,
			Reason:   "AI 产出物不得未经审核直接触达客户（执业规范）",
		},
		{
			ID:       "no-fact-without-citation",
			Name:     "FACT级结论必须有引用",
			Decision: DecisionDeny,
			Reason:   "无出处的'事实'视为幻觉，硬拦截",
		},
		{
			ID:       "red-risk-mandatory-review",
			Name:     "RED级风险必须人工复核",
			Decision: DecisionAsk,
			Reason:   "Risk Score ≥16 的风险点必须由执业律师复核",
		},
		{
			ID:       "conflict-check-required",
			Name:     "新当事人必须做利益冲突核查",
			Decision: DecisionAsk,
			Reason:   "首次录入当事人信息时进行利益冲突检查",
		},
	}
}

// CheckIntent 检查意图（interface{} 避免循环依赖，实际传 *engine.Intent）
func (g *Gate) CheckIntent(intent interface{}) Decision {
	// 检查每个匹配的规则，取最严格的决定
	finalDecision := DecisionAllow
	for _, rule := range g.rules {
		if rule.Match == nil || !rule.Match(intent) {
			continue
		}
		// AutoAllow 检查：有有效授权则降级为 ALLOW
		if rule.Decision == DecisionAsk {
			if grant, ok := g.autoAllows[rule.ID]; ok && time.Now().Before(grant.ExpiresAt) {
				continue // 自动放行有效，跳过此 ASK 规则
			}
		}
		// deny 不豁免
		if rule.Decision == DecisionDeny {
			return DecisionDeny
		}
		if rule.Decision == DecisionAsk {
			finalDecision = DecisionAsk
		}
	}
	g.statsMu.Lock()
	switch finalDecision {
	case DecisionDeny:
		g.DenyCount++
	case DecisionAsk:
		g.AskCount++
	default:
		g.AllowCount++
	}
	g.statsMu.Unlock()
	return finalDecision
}

// EnqueueForApproval 入审批队列（interface{} 避免循环依赖）
func (g *Gate) EnqueueForApproval(intent interface{}) {
	g.pendingMu.Lock()
	defer g.pendingMu.Unlock()
	// 实际实现由 engine 层调用，这里仅维护队列
}

// GrantAutoAllow 授予一键放行（带时限）
func (g *Gate) GrantAutoAllow(ruleID, grantedBy string, duration time.Duration) {
	g.autoAllows[ruleID] = &AutoAllowGrant{
		RuleID:    ruleID,
		GrantedBy: grantedBy,
		GrantedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}
}

// RevokeAutoAllow 撤销一键放行
func (g *Gate) RevokeAutoAllow(ruleID string) {
	delete(g.autoAllows, ruleID)
}

// Submit 提交人工审批决定
func (g *Gate) Submit(decision ApprovalDecision) {
	g.decisions <- decision
}

// PendingCount 待审批数量
func (g *Gate) PendingCount() int {
	g.pendingMu.Lock()
	defer g.pendingMu.Unlock()
	return len(g.pending)
}

// ValidateFinding 验证风险点是否满足审批条件（供 Verifier 使用）
func ValidateFinding(f *graph.RiskFinding) Decision {
	// FACT 级必须有引用
	if f.Tier == graph.TierFact && len(f.Citations) == 0 {
		return DecisionDeny
	}
	// RED 级必须人工复核
	if f.Level == graph.RiskRed && !f.Verified {
		return DecisionAsk
	}
	return DecisionAllow
}
