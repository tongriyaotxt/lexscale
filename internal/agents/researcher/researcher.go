// Package researcher 法律检索员 Agent
//
// 职责：检索法条、判例、法规、司法解释
// 输出：LegalEntity（法条/判例实体）+ Claim（检索发现）
//
// 安全约束：
//   - 只输出有引用的结果，没有引用的结果标注 UNKNOWN 层级
//   - 不生成法律意见，只做检索和引用
//   - 检索结果必须通过 Verifier 验证后才能升级为 FACT
package researcher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/llm"
)

// Researcher 法律检索员
type Researcher struct {
	llm   *llm.Router
	graph *graph.LegalGraph
}

// NewResearcher 创建检索员
func NewResearcher(r *llm.Router, lg *graph.LegalGraph) *Researcher {
	return &Researcher{llm: r, graph: lg}
}

// ID 返回 Worker 名称
func (r *Researcher) ID() string { return "researcher" }

// CanHandle 能处理检索类意图
func (r *Researcher) CanHandle(intentType engine.IntentType) bool {
	switch intentType {
	case engine.IntentStatuteResearch, engine.IntentPrecedentSearch, engine.IntentComplianceCheck:
		return true
	}
	return false
}

// Execute 执行检索
func (r *Researcher) Execute(ctx context.Context, intent *engine.Intent, g *graph.LegalGraph) (*engine.IntentResult, error) {
	// 构建检索提示
	systemPrompt := `你是一名法律检索员。你的任务是检索相关法律法规和判例。

规则：
1. 每个检索结果必须包含准确的法律来源引用
2. 如果没有找到相关法条，明确说明"未找到"
3. 不要编造法律条文
4. 引用格式：法律名称 + 条款号 + 发布机关 + 生效日期
5. 使用中文检索中国法律，使用英文检索国际条约和外国判例`

	userPrompt := fmt.Sprintf(`请检索以下法律问题相关的法条和判例：

问题：%s

请按以下格式输出每个检索结果：
---
法条名称：[法律名称]
条款号：[第X条]
发布机关：[机关名称]
生效日期：[日期]
内容摘要：[简要内容]
相关性评分：[1-10]
---

如果没有找到相关结果，请输出"未找到相关法律条文"。`, intent.Goal)

	resp, err := r.llm.Complete(ctx, llm.RoleResearcher, &llm.Request{
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1, // 极低温度，检索要精确
		MaxTokens:   4096,
	})
	if err != nil {
		return &engine.IntentResult{Error: fmt.Sprintf("检索失败: %v", err)}, err
	}

	// 解析检索结果，创建实体和主张
	entities, claims := r.parseResults(resp.Content)

	result := &engine.IntentResult{
		Summary:  fmt.Sprintf("检索完成，找到 %d 个相关法条/判例", len(entities)),
		Entities: entities,
		Claims:   claims,
	}

	return result, nil
}

// parseResults 解析检索结果
func (r *Researcher) parseResults(content string) ([]*graph.EntityNode, []*graph.ClaimNode) {
	var entities []*graph.EntityNode
	var claims []*graph.ClaimNode

	// 按 --- 分隔解析
	blocks := strings.Split(content, "---")
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" || strings.Contains(block, "未找到") {
			continue
		}

		name := extractField(block, "法条名称")
		if name == "" {
			continue
		}

		entity := &graph.EntityNode{
			ID:   graph.NodeID(fmt.Sprintf("statute-%d", time.Now().UnixNano())),
			Type: graph.EntityStatute,
			Name: name,
			Attributes: map[string]string{
				"条款号":   extractField(block, "条款号"),
				"发布机关": extractField(block, "发布机关"),
				"生效日期": extractField(block, "生效日期"),
				"内容摘要": extractField(block, "内容摘要"),
			},
			CreatedAt: time.Now(),
		}

		// 构建引用
		citation := graph.Citation{
			SourceType: "statute",
			SourceRef:  name + " " + extractField(block, "条款号"),
			Locator:    extractField(block, "条款号"),
			Excerpt:    extractField(block, "内容摘要"),
		}

		// 构建 Claim（检索发现）
		claim := &graph.ClaimNode{
			ID:        graph.NodeID(fmt.Sprintf("claim-res-%d", time.Now().UnixNano())),
			Statement: fmt.Sprintf("检索发现：%s", extractField(block, "内容摘要")),
			Status:    graph.ClaimOpen,
			Support: []graph.SupportEdge{
				{
					ID:        graph.NodeID(fmt.Sprintf("support-%d", time.Now().UnixNano())),
					Type:      graph.SupportStatute,
					Target:    entity.ID,
					Strength:  0.7,
					Reasoning: "法律检索结果，待验证",
					Citations: []graph.Citation{citation},
					Tier:      graph.TierInference, // 检索结果是 INFERENCE 级别
				},
			},
			CreatedAt: time.Now(),
		}

		entities = append(entities, entity)
		claims = append(claims, claim)
	}

	return entities, claims
}

// extractField 从文本块中提取字段
func extractField(block, field string) string {
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, field+"：") || strings.HasPrefix(line, field+":") {
			val := strings.TrimPrefix(strings.TrimPrefix(line, field+"："), field+":")
			return strings.TrimSpace(val)
		}
	}
	return ""
}
