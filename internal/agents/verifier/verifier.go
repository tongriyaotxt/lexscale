// Package verifier 论证验证 Worker——对每个风险点做反证验证。
//
// 对应 ARTEX 的 Verifier：对 finding 做反证验证。
// 法律版核心职能：反幻觉——检查每个 FACT 级结论是否有真引用。
package verifier

import (
	"context"
	"fmt"
	"strings"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Worker 论证验证师
type Worker struct {
	provider llm.Provider
	id       string
}

// New 创建验证师
func New(provider llm.Provider) *Worker {
	return &Worker{provider: provider, id: "verifier-1"}
}

func (w *Worker) ID() string { return w.id }

func (w *Worker) CanHandle(t engine.IntentType) bool {
	return t == engine.IntentVerifyFinding
}

// Execute 执行反证验证
func (w *Worker) Execute(ctx context.Context, intent *engine.Intent, g *graph.LegalGraph) (*engine.IntentResult, error) {
	// 找出未验证的风险点
	var unverified []*graph.RiskFinding
	for _, f := range g.Findings {
		if !f.Verified {
			unverified = append(unverified, f)
		}
	}

	if len(unverified) == 0 {
		return &engine.IntentResult{Summary: "所有风险点已验证"}, nil
	}

	var verifiedCount int
	var rejected []*graph.RiskFinding

	for _, f := range unverified {
		prompt := buildVerifyPrompt(f)
		resp, err := w.provider.Complete(ctx, &llm.Request{
			Messages: []llm.Message{
				{Role: "system", Content: verifierSystemPrompt},
				{Role: "user", Content: prompt},
			},
			Temperature: 0.1, // 验证用低温度，减少创造性
		})
		if err != nil {
			continue
		}

		// 验证逻辑
		if w.validate(f, resp.Content) {
			f.Verified = true
			now := f.VerifiedAt
			_ = now
			verifiedCount++
		} else {
			// 幻觉打回：降级为 INFERENCE 或标记 UNKNOWN
			if f.Tier == graph.TierFact && len(f.Citations) == 0 {
				f.Tier = graph.TierInference
				rejected = append(rejected, f)
			}
		}
	}

	result := &engine.IntentResult{
		Summary: fmt.Sprintf("验证完成: %d/%d 通过", verifiedCount, len(unverified)),
	}
	if len(rejected) > 0 {
		result.Summary += fmt.Sprintf(", %d 个 FACT 级结论因无引用被降级为 INFERENCE", len(rejected))
	}
	return result, nil
}

func buildVerifyPrompt(f *graph.RiskFinding) string {
	var cites string
	for _, c := range f.Citations {
		cites += "- " + c.SourceType + ": " + c.SourceRef + "\n"
	}
	return "请反证验证以下法律风险点：\n\n标题: " + f.Title +
		"\n描述: " + f.Description +
		"\n引用:\n" + cites +
		"\n\n要求：\n1. 检查引用是否真实存在且支持结论\n2. 检查是否有相反的判例或法条被忽略\n3. 结论是否有逻辑漏洞\n4. 回复 VALID 或 REJECTED: [原因]"
}

// validate 验证结果判定
func (w *Worker) validate(f *graph.RiskFinding, response string) bool {
	response = strings.ToUpper(response)
	// FACT 级必须有引用
	if f.Tier == graph.TierFact && len(f.Citations) == 0 {
		return false
	}
	// 验证器返回 REJECTED 则不通过
	if strings.Contains(response, "REJECTED") {
		return false
	}
	return true
}

const verifierSystemPrompt = `你是 LexAGI 的论证验证师，负责对法律风险点做反证验证。

你的存在只有一个目的：抓出幻觉。

铁律：
1. FACT 级结论必须有真实可查的引用——检查引用是否真实存在
2. 主动寻找相反的判例、被忽略的法条、逻辑漏洞
3. 宁可错杀不可放过——拿不准就 REJECTED
4. 被 REJECTED 的结论将降级为 INFERENCE 并打回重新论证
5. 你是 AI 法律系统的最后一道防线——你的松懈等于当事人的损失`
