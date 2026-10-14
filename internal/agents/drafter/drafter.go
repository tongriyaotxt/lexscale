// Package drafter 文书起草师 Agent
//
// 职责：起草法律意见书、合同条款、律师函、起诉状等
// 输出：Artifact（文档）+ Claim（法律论证）
//
// 安全约束：
//   - 所有产出物标注 "DRAFT - 未经律师审核"
//   - 引用必须来自已验证的 FACT 级 Claim
//   - 产出物必须通过 guard 审批后才能标记为 FINAL
//   - 关键条款必须给出备选方案和风险提示
package drafter

import (
	"context"
	"fmt"
	"time"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Drafter 文书起草师
type Drafter struct {
	llm   *llm.Router
	graph *graph.LegalGraph
}

// NewDrafter 创建起草师
func NewDrafter(r *llm.Router, lg *graph.LegalGraph) *Drafter {
	return &Drafter{llm: r, graph: lg}
}

// ID 返回 Worker 名称
func (d *Drafter) ID() string { return "drafter" }

// CanHandle 能处理起草类意图
func (d *Drafter) CanHandle(intentType engine.IntentType) bool {
	return intentType == engine.IntentDraftDocument
}

// Execute 执行起草
func (d *Drafter) Execute(ctx context.Context, intent *engine.Intent, g *graph.LegalGraph) (*engine.IntentResult, error) {
	// 获取已有风险发现作为参考
	findings := g.Findings

	systemPrompt := fmt.Sprintf(`你是一名专业法律文书起草师。你的任务是起草高质量的法律文书。

规则：
1. 所有文书必须标注 "DRAFT - 未经律师审核，不得直接使用"
2. 关键条款必须给出至少一个备选方案
3. 必须包含风险提示段落
4. 使用专业法律语言，避免歧义
5. 如果输入信息不足，列出需要补充的信息清单

当前案件风险发现：
%s`, formatFindings(findings))

	userPrompt := fmt.Sprintf(`请起草以下法律文书：

要求：%s

请按专业法律文书格式输出。`, intent.Goal)

	resp, err := d.llm.Complete(ctx, llm.RoleDrafter, &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4, // 起草需要一定创造性
		MaxTokens:   8192, // 文书较长
	})
	if err != nil {
		return &engine.IntentResult{Error: fmt.Sprintf("起草失败: %v", err)}, err
	}

	// 创建 Claim 记录起草行为
	claim := &graph.ClaimNode{
		ID:        graph.NodeID(fmt.Sprintf("claim-draft-%d", time.Now().UnixNano())),
		Statement: fmt.Sprintf("已起草文书：%s（草稿状态，待律师审核）", intent.Goal),
		Status:    graph.ClaimOpen,
		Support:   []graph.SupportEdge{},
		CreatedAt: time.Now(),
	}

	return &engine.IntentResult{
		Summary: fmt.Sprintf("文书起草完成（草稿，待审核）\n\n%s", resp.Content),
		Claims:  []*graph.ClaimNode{claim},
	}, nil
}

// formatFindings 格式化风险发现
func formatFindings(findings map[graph.NodeID]*graph.RiskFinding) string {
	if len(findings) == 0 {
		return "（暂无风险发现）"
	}
	result := ""
	for _, f := range findings {
		result += fmt.Sprintf("- [%s] %s: %s\n", f.Level, f.Title, f.Description)
	}
	return result
}
