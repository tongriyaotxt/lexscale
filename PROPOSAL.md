# LexAGI Framework — 开源法律 Agent 框架方案

## 一句话定位

> **"法律界的 LangChain"** — 让开发者 30 分钟搭出一个企业级法律 AI Agent

---

## 为什么选这个方向？

### 市场空白

| 框架 | 定位 | 法律支持 |
|------|------|----------|
| LangChain | 通用 Agent 框架 | ❌ 无 |
| LlamaIndex | 通用 RAG | ❌ 无 |
| Harvey | 法律 AI 产品 | ❌ 闭源，不开放 |
| CoCounsel | 法律 AI 产品 | ❌ 闭源，不开放 |
| **LexAGI** | **法律 Agent 框架** | ✅ **专注法律** |

### 法律 AI 开发者的痛点

```
想做一个法律 AI 产品，你需要：

1. 证据链管理（FACT/INFERENCE/UNKNOWN）——自己设计
2. 审批门（防止 AI 幻觉害死人）——自己设计
3. 法律关系图 + 论证图——自己设计
4. 法条引用验证——自己设计
5. 风险评级体系——自己设计
6. 审计合规（法律行业刚需）——自己设计

→ 每个团队都在重复造轮子！
→ LexAGI 把这些全部标准化！
```

---

## 核心架构

```
┌─────────────────────────────────────────────────────┐
│                   开发者应用层                        │
│  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │
│  │ 合同审查  │  │ 法律咨询  │  │ 合规检查          │  │
│  │ Chatbot  │  │ 文书生成  │  │ 尽调报告          │  │
│  └────┬─────┘  └────┬─────┘  └───────┬──────────┘  │
│       │              │                │              │
├───────┼──────────────┼────────────────┼──────────────┤
│       ▼              ▼                ▼              │
│  ┌────────────────────────────────────────────────┐ │
│  │           LexAGI Framework                      │ │
│  │                                                 │ │
│  │  ┌─────────┐  ┌─────────┐  ┌───────────────┐  │ │
│  │  │  双图模型 │  │  审批门  │  │  多 Agent 调度 │  │ │
│  │  │ Graph    │  │ Guard    │  │  Engine       │  │ │
│  │  └────┬─────┘  └────┬────┘  └──────┬────────┘  │ │
│  │       │              │               │           │ │
│  │  ┌────┴──────────────┴───────────────┴────────┐ │ │
│  │  │           三层证据体系                      │ │ │
│  │  │  FACT（必须有引用）                         │ │ │
│  │  │  INFERENCE（必须有推理链）                   │ │ │
│  │  │  UNKNOWN（明确标注不确定）                   │ │ │
│  │  └─────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────┘ │
│       │              │               │                  │
├───────┼──────────────┼───────────────┼──────────────────┤
│       ▼              ▼               ▼                  │
│  ┌─────────┐  ┌─────────┐  ┌──────────────────┐      │
│  │ OpenAI  │  │ Claude  │  │ 国产模型          │      │
│  │ GPT-4   │  │ Sonnet  │  │ Kimi/DeepSeek    │      │
│  └─────────┘  └─────────┘  └──────────────────┘      │
└─────────────────────────────────────────────────────┘
```

---

## 核心模块（已实现的）

### 1. 双图模型（`internal/graph`）⭐ 独创

```
实体图（Entity Graph）          论证图（Argument Graph）
┌──────────┐                  ┌──────────┐
│  当事人A  │                  │ 主张1     │◄── FACT级
│  当事人B  │                  │ 主张2     │◄── INFERENCE级
│  合同     │                  │ 主张3     │◄── UNKNOWN级
│  条款     │                  └──────────┘
│  法条     │
└──────────┘
```

**为什么双图？**
- 实体图回答"**谁和谁什么关系**"
- 论证图回答"**这个结论怎么推出来的**"
- 法律 AI 必须同时回答两个问题

### 2. 三层证据体系 ⭐ 解决幻觉

```go
// FACT — 必须有法条/判例引用
finding := &RiskFinding{
    Tier: TierFact,
    Citations: []Citation{
        {SourceRef: "民法典第577条", Excerpt: "..."},
    },
}

// INFERENCE — 必须有推理链
finding := &RiskFinding{
    Tier: TierInference,
    Reasoning: "根据合同第8.2条+上下文可推断...",
}

// UNKNOWN — 明确标注不确定
finding := &RiskFinding{
    Tier: TierUnknown,
    Note: "该条款效力存在争议，需人工确认",
}
```

**价值**：法律 AI 最大的问题是幻觉。三层证据强制 AI 标注"这个事实从哪来"，没有出处就标 UNKNOWN。

### 3. 审批门（`internal/guard`）⭐ 安全合规

```go
gate := guard.NewGate()

// 内置法律铁则
gate.AddRule(guard.Rule{
    ID:       "no-fact-without-citation",
    Effect:   guard.EffectDeny,    // 硬拦截！不可豁免！
    Match:    func(intent *Intent) bool {
        return intent.Type == IntentContractReview
    },
    Reason:   "FACT级结论无引用 → 幻觉，禁止输出",
})

gate.AddRule(guard.Rule{
    ID:       "red-risk-mandatory-review",
    Effect:   guard.EffectAsk,     // 需要人工审批
    Match:    func(intent *Intent) bool {
        return finding.Level == RiskRed
    },
    Reason:   "RED级风险必须由执业律师复核",
})
```

**价值**：AI 不会自动把幻觉发给客户。法律行业合规刚需。

### 4. 多 Agent 调度（`internal/engine`）

```
Counsel Agent（主办律师）
    │
    ├──→ Researcher（检索法条）
    ├──→ Analyst（分析条款）
    ├──→ Drafter（起草文书）
    ├──→ Verifier（验证论证）
    └──→ Assessor（评估风险）
         │
         ▼
    Guard 审批门（每个 Agent 产出都要过审）
```

### 5. 完整可观测性

- Prometheus 20+ 指标
- 审计哈希链（区块链式防篡改）
- 每个决策可追溯

---

## 开发者体验（30 分钟上手）

### 快速开始

```bash
# 安装
go get github.com/lexagi/lexagi

# 或者 Docker 一键启动
docker compose up
```

### 30 行代码做一个合同审查 AI

```go
package main

import (
    "context"
    "github.com/lexagi/lexagi"
)

func main() {
    // 1. 创建引擎
    eng := lexagi.NewEngine(lexagi.Config{
        MatterID: "contract-001",
        LLM: lexagi.NewOpenAIProvider(
            os.Getenv("OPENAI_API_KEY"),
            "gpt-4o",
        ),
    })

    // 2. 注入合同
    eng.AddEntity(lexagi.Entity{
        Type: lexagi.EntityContract,
        Name: "技术服务协议",
        Content: contractText,
    })

    // 3. 启动审查流水线
    eng.SubmitIntent(&lexagi.Intent{
        Type: lexagi.IntentContractReview,
        Goal: "全面审查合同法律风险",
    })

    // 4. 等待结果（或 SSE 实时推送）
    findings := eng.WaitForFindings()
    
    // 5. 输出带证据链的风险报告
    for _, f := range findings {
        fmt.Printf("[%s] %s\n", f.Level, f.Title)
        for _, c := range f.Citations {
            fmt.Printf("  📎 %s: %s\n", c.SourceRef, c.Excerpt)
        }
    }
}
```

### 输出示例

```
🔴 [RED] 违约金条款显失公平
   📎 民法典第585条: 约定的违约金低于造成的损失的...
   📎 最高法合同编通则解释第65条: 违约金超过实际损失30%...
   🔗 推理链: 合同第12.1条约定违约金为合同总额50%，
      远超司法解释30%上限， courts may adjust...

🟡 [YELLOW] 知识产权归属约定不明确
   📎 民法典第843条: 委托开发完成的发明创造...
   🔗 推理链: 合同第8条仅约定"归双方共有"，
      未明确使用范围和收益分配，存在争议空间...

🟢 [GREEN] 保密条款基本完善
   📎 参考: 标准NDA模板 v3.2
```

---

## 开源策略

### 许可证：Apache 2.0

- 商业可用（吸引企业用户）
- 专利授权（保护贡献者）
- 修改需标注（保护品牌）

### 商业模式

```
开源层（Apache 2.0）          商业层（Enterprise）
┌─────────────────────┐      ┌─────────────────────┐
│ 核心框架             │      │ 云端 SaaS           │
│ 双图模型             │      │ 私有化部署           │
│ 三层证据             │      │ 企业 SSO            │
│ 基础 Agent          │      │ 审计报告导出         │
│ 本地 LLM 支持        │      │ 行业模板库           │
│                     │      │ SLA 支持             │
│                     │      │ 合规认证（等保/ISO） │
└─────────────────────┘      └─────────────────────┘
       免费                        按需付费
```

---

## 内地市场适配

### 国产 LLM 适配（已实现）

```go
// 支持所有 OpenAI-compatible API
providers := map[string]llm.Provider{
    "kimi":     llm.NewOpenAIProvider("https://api.moonshot.cn/v1", apiKey, "moonshot-v1-8k"),
    "deepseek": llm.NewOpenAIProvider("https://api.deepseek.com/v1", apiKey, "deepseek-chat"),
    "qwen":     llm.NewOpenAIProvider("https://dashscope.aliyuncs.com/v1", apiKey, "qwen-max"),
    "glm":      llm.NewOpenAIProvider("https://open.bigmodel.cn/api/paas/v4", apiKey, "glm-4"),
    "openai":   llm.NewOpenAIProvider("https://api.openai.com/v1", apiKey, "gpt-4o"),
}
```

### 中国法律数据源

```go
// 法条检索插件
engine.AddRetriever("chinalaw", &ChinaLawRetriever{
    // 北大法宝 / 威科先行 / 国家法律法规数据库
})

// 判例检索插件
engine.AddRetriever("court", &CourtCaseRetriever{
    // 中国裁判文书网
})
```

### 合规特性

- **数据不出境**：支持全本地部署（本地 LLM + 本地数据库）
- **等保合规**：审计日志 + 访问控制 + 数据加密
- **律师执业规范**：AI 产出物必须人工审核（Guard 内置）

---

## 路线图

### v0.1 MVP（已完成 ✅）
- [x] 双图模型 + 三层证据
- [x] 审批门（Guard）
- [x] 多 Agent 调度
- [x] OpenAI-compatible Provider
- [x] 6 个法律 Agent
- [x] REST API
- [x] React 前端
- [x] 24 个单元测试

### v0.2 开发者体验（2 周）
- [ ] Go SDK（import 即用）
- [ ] Python SDK（法律圈 Python 用户多）
- [ ] 5 个示例项目
- [ ] 交互式文档（文档即代码）
- [ ] CLI 工具（`lexagi init`）

### v0.3 法律数据源（4 周）
- [ ] 中国法律法规数据库插件
- [ ] 裁判文书网检索插件
- [ ] 合同模板库
- [ ] RAG 管道（法条向量化）

### v0.4 企业版（8 周）
- [ ] 多租户 SaaS
- [ ] 企业 SSO（飞书/钉钉/企业微信）
- [ ] 审计报告导出（PDF/Word）
- [ ] 等保三级认证支持

---

## 竞争壁垒

```
短期壁垒（3-6 个月）:
  ✅ 双图模型 + 三层证据（独创）
  ✅ 审批门 + 审计哈希链
  ✅ 完整的 Go 实现（性能好）

中期壁垒（6-12 个月）:
  🛡️ 开发者生态（Plugin/模板市场）
  🛡️ 法律数据源集成（北大法宝/裁判文书网）
  🛡️ 行业 Know-How（法律提示词库）

长期壁垒（12 个月+）:
  🏰 品牌（"法律 AI 就用 LexAGI"）
  🏰 数据飞轮（用户反馈 → 模型优化）
  🏰 合规认证（律所/法院认可）
```

---

## 总结

| 维度 | 方案 |
|------|------|
| **定位** | 法律界的 LangChain |
| **差异化** | 双图模型 + 三层证据 + 审批门 |
| **目标用户** | 法律科技公司 / 律所 IT / 独立开发者 |
| **商业模式** | 开源核心 + 企业版 SaaS |
| **地域** | 中国优先（国产 LLM + 中国法律数据源） |
| **护城河** | 开发者生态 + 法律数据 + 合规认证 |

队长！这个方案怎么样？选好了我马上调产品方向！⚡🔥
