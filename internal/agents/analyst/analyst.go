// Package analyst 合同分析 Worker——条款级审查。
package analyst

import (
	"context"
	"strings"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Worker 合同分析师
type Worker struct {
	provider llm.Provider
	id       string
}

// New 创建合同分析师
func New(provider llm.Provider) *Worker {
	return &Worker{provider: provider, id: "contract-analyst-1"}
}

func (w *Worker) ID() string { return w.id }

func (w *Worker) CanHandle(t engine.IntentType) bool {
	return t == engine.IntentContractReview
}

// Execute 执行合同审查
func (w *Worker) Execute(ctx context.Context, intent *engine.Intent, g *graph.LegalGraph) (*engine.IntentResult, error) {
	clauseText, _ := intent.Context["clause_text"]
	clauseName, _ := intent.Context["clause_name"]

	prompt := buildReviewPrompt(clauseName, clauseText)
	resp, err := w.provider.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: analystSystemPrompt},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	// 解析产出：风险点 + 实体
	result := &engine.IntentResult{
		Summary: resp.Content,
	}

	// 从分析结果中提取风险点（生产版用结构化输出）
	findings := extractFindings(resp.Content, clauseName)
	for _, f := range findings {
		g.AddFinding(f)
	}
	result.Findings = findings

	return result, nil
}

func buildReviewPrompt(name, text string) string {
	return "请审查以下合同条款，识别法律风险：\n\n条款名称: " + name + "\n条款内容:\n" + text +
		"\n\n要求：\n1. 每个风险点标注 severity(1-5) 和 likelihood(1-5)\n2. 引用相关法律依据\n3. 标注是 FACT 还是 INFERENCE"
}

// extractFindings 从 LLM 输出提取风险点
func extractFindings(content, clauseName string) []*graph.RiskFinding {
	// 简化实现：生产版用 JSON schema 结构化输出
	if strings.Contains(content, "无风险") || len(content) < 10 {
		return nil
	}
	f := &graph.RiskFinding{
		Title:      "条款风险: " + clauseName,
		Description: content[:min(len(content), 500)],
		Severity:   3,
		Likelihood: 2,
		Tier:       graph.TierInference,
	}
	f.CalculateScore()
	return []*graph.RiskFinding{f}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const analystSystemPrompt = `你是 LexAGI 的合同分析师，负责条款级法律风险审查。

铁律：
1. 每个风险点必须引用法律依据（法条号/判例号），无引用即标记 INFERENCE
2. severity(1-5): 1=可忽略 5=严重（可能造成重大损失）
3. likelihood(1-5): 1=极不可能 5=几乎确定
4. 不确定的写 UNKNOWN，禁止编造法条
5. 你的产出会经过论证验证师（Verifier）反证审查——幻觉会被打回`
