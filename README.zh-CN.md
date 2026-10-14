<div align="center">

# ⚖️ 律衡 LexScale

**开源法律 AI Agent 框架**

*30 分钟搭建企业级法律 AI，而不是几个月。*

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=white)](https://react.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue?style=flat)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-24%20passing-brightgreen?style=flat)]()

[English](README.md) · [中文](README.zh-CN.md)

</div>

---

## 什么是律衡？

律衡是一个**开源法律 AI Agent 框架**——你可以理解为**"法律界的 LangChain"**。它为法律 AI 应用提供专用基础设施：分层证据体系、审批门、多 Agent 调度、审计链。

### 解决什么问题？

法律 AI 开发面临的核心痛点：

| 痛点 | 后果 |
|------|------|
| AI 在法律建议中幻觉 | 律师失职、法律纠纷 |
| 无法追溯证据来源 | 无法验证 AI 结论 |
| 没有审批流程 | AI 直接把错误建议发给客户 |
| 每个团队重复造轮子 | 浪费工程资源 |

律衡——开箱解决全部问题。

## 30 秒上手

```bash
git clone https://github.com/tongriyaotxt/lexscale.git
cd lexscale

# 构建 CLI
go build -o lexscale ./cmd/lexscale-cli/

# 审查合同
./lexscale -f examples/nda.txt
```

### 输出效果

```
⚖️ 律衡 LexScale — 合同审查报告
═══════════════════════════════════════
📄 文件: examples/nda.txt
🔍 发现: 4 个风险

🔴 [RED] 违约金过高
   📎 民法典第585条
   💡 违约金远超实际损失（通常超过30%），
      法院可能调低...

🟠 [ORANGE] 保密期限永久
   📎 劳动合同法第23条
   💡 保密期限约定为"永久"可能因显失公平
      被法院调整...

═══════════════════════════════════════
统计: 🔴1 🟠2 🟡1 🟢0
```

## 核心创新

| 特性 | 说明 | 为什么重要 |
|------|------|-----------|
| **双图模型** | 法律关系图 + 论证图 | AI 看到的不只是条文，还有关系 |
| **三层证据** | FACT（有引用）/ INFERENCE（有推理）/ UNKNOWN（明确不确定） | 从架构层面消灭幻觉 |
| **审批门** | `deny` 不可豁免 / `ask` 人工审批 / `allow` 放行 | AI 不会把未经审核的结论发给客户 |
| **多 Agent 调度** | 6 个法律角色：主办律师/检索/分析/起草/验证/评估 | 单 Agent 搞不定法律复杂场景 |
| **审计哈希链** | 区块链式防篡改审计日志 | 法律行业合规刚需 |
| **LLM 路由** | 不同 Agent 用不同模型（主办→GPT-4，工人→小模型） | 降本不减质 |

## 架构

```
┌──────────────────────────────────────────────────────────┐
│                     开发者应用层                          │
│   合同审查        │ 法律咨询 Chatbot │ 合规检查           │
├──────────────────────┬───────────────────────────────────┤
│                      ▼                                    │
│   ┌─────────────────────────────────────────────┐        │
│   │            律衡 LexScale Framework           │        │
│   │                                              │        │
│   │  ┌──────────┐  ┌─────────┐  ┌───────────┐  │        │
│   │  │ 双图模型  │  │ 审批门   │  │ 多Agent   │  │        │
│   │  │ Graph    │  │ Guard   │  │ Engine    │  │        │
│   │  └────┬─────┘  └────┬────┘  └─────┬─────┘  │        │
│   │       │              │              │         │        │
│   │  ┌────┴──────────────┴──────────────┴─────┐  │        │
│   │  │          三层证据体系                    │  │        │
│   │  │  FACT → 必须有法条/判例引用             │  │        │
│   │  │  INFERENCE → 必须有推理链               │  │        │
│   │  │  UNKNOWN → 明确标注不确定               │  │        │
│   │  └──────────────────────────────────────────┘  │        │
│   └─────────────────────────────────────────────┘        │
│          │              │              │                   │
├──────────┼──────────────┼──────────────┼───────────────────┤
│          ▼              ▼              ▼                   │
│   ┌──────────┐  ┌──────────┐  ┌─────────────────┐        │
│   │ OpenAI   │  │ Claude   │  │ Kimi/DeepSeek   │        │
│   │ GPT-4    │  │ Sonnet   │  │ 国产模型         │        │
│   └──────────┘  └──────────┘  └─────────────────┘        │
└──────────────────────────────────────────────────────────┘
```

## 多 Agent 流水线

```
Counsel Agent（主办律师）
    │
    ├──→ Researcher  （法条检索，输出引用）
    ├──→ Analyst     （条款分析，风险提取）
    ├──→ Drafter     （文书起草，备选方案）
    ├──→ Verifier    （论证验证）
    └──→ Assessor    （风险评估：严重度×可能性）
              │
              ▼
    ┌─── 审批门 Guard ───┐
    │  每个 Agent 的产出  │
    │  必须经过审批       │
    │  才能交付           │
    └─────────────────────┘
```

## 安装使用

### CLI 工具

```bash
go build -o lexscale ./cmd/lexscale-cli/

# 彩色输出审查
./lexscale -f examples/nda.txt

# JSON 输出（适合管道）
./lexscale -f examples/nda.txt -o json

# 英文输出
./lexscale -f examples/nda.txt -l en
```

### Docker 全栈

```bash
docker compose up -d

# API → http://localhost:8080
# 前端 → http://localhost:3000
curl http://localhost:8080/healthz
```

### Go SDK

```go
import "github.com/tongriyaotxt/lexscale"

eng := lexscale.NewEngine(lexscale.Config{
    LLM: lexscale.NewOpenAIProvider(apiKey, "gpt-4o"),
})
```

## Web 仪表盘

React 前端，功能包括：

- **双图可视化**（React Flow）— 实体、主张、风险节点，风险色标
- **审批工作台** — 待审批队列，批准/拒绝操作
- **案件管理** — 列表/详情页，Tab 切换（图 / 发现 / 实体）
- **风险仪表盘** — Recharts 饼图/柱状图，统计卡片
- **实时 SSE 推送** — Agent 工作时图实时更新

## 示例合同

| 文件 | 类型 | 风险数 |
|------|------|--------|
| `examples/nda.txt` | 保密协议 | 4 |
| `examples/labor_contract.txt` | 劳动合同 | 7 |
| `examples/procurement_contract.txt` | 采购合同 | 3 |

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23+ |
| 前端 | Next.js 14, React 18, Tailwind CSS, Recharts |
| 数据库 | PostgreSQL 15+ |
| 缓存 | Redis 7+ |
| 监控 | Prometheus + 报警规则 |
| 测试 | Go test + Playwright E2E |
| CI/CD | GitHub Actions |

## 文档

| 文档 | 说明 |
|------|------|
| [README.md](README.md) | English Documentation |
| [PROPOSAL.md](PROPOSAL.md) | 完整方案 & 架构深度解析 |
| [ROADMAP.md](ROADMAP.md) | 开发路线图 |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献指南 |
| [docs/API.md](docs/API.md) | REST API 参考 |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | 部署指南 |
| [docs/openapi.yaml](docs/openapi.yaml) | OpenAPI 3.1 规范 |

## 项目统计

| 类型 | 行数 |
|------|------|
| Go | ~2,550 |
| TypeScript/React | ~850 |
| SQL | ~150 |
| Markdown/YAML | ~1,400 |
| **总计** | **~5,000** |

## 贡献

欢迎贡献！请先阅读 [CONTRIBUTING.md](CONTRIBUTING.md)。

```bash
# 运行测试
go test ./...

# 带覆盖率
go test -cover ./...
```

## 许可证

[Apache 2.0](LICENSE) — 商业可用，含专利授权。

---

<div align="center">

**权衡之间，法律智能**

⚖️

</div>
