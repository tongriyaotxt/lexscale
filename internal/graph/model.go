// Package graph 实现 LexAGI 的双图数据模型：
// 图1 法律关系图（LegalEntityGraph）——当事人/合同/条款/义务/权利/证据/法域
// 图2 论证图（ArgumentationGraph）——主张/抗辩/证据/法条/判例/结论
// 风险点（RiskFinding）是两图的锚点。
//
// 设计来源：ARTEX 双图模型（资产图+探索图，finding 为锚点）的法律领域映射。
// 证据三层模型来源：REA 的 fact/inference/unknown 分离，法律场景下用于反幻觉。
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// EvidenceTier 证据三层类型：事实 / 推断 / 未知。
// 这是反幻觉的核心机制——无出处的"事实"会被 verifier 降级或打回。
type EvidenceTier string

const (
	TierFact      EvidenceTier = "FACT"      // 有明确出处（法条号/判例号/条款号/证据编号）
	TierInference EvidenceTier = "INFERENCE" // 模型推理链产物，必须附推理依据
	TierUnknown   EvidenceTier = "UNKNOWN"   // 明确标注"还不知道什么"
)

// Citation 出处引用。FACT 级结论必须有至少一个 Citation，否则 verifier 拦截。
type Citation struct {
	SourceType string `json:"source_type"` // statute(法条) / precedent(判例) / clause(合同条款) / evidence(证据) / regulation(法规)
	SourceRef  string `json:"source_ref"`  // 如 "民法典第577条" / "(2021)京01民终1234号" / "合同-服务协议-第8.2条"
	Locator    string `json:"locator"`     // 精确位置（页码/段落/行号）
	Excerpt    string `json:"excerpt"`     // 原文摘录
}

// NodeID 图节点唯一标识。
type NodeID string

func newNodeID(prefix string) NodeID {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())))
	return NodeID(prefix + "_" + hex.EncodeToString(h[:6]))
}

// ==================== 图1：法律关系图 ====================

// EntityType 法律关系图节点类型
type EntityType string

const (
	EntityParty     EntityType = "PARTY"     // 当事人（自然人/法人）
	EntityContract  EntityType = "CONTRACT"  // 合同
	EntityClause    EntityType = "CLAUSE"    // 条款
	EntityObligation EntityType = "OBLIGATION" // 义务
	EntityRight     EntityType = "RIGHT"     // 权利
	EntityEvidence  EntityType = "EVIDENCE"  // 证据
	EntityJurisdiction EntityType = "JURISDICTION" // 法域
	EntityStatute   EntityType = "STATUTE"   // 法条
)

// EntityNode 法律关系图节点
type EntityNode struct {
	ID         NodeID            `json:"id"`
	Type       EntityType        `json:"type"`
	Name       string            `json:"name"`        // 显示名
	Attributes map[string]string `json:"attributes"`  // 扩展属性（如身份证号、合同金额）
	CreatedAt  time.Time         `json:"created_at"`
}

// EntityEdge 法律关系图边
type EntityEdge struct {
	ID         NodeID     `json:"id"`
	From       NodeID     `json:"from"`
	To         NodeID     `json:"to"`
	Relation   string     `json:"relation"`   // 签订/包含/设定/证明/适用/约束
	CreatedAt  time.Time  `json:"created_at"`
}

// ==================== 图2：论证图 ====================

// ClaimStatus 主张状态
type ClaimStatus string

const (
	ClaimOpen       ClaimStatus = "OPEN"       // 待论证
	ClaimSupported  ClaimStatus = "SUPPORTED"  // 已获支持
	ClaimWeakened   ClaimStatus = "WEAKENED"   // 存在削弱因素
	ClaimRefuted    ClaimStatus = "REFUTED"    // 被反证推翻
	ClaimUndetermined ClaimStatus = "UNDETERMINED" // 无法确定
)

// ClaimNode 论证图节点（主张）
type ClaimNode struct {
	ID          NodeID       `json:"id"`
	Statement   string       `json:"statement"`    // 主张内容
	Status      ClaimStatus  `json:"status"`
	Support     []SupportEdge `json:"support"`     // 支持/削弱关系
	CreatedAt   time.Time    `json:"created_at"`
}

// SupportType 支持类型
type SupportType string

const (
	SupportStatute   SupportType = "STATUTE"   // 法条支持
	SupportPrecedent SupportType = "PRECEDENT" // 判例支持
	SupportEvidence  SupportType = "EVIDENCE"  // 证据支持
	SupportLogic     SupportType = "LOGIC"     // 逻辑推理（INFERENCE 级）
	CounterArgument  SupportType = "COUNTER"   // 抗辩/削弱
)

// SupportEdge 论证支持边
type SupportEdge struct {
	ID       NodeID      `json:"id"`
	Type     SupportType `json:"type"`
	Target   NodeID      `json:"target"`   // 指向法条/判例/证据节点
	Strength float64     `json:"strength"` // 支持强度 0-1
	Reasoning string     `json:"reasoning"` // 推理说明（INFERENCE 级必填）
	Citations []Citation `json:"citations"`
	Tier      EvidenceTier `json:"tier"`
}

// ==================== 锚点：风险点 ====================

// RiskLevel 风险等级（severity × likelihood 矩阵映射）
type RiskLevel string

const (
	RiskGreen  RiskLevel = "GREEN"   // 1-4 分
	RiskYellow RiskLevel = "YELLOW"  // 5-9
	RiskOrange RiskLevel = "ORANGE"  // 10-15
	RiskRed    RiskLevel = "RED"     // 16-25
)

// RiskFinding 法律风险点——双图锚点。
// 一边挂在法律关系图实体上（哪个条款/当事人有风险），
// 一边是论证图的产物（风险结论从哪条论证链推出）。
type RiskFinding struct {
	ID          NodeID       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Severity    int          `json:"severity"`   // 1-5
	Likelihood  int          `json:"likelihood"` // 1-5
	Score       int          `json:"score"`      // severity × likelihood
	Level       RiskLevel    `json:"level"`

	EntityRefs []NodeID `json:"entity_refs"` // 关联的法律关系图节点
	ClaimRef   NodeID   `json:"claim_ref"`   // 来源论证链

	Tier      EvidenceTier `json:"tier"`
	Citations []Citation   `json:"citations"`

	// 复核状态（审批门联动）
	Verified    bool      `json:"verified"`
	VerifiedBy  string    `json:"verified_by"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
}

// CalculateScore 计算风险分数与等级
func (r *RiskFinding) CalculateScore() {
	r.Score = r.Severity * r.Likelihood
	switch {
	case r.Score >= 16:
		r.Level = RiskRed
	case r.Score >= 10:
		r.Level = RiskOrange
	case r.Score >= 5:
		r.Level = RiskYellow
	default:
		r.Level = RiskGreen
	}
}

// ==================== 图容器 ====================

// LegalGraph 双图容器，按案件（Matter）隔离——防串案隔离的基础。
type LegalGraph struct {
	MatterID string                 `json:"matter_id"`
	Entities map[NodeID]*EntityNode `json:"entities"`
	Edges    map[NodeID]*EntityEdge `json:"edges"`
	Claims   map[NodeID]*ClaimNode  `json:"claims"`
	Findings map[NodeID]*RiskFinding `json:"findings"`
}

// NewLegalGraph 创建按案件隔离的双图
func NewLegalGraph(matterID string) *LegalGraph {
	return &LegalGraph{
		MatterID: matterID,
		Entities: make(map[NodeID]*EntityNode),
		Edges:    make(map[NodeID]*EntityEdge),
		Claims:   make(map[NodeID]*ClaimNode),
		Findings: make(map[NodeID]*RiskFinding),
	}
}

// AddEntity 添加法律关系节点
func (g *LegalGraph) AddEntity(t EntityType, name string, attrs map[string]string) *EntityNode {
	n := &EntityNode{
		ID:         newNodeID(string(t)),
		Type:       t,
		Name:       name,
		Attributes: attrs,
		CreatedAt:  time.Now(),
	}
	g.Entities[n.ID] = n
	return n
}

// AddFinding 添加风险点（自动算分）
func (g *LegalGraph) AddFinding(f *RiskFinding) {
	f.ID = newNodeID("FINDING")
	f.CalculateScore()
	g.Findings[f.ID] = f
}

// LinkFindingToEntity 锚点关联：风险点 ↔ 法律实体
func (g *LegalGraph) LinkFindingToEntity(findingID, entityID NodeID) {
	if f, ok := g.Findings[findingID]; ok {
		f.EntityRefs = append(f.EntityRefs, entityID)
	}
}

// MatterStats 案件统计
type MatterStats struct {
	EntityCount  int `json:"entity_count"`
	ClaimCount   int `json:"claim_count"`
	FindingCount int `json:"finding_count"`
	RedCount     int `json:"red_count"`
	OrangeCount  int `json:"orange_count"`
}

// Stats 返回案件统计
func (g *LegalGraph) Stats() MatterStats {
	s := MatterStats{
		EntityCount:  len(g.Entities),
		ClaimCount:   len(g.Claims),
		FindingCount: len(g.Findings),
	}
	for _, f := range g.Findings {
		switch f.Level {
		case RiskRed:
			s.RedCount++
		case RiskOrange:
			s.OrangeCount++
		}
	}
	return s
}
