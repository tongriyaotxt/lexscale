// Package assessor 风险评级师 Agent
//
// 职责：综合案件所有信息，评估整体风险等级
// 输出：RiskFinding（风险发现）+ 风险报告
//
// 安全约束：
//   - 使用 severity × likelihood 矩阵评分
//   - RED 级风险必须标注"建议寻求专业律师意见"
//   - 风险评级是 INFERENCE 级别，不是 FACT
//   - 不替代律师判断，只是辅助工具
package assessor

import (
	"context"
	"fmt"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Assessor 风险评级师
type Assessor struct {
	llm   *llm.Router
	graph *graph.LegalGraph
}

// NewAssessor 创建评级师
func NewAssessor(r *llm.Router, lg *graph.LegalGraph) *Assessor {
	return &Assessor{llm: r, graph: lg}
}

// ID 返回 Worker 名称
func (a *Assessor) ID() string { return "assessor" }

// CanHandle 能处理风险评估类意图
func (a *Assessor) CanHandle(intentType engine.IntentType) bool {
	return intentType == engine.IntentRiskAssess
}

// Execute 执行风险评估
func (a *Assessor) Execute(ctx context.Context, intent *engine.Intent, g *graph.LegalGraph) (*engine.IntentResult, error) {
	// 获取所有实体、Claim、已有发现
	entities := g.Entities
	claims := g.Claims
	existingFindings := g.Findings

	systemPrompt := `你是一名法律风险评估师。你的任务是对案件进行全面的风险评估。

评估矩阵：
- Severity: 1(轻微) - 5(致命)
- Likelihood: 1(极不可能) - 5(几乎确定)
- Score = Severity × Likelihood
  - Score ≥ 16: RED（必须立即处理）
  - Score ≥ 10: ORANGE（高度重视）
  - Score ≥ 5:  YELLOW（需要关注）
  - Score < 5:  GREEN（可接受）

规则：
1. 每个风险发现必须说明严重度和可能性评分依据
2. RED 级风险必须标注"建议寻求专业律师意见"
3. 风险评级仅供参考，不替代律师判断
4. 考虑法律风险、商业风险、声誉风险
5. 输出风险缓解建议`

	userPrompt := fmt.Sprintf(`请对以下案件进行风险评估：

案件主体：%d 个
论证节点：%d 个
已有风险发现：%d 个

已有发现详情：
%s

请输出完整的风险评估报告，包含：
1. 总体风险等级
2. 各风险项详细评分
3. 风险缓解建议
4. 需要律师重点关注的领域`, len(entities), len(claims), len(existingFindings), formatFindings(existingFindings))

	resp, err := a.llm.Complete(ctx, llm.RoleAssessor, &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   4096,
	})
	if err != nil {
		return &engine.IntentResult{Error: fmt.Sprintf("风险评估失败: %v", err)}, err
	}

	// 创建综合风险发现
	finding := &graph.RiskFinding{
		Title:       "综合风险评估",
		Description: fmt.Sprintf("综合风险评估完成，基于 %d 个已有发现\n\n%s", len(existingFindings), resp.Content),
		Severity:    calculateOverallSeverity(existingFindings),
		Likelihood:  calculateOverallLikelihood(existingFindings),
		EntityRefs:  []graph.NodeID{},
		Tier:        graph.TierInference,
	}

	return &engine.IntentResult{
		Summary:  "风险评估报告已生成",
		Findings: []*graph.RiskFinding{finding},
	}, nil
}

// formatFindings 格式化已有发现
func formatFindings(findings map[graph.NodeID]*graph.RiskFinding) string {
	if len(findings) == 0 {
		return "（暂无）"
	}
	result := ""
	for _, f := range findings {
		result += fmt.Sprintf("- [%s] %s (S:%d L:%d Score:%d)\n",
			f.Level, f.Description, f.Severity, f.Likelihood, f.Score)
	}
	return result
}

// calculateOverallSeverity 计算综合严重度
func calculateOverallSeverity(findings map[graph.NodeID]*graph.RiskFinding) int {
	max := 1
	for _, f := range findings {
		if f.Severity > max {
			max = f.Severity
		}
	}
	return max
}

// calculateOverallLikelihood 计算综合可能性
func calculateOverallLikelihood(findings map[graph.NodeID]*graph.RiskFinding) int {
	max := 1
	for _, f := range findings {
		if f.Likelihood > max {
			max = f.Likelihood
		}
	}
	return max
}
