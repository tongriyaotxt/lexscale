# LexAGI API 文档
# REST API 完整参考

## 认证

所有 API 请求需要 Bearer Token：

```bash
curl -H "Authorization: Bearer $LEXAGI_AUTH_TOKEN" \
  http://localhost:8080/api/v1/matters
```

## 端点

### 健康检查

```
GET /healthz
```

响应：
```json
{"status": "ok"}
```

---

### 创建案件

```
POST /api/v1/matters
```

请求体：
```json
{
  "name": "合同审查-ABC公司",
  "description": "服务协议法律风险审查",
  "matter_type": "contract_review",
  "client_name": "ABC科技有限公司",
  "opposing_party": "XYZ服务有限公司"
}
```

响应：
```json
{
  "matter_id": "matter-1697193600000000000",
  "name": "合同审查-ABC公司",
  "status": "created"
}
```

---

### 列出案件

```
GET /api/v1/matters
```

响应：
```json
{
  "matters": [
    {
      "matter_id": "matter-xxx",
      "stats": {
        "entity_count": 5,
        "claim_count": 3,
        "finding_count": 2,
        "red_count": 1,
        "orange_count": 0
      }
    }
  ],
  "total": 1
}
```

---

### 案件详情

```
GET /api/v1/matters/:id
```

响应：
```json
{
  "matter_id": "matter-xxx",
  "stats": {...},
  "entities": {
    "entity-1": {
      "id": "entity-1",
      "type": "PARTY",
      "name": "甲方",
      "attributes": {...}
    }
  },
  "claims": {...},
  "findings": {...}
}
```

---

### 注入输入

```
POST /api/v1/matters/:id/input
```

请求体：
```json
{
  "input": "请审查以下合同条款：付款期限为30天，违约金为日万分之五..."
}
```

响应：
```json
{
  "status": "processing"
}
```

触发 Counsel 重新规划，异步处理。

---

### 风险发现列表

```
GET /api/v1/matters/:id/findings
```

响应：
```json
{
  "findings": [
    {
      "id": "finding-xxx",
      "title": "违约金条款风险",
      "description": "日万分之五违约金过高...",
      "severity": 4,
      "likelihood": 3,
      "score": 12,
      "level": "ORANGE",
      "entity_refs": ["entity-clause-8"],
      "tier": "INFERENCE",
      "citations": [...]
    }
  ],
  "total": 1
}
```

---

### 案件统计

```
GET /api/v1/matters/:id/stats
```

响应：
```json
{
  "entity_count": 5,
  "claim_count": 3,
  "finding_count": 2,
  "red_count": 1,
  "orange_count": 1
}
```

---

### 审批操作

```
POST /api/v1/matters/:id/approve
```

请求体：
```json
{
  "action": "approve",
  "resource_id": "intent-xxx",
  "comment": "同意执行该分析意图"
}
```

响应：
```json
{
  "status": "recorded"
}
```

---

### 审计日志

```
GET /api/v1/matters/:id/audit
```

响应：
```json
{
  "matter_id": "matter-xxx",
  "audit": [...],
  "note": "audit log requires database implementation"
}
```

---

### 实时事件（SSE）

```
GET /api/v1/matters/:id/events
```

Server-Sent Events 流，推送以下事件：

```
data: {"type":"connected","matter_id":"matter-xxx"}

data: {"type":"graph_updated","matter_id":"matter-xxx","data":{...},"timestamp":"..."}

data: {"type":"intent_completed","matter_id":"matter-xxx","data":{...},"timestamp":"..."}

data: {"type":"approval_requested","matter_id":"matter-xxx","data":{...},"timestamp":"..."}
```

---

## 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未认证 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

错误响应格式：
```json
{
  "error": "matter not found"
}
```

---

## 数据模型

### EvidenceTier
- `FACT`：有明确出处
- `INFERENCE`：模型推理产物
- `UNKNOWN`：明确未知

### RiskLevel
- `GREEN`：Score 1-4
- `YELLOW`：Score 5-9
- `ORANGE`：Score 10-15
- `RED`：Score 16-25

### EntityType
- `PARTY`：当事人
- `CONTRACT`：合同
- `CLAUSE`：条款
- `OBLIGATION`：义务
- `RIGHT`：权利
- `EVIDENCE`：证据
- `JURISDICTION`：法域
- `STATUTE`：法条

### ClaimStatus
- `OPEN`：待论证
- `SUPPORTED`：已获支持
- `WEAKENED`：存在削弱
- `REFUTED`：被推翻
- `UNDETERMINED`：无法确定
