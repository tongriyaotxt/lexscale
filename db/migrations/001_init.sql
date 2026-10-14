-- LexAGI Database Schema
-- PostgreSQL 14+
-- 案件隔离、审计不可篡改、图数据 JSONB 存储

-- 案件表
CREATE TABLE IF NOT EXISTS matters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    matter_type VARCHAR(50) DEFAULT 'general', -- contract_review / litigation / compliance / etc.
    status VARCHAR(20) DEFAULT 'active', -- active / paused / closed / archived
    client_name VARCHAR(255),
    opposing_party VARCHAR(255),
    jurisdiction VARCHAR(100),
    created_by VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 法律实体表（图1节点）
CREATE TABLE IF NOT EXISTS entities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    node_type VARCHAR(50) NOT NULL, -- PARTY / CONTRACT / CLAUSE / OBLIGATION / RIGHT / EVIDENCE / JURISDICTION / STATUTE
    name VARCHAR(500) NOT NULL,
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(matter_id, node_type, name)
);

-- 实体关系表（图1边）
CREATE TABLE IF NOT EXISTS entity_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    from_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    to_entity_id UUID NOT NULL REFERENCES entities(id) ON DELETE CASCADE,
    relation VARCHAR(100) NOT NULL, -- 签订/包含/设定/证明/适用/约束
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 论证主张表（图2节点）
CREATE TABLE IF NOT EXISTS claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    statement TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'OPEN', -- OPEN / SUPPORTED / WEAKENED / REFUTED / UNDETERMINED
    support_edges JSONB DEFAULT '[]', -- SupportEdge 数组
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 风险发现表（锚点）
CREATE TABLE IF NOT EXISTS findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    severity INT NOT NULL CHECK (severity BETWEEN 1 AND 5),
    likelihood INT NOT NULL CHECK (likelihood BETWEEN 1 AND 5),
    score INT GENERATED ALWAYS AS (severity * likelihood) STORED,
    level VARCHAR(10) GENERATED ALWAYS AS (
        CASE
            WHEN severity * likelihood >= 16 THEN 'RED'
            WHEN severity * likelihood >= 10 THEN 'ORANGE'
            WHEN severity * likelihood >= 5 THEN 'YELLOW'
            ELSE 'GREEN'
        END
    ) STORED,
    evidence_tier VARCHAR(20) DEFAULT 'INFERENCE', -- FACT / INFERENCE / UNKNOWN
    citations JSONB DEFAULT '[]',
    entity_refs JSONB DEFAULT '[]', -- UUID array
    claim_ref UUID REFERENCES claims(id),
    verified BOOLEAN DEFAULT FALSE,
    verified_by VARCHAR(100),
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 意图执行表（审计日志）
CREATE TABLE IF NOT EXISTS intents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    intent_type VARCHAR(50) NOT NULL,
    goal TEXT,
    context JSONB DEFAULT '{}',
    priority INT DEFAULT 5,
    status VARCHAR(20) DEFAULT 'PENDING', -- PENDING / APPROVED / REJECTED / RUNNING / COMPLETED / FAILED
    worker_id VARCHAR(100),
    result JSONB,
    error TEXT,
    duration_ms INT,
    guard_decision VARCHAR(10), -- deny / ask / allow
    guard_rule_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- 审计日志（不可篡改，法律合规刚需）
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL, -- intent_created / intent_completed / guard_approved / finding_verified / etc.
    actor VARCHAR(100) NOT NULL, -- agent name / user id
    actor_type VARCHAR(20) NOT NULL, -- agent / human / system
    resource_type VARCHAR(50), -- intent / finding / claim / entity
    resource_id UUID,
    detail JSONB DEFAULT '{}',
    prev_hash VARCHAR(64), -- 前一记录哈希（区块链式防篡改）
    record_hash VARCHAR(64) NOT NULL, -- 当前记录哈希
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 产出物表
CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100),
    content BYTEA,
    content_text TEXT,
    created_by VARCHAR(100), -- agent name
    is_draft BOOLEAN DEFAULT TRUE, -- 是否草稿（未经审核）
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 审批记录表
CREATE TABLE IF NOT EXISTS approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matter_id UUID NOT NULL REFERENCES matters(id) ON DELETE CASCADE,
    intent_id UUID REFERENCES intents(id),
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    decision VARCHAR(10) NOT NULL, -- approved / rejected
    decided_by VARCHAR(100) NOT NULL,
    comment TEXT,
    auto_allow BOOLEAN DEFAULT FALSE,
    auto_allow_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 索引
CREATE INDEX IF NOT EXISTS idx_entities_matter ON entities(matter_id);
CREATE INDEX IF NOT EXISTS idx_findings_matter ON findings(matter_id);
CREATE INDEX IF NOT EXISTS idx_findings_level ON findings(level);
CREATE INDEX IF NOT EXISTS idx_intents_matter ON intents(matter_id);
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_audit_matter ON audit_log(matter_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_artifacts_matter ON artifacts(matter_id);
