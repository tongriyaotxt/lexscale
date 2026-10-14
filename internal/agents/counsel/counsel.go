// Package counsel 主办律师 Agent——唯一策略生成者。
//
// 对应 ARTEX 的 Planner + PentAGI 的 primary_agent 的合并职能：
//   - 读取双图态势（争议焦点、已有风险点、论证缺口）
//   - 生成下一轮分析意图（Intent），保证论证链不重复、不错序
//   - 持有 per-matter 的 TodoStore 草稿本（跨轮保留策略）
package counsel

import (
	"context"
	"fmt"
	"sync"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Agent 主办律师
type Agent struct {
	provider llm.Provider
	role     llm.ChainRole

	// 草稿本：per-matter 策略记忆（ARTEX TodoStore 的法律版）
	todos   map[string]*TodoStore
	todosMu sync.Mutex
}

// TodoStore 策略草稿本
type TodoStore struct {
	OpenIssues   []string `json:"open_issues"`   // 未解决的争议焦点
	Strategy     string   `json:"strategy"`      // 当前总体策略
	NextSteps    []string `json:"next_steps"`    // 待执行步骤
	Round        int      `json:"round"`         // 已规划轮数
}

// New 创建主办律师
func New(provider llm.Provider) *Agent {
	return &Agent{
		provider: provider,
		role:     llm.RoleCounsel,
		todos:    make(map[string]*TodoStore),
	}
}

// Plan 读取态势，生成意图列表
func (a *Agent) Plan(ctx context.Context, g *graph.LegalGraph, history []*engine.Intent) ([]*engine.Intent, error) {
	a.todosMu.Lock()
	todo, ok := a.todos[g.MatterID]
	if !ok {
		todo = &TodoStore{Round: 0}
		a.todos[g.MatterID] = todo
	}
	todo.Round++
	a.todosMu.Unlock()

	// 构建态势摘要
	situation := a.buildSituation(g, history)

	// 调用 LLM 生成策略
	resp, err := a.provider.Complete(ctx, &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: counselSystemPrompt},
			{Role: "user", Content: situation},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("counsel planning failed: %w", err)
	}

	// 解析意图（简化版：从 LLM 输出提取，生产版用结构化输出）
	intents := a.parseIntents(resp.Content, g.MatterID, todo)

	// 过滤已完成的意图类型（防重复）
	intents = a.dedupIntents(intents, history)

	return intents, nil
}

// buildSituation 构建态势摘要
func (a *Agent) buildSituation(g *graph.LegalGraph, history []*engine.Intent) string {
	stats := g.Stats()
	s := fmt.Sprintf("案件态势:\n- 法律实体: %d\n- 论证主张: %d\n- 风险点: %d (红:%d 橙:%d)\n- 已执行意图: %d\n",
		stats.EntityCount, stats.ClaimCount, stats.FindingCount,
		stats.RedCount, stats.OrangeCount, len(history))

	// 列出已有风险点标题
	if len(g.Findings) > 0 {
		s += "\n已有风险点:\n"
		for _, f := range g.Findings {
			s += fmt.Sprintf("- [%s] %s (S%d×L%d=%d)\n", f.Level, f.Title, f.Severity, f.Likelihood, f.Score)
		}
	}
	return s
}

// parseIntents 解析 LLM 输出为意图列表
func (a *Agent) parseIntents(content string, matterID string, todo *TodoStore) []*engine.Intent {
	// 简化实现：生产版用 structured output / function calling
	// 这里返回基于态势的基础意图
	var intents []*engine.Intent

	// 首轮：先做合同审查
	if todo.Round == 1 {
		intents = append(intents, &engine.Intent{
			Type:     engine.IntentContractReview,
			Goal:     "审查合同全部条款，识别风险点",
			Priority: 10,
		})
	}
	return intents
}

// dedupIntents 去重：同类型意图在历史中已完成则不再生成
func (a *Agent) dedupIntents(intents []*engine.Intent, history []*engine.Intent) []*engine.Intent {
	completed := make(map[engine.IntentType]bool)
	for _, h := range history {
		if h.Status == engine.IntentCompleted {
			completed[h.Type] = true
		}
	}
	var out []*engine.Intent
	for _, it := range intents {
		if !completed[it.Type] {
			out = append(out, it)
		}
	}
	return out
}

const counselSystemPrompt = `你是 LexAGI 的主办律师（Lead Counsel），一个 AI 法律分析团队的策略规划者。

你的职责：
1. 读取案件态势（法律实体、论证主张、风险点）
2. 生成分析意图（Intent）分配给专家团队
3. 保证论证链不重复、不错序

规则：
- 每个风险点必须有出处（法条/判例/条款），无出处的"事实"是幻觉
- 高风险点优先处理（Risk Score 高的先验证）
- 不确定的结论标注为 INFERENCE 或 UNKNOWN，不得伪装成 FACT
- 你生成的每个意图都会经过审批门检查`
