# LexAGI Deployment Guide
# 部署指南

## 快速部署（Docker Compose）

### 前置要求
- Docker 20.10+
- Docker Compose 2.0+
- 至少 4GB RAM

### 1. 克隆代码
```bash
git clone https://github.com/your-org/lexagi.git
cd lexagi
```

### 2. 配置环境
```bash
cp .env.example .env
# 编辑 .env，填入：
# - LEXAGI_LLM_API_KEY（必需）
# - LEXAGI_DB_PASSWORD（建议修改默认值）
# - LEXAGI_AUTH_TOKEN（建议修改默认值）
```

### 3. 启动服务
```bash
docker-compose up -d
```

### 4. 验证部署
```bash
# API 健康检查
curl http://localhost:8080/healthz

# 前端
open http://localhost:3000
```

### 5. 查看日志
```bash
docker-compose logs -f api
docker-compose logs -f frontend
```

---

## 生产部署

### 架构图
```
┌─────────────────────────────────────────┐
│           Nginx (SSL/TLS)               │
│         反向代理 + 负载均衡              │
└──────────────┬──────────────────────────┘
               │
    ┌──────────┼──────────┐
    │          │          │
┌───▼───┐ ┌───▼───┐ ┌───▼────┐
│ Web   │ │ API   │ │ API    │
│ (x2)  │ │ (x3)  │ │ (x3)   │
└───┬───┘ └───┬───┘ └───┬────┘
    │          │          │
    └──────────┼──────────┘
               │
    ┌──────────┼──────────┐
    │          │          │
┌───▼───┐ ┌───▼───┐ ┌───▼────┐
│Postgre│ │ Redis │ │Prometh.│
│  SQL  │ │       │ │  eus   │
└───────┘ └───────┘ └────────┘
```

### Nginx 配置示例
```nginx
upstream lexagi_api {
    least_conn;
    server api1:8080;
    server api2:8080;
    server api3:8080;
}

upstream lexagi_web {
    least_conn;
    server web1:3000;
    server web2:3000;
}

server {
    listen 80;
    server_name lexagi.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name lexagi.example.com;

    ssl_certificate /etc/letsencrypt/live/lexagi.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/lexagi.example.com/privkey.pem;

    # API
    location /api/ {
        proxy_pass http://lexagi_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE
    location /api/v1/matters/*/events {
        proxy_pass http://lexagi_api;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        chunked_transfer_encoding off;
    }

    # Web
    location / {
        proxy_pass http://lexagi_web;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

## 备份与恢复

### 数据库备份
```bash
# 备份
docker-compose exec postgres pg_dump -U lexagi lexagi > backup_$(date +%Y%m%d).sql

# 恢复
docker-compose exec -T postgres psql -U lexagi lexagi < backup_20261013.sql
```

### 完整备份（含数据卷）
```bash
docker run --rm -v lexagi_postgres-data:/data -v $(pwd):/backup alpine tar czf /backup/postgres-backup.tar.gz -C /data .
```

---

## 监控

### Prometheus + Grafana
```bash
docker-compose -f docker-compose.monitoring.yml up -d
```

访问 `http://localhost:3001`（Grafana 默认账号 admin/admin）

### 关键指标
| 指标 | 说明 | 告警阈值 |
|------|------|----------|
| `lexagi_http_requests_total` | HTTP 请求数 | - |
| `lexagi_http_request_duration_seconds` | 请求延迟 | P95 > 2s |
| `lexagi_engine_queue_depth` | 意图队列深度 | > 50 |
| `lexagi_findings_total{level="RED"}` | RED 级风险发现 | 速率异常 |
| `lexagi_llm_provider_up` | LLM Provider 状态 | == 0 |
| `lexagi_guard_denies_total` | Guard 拦截次数 | - |
| `lexagi_pending_approvals` | 待审批数 | > 20 |

---

## 故障排查

### API 无法启动
```bash
# 检查日志
docker-compose logs api

# 常见原因
# 1. 数据库未就绪 → 等 postgres healthy
# 2. LLM API Key 未配置 → 检查 .env
# 3. 端口占用 → 修改 docker-compose.yml 端口映射
```

### LLM 调用失败
```bash
# 检查 Provider 状态
curl http://localhost:8080/metrics | grep lexagi_llm_provider_up

# 测试 LLM 连接
curl -X POST http://localhost:8080/api/v1/matters/test-matter/input \
  -H "Authorization: Bearer $LEXAGI_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"input": "test"}'
```

### 前端无法访问
```bash
# 检查前端日志
docker-compose logs frontend

# 检查 API 代理
curl http://localhost:3000/api/v1/matters
```

---

## 安全建议

1. **修改默认密码**：`LEXAGI_DB_PASSWORD`、`LEXAGI_AUTH_TOKEN`
2. **启用 SSL**：使用 Let's Encrypt 或商业证书
3. **限制访问**：防火墙只允许 80/443
4. **定期更新**：`docker-compose pull && docker-compose up -d`
5. **审计日志**：定期导出 `audit_log` 表
6. **密钥管理**：生产环境使用 Vault/AWS Secrets Manager
