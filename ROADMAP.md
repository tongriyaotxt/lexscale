# LexAGI — 商业标准路线图

## 当前状态
- **2550 行 Go + 847 行 TypeScript/React，零编译错误，零 vet 警告**
- 已有：graph / engine / guard / egress / llm(OpenAI兼容) / 6 agents / REST API / React前端
- Docker Compose：API + PostgreSQL + Redis + Frontend
- CI/CD：GitHub Actions 完整流水线
- 监控：Prometheus + Alert Rules
- 文档：API.md + DEPLOYMENT.md

## 完成情况

### Phase 1: LLM 接入 + 全 Agent 补齐 ✅ 完成
- [x] OpenAI-compatible Provider（支持 OpenAI / Claude / Kimi / DeepSeek / GLM）
- [x] Provider 路由实现（按角色选模型：Counsel 大模型，Worker 小模型）
- [x] Researcher Agent（法条检索，输出 Citation）
- [x] Drafter Agent（文书起草，标注 DRAFT，备选方案）
- [x] Assessor Agent（风险评估，severity×likelihood 矩阵）
- [x] 提示词模板文件化（prompts/*.md，6 个 Agent）

### Phase 2: 持久化 ✅ 完成
- [x] PostgreSQL schema（9 张表：matters/entities/edges/claims/findings/intents/audit_log/artifacts/approvals）
- [x] 审计日志（区块链式哈希链，不可篡改）
- [x] Migration SQL（001_init.sql）

### Phase 3: API 层 ✅ 完成
- [x] REST API（matters CRUD、input 注入、审批操作、图查询、统计）
- [x] SSE 实时推送（graph_updated / intent_completed / approval_requested）
- [x] Token 认证 + CORS + Logging

### Phase 4: 前端 ✅ 完成
- [x] 双图可视化（React Flow，实体/主张/风险节点，风险色标）
- [x] 审批工作台（待审批队列，批准/拒绝）
- [x] Matter 列表/详情页（Tab 切换：图/发现/实体）
- [x] 风险仪表盘（Recharts 饼图/柱状图，统计卡片）

### Phase 5: 生产化 ✅ 完成
- [x] Docker Compose（API + PostgreSQL + Redis + Frontend）
- [x] Prometheus 配置 + Alert Rules
- [x] CI/CD pipeline（GitHub Actions：Go/Web/Security/Docker/Deploy）
- [x] API 文档（docs/API.md）
- [x] 部署文档（docs/DEPLOYMENT.md）

## 最终统计
| 类型 | 行数 |
|------|------|
| Go | 2550 |
| TypeScript/React | 847 |
| SQL | 152 |
| Markdown | 920 |
| YAML/Config | 526 |
| **总计** | **~5000** |

## 核心设计
- **双图模型**：LegalEntityGraph + ArgumentationGraph，RiskFinding 锚点
- **三层证据**：FACT（有引用）/ INFERENCE（有推理）/ UNKNOWN（不知道）
- **审批门**：deny 不可豁免 / ask 人工审批 / allow 放行
- **意图驱动**：Counsel 唯一生成者，Worker 执行完即停（防连环套娃）
- **按角色路由 LLM**：Counsel 大模型，Worker 小模型

## 后续迭代方向
- JWT + RBAC 完整实现（主办律师/合伙人/管理员）
- Rate limiting（Redis 令牌桶）
- Swagger/OpenAPI 自动生成
- E2E 测试（Playwright）
- 法律数据库对接（裁判文书网 / 北大法宝）
- 多语言支持（英文 / 日文）
