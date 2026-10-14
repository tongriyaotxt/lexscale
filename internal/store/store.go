// Package store 图数据持久化层
//
// 负责 LegalGraph 与 PostgreSQL 之间的映射
// 支持：实体/边/主张/风险发现的 CRUD + 审计日志
//
// 设计要点：
//   - 事务保证一致性
//   - JSONB 存储扩展属性
//   - 审计日志哈希链（区块链式防篡改）
package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"lexscale/internal/graph"
)

// Store 图数据存储接口
type Store interface {
	// 案件
	CreateMatter(ctx context.Context, m *Matter) error
	GetMatter(ctx context.Context, id string) (*Matter, error)
	UpdateMatter(ctx context.Context, m *Matter) error
	DeleteMatter(ctx context.Context, id string) error

	// 实体
	AddEntity(ctx context.Context, matterID string, e *graph.EntityNode) error
	GetEntities(ctx context.Context, matterID string) ([]*graph.EntityNode, error)
	DeleteEntity(ctx context.Context, id string) error

	// 边
	AddEdge(ctx context.Context, matterID string, e *graph.EntityEdge) error
	GetEdges(ctx context.Context, matterID string) ([]*graph.EntityEdge, error)

	// 主张
	AddClaim(ctx context.Context, matterID string, c *graph.ClaimNode) error
	GetClaims(ctx context.Context, matterID string) ([]*graph.ClaimNode, error)
	UpdateClaim(ctx context.Context, c *graph.ClaimNode) error

	// 风险发现
	AddFinding(ctx context.Context, matterID string, f *graph.RiskFinding) error
	GetFindings(ctx context.Context, matterID string) ([]*graph.RiskFinding, error)
	UpdateFinding(ctx context.Context, f *graph.RiskFinding) error

	// 审计日志
	AppendAudit(ctx context.Context, a *AuditEntry) error
	GetAuditLog(ctx context.Context, matterID string, limit int) ([]*AuditEntry, error)

	// 图加载/保存
	LoadGraph(ctx context.Context, matterID string) (*graph.LegalGraph, error)
	SaveGraph(ctx context.Context, g *graph.LegalGraph) error
}

// Matter 案件元数据
type Matter struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MatterType  string    `json:"matter_type"`
	Status      string    `json:"status"`
	ClientName  string    `json:"client_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditEntry 审计日志条目
type AuditEntry struct {
	ID           string          `json:"id"`
	MatterID     string          `json:"matter_id"`
	Action       string          `json:"action"`
	Actor        string          `json:"actor"`
	ActorType    string          `json:"actor_type"` // agent / human / system
	ResourceType string          `json:"resource_type"`
	ResourceID   string          `json:"resource_id"`
	Detail       json.RawMessage `json:"detail"`
	PrevHash     string          `json:"prev_hash"`
	RecordHash   string          `json:"record_hash"`
	CreatedAt    time.Time       `json:"created_at"`
}

// ComputeHash 计算审计记录哈希
func (a *AuditEntry) ComputeHash() string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		a.MatterID, a.Action, a.Actor, a.ActorType,
		a.ResourceType, a.ResourceID, string(a.Detail), a.PrevHash)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// ChainVerify 验证审计链完整性
func ChainVerify(entries []*AuditEntry) bool {
	for i, entry := range entries {
		expected := entry.ComputeHash()
		if expected != entry.RecordHash {
			return false
		}
		if i > 0 && entry.PrevHash != entries[i-1].RecordHash {
			return false
		}
	}
	return true
}

// PostgresStore PostgreSQL 实现（骨架）
type PostgresStore struct {
	// db *sql.DB  // 生产版接入真实 DB
	dsn string
}

// NewPostgresStore 创建存储
func NewPostgresStore(dsn string) *PostgresStore {
	return &PostgresStore{dsn: dsn}
}

// LoadGraph 从数据库加载完整双图
func (s *PostgresStore) LoadGraph(ctx context.Context, matterID string) (*graph.LegalGraph, error) {
	// 生产版实现：
	// 1. 查询 entities 表
	// 2. 查询 entity_edges 表
	// 3. 查询 claims 表
	// 4. 查询 findings 表
	// 5. 组装为 LegalGraph
	
	g := graph.NewLegalGraph(matterID)
	// 骨架版返回空图，生产版从 DB 加载
	return g, nil
}

// SaveGraph 保存完整双图到数据库
func (s *PostgresStore) SaveGraph(ctx context.Context, g *graph.LegalGraph) error {
	// 生产版实现：
	// 1. 开启事务
	// 2. UPSERT entities
	// 3. UPSERT edges
	// 4. UPSERT claims
	// 5. UPSERT findings
	// 6. 提交事务
	return nil
}

// AppendAudit 追加审计日志（自动计算哈希链）
func (s *PostgresStore) AppendAudit(ctx context.Context, a *AuditEntry) error {
	// 查询上一条记录的哈希
	var prevHash string
	// SELECT record_hash FROM audit_log WHERE matter_id = $1 ORDER BY created_at DESC LIMIT 1
	// 骨架版简化处理
	
	a.PrevHash = prevHash
	a.RecordHash = a.ComputeHash()
	
	// INSERT INTO audit_log (...)
	// 骨架版不实际写入
	return nil
}
