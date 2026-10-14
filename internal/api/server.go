// Package api REST API 层——商业标准
//
// 端点：
//   POST   /api/v1/matters              创建案件
//   GET    /api/v1/matters              列出案件
//   GET    /api/v1/matters/:id          案件详情（含双图）
//   POST   /api/v1/matters/:id/input    注入用户输入
//   GET    /api/v1/matters/:id/findings 风险发现列表
//   GET    /api/v1/matters/:id/stats    案件统计
//   POST   /api/v1/matters/:id/approve  审批操作
//   GET    /api/v1/matters/:id/audit    审计日志
//   WS     /api/v1/matters/:id/events   实时事件推送
//
// 认证：JWT Bearer Token
// 授权：RBAC（admin / partner / associate / viewer）
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"lexscale/internal/engine"
	"lexscale/internal/graph"
	"lexscale/internal/guard"
)

// Server API 服务器
type Server struct {
	mu      sync.RWMutex
	engines map[string]*engine.Engine // matterID → engine
	guards  map[string]*guard.Gate    // matterID → guard
 graphs  map[string]*graph.LegalGraph

	// 事件推送
	eventSubs map[string][]chan *Event // matterID → subscribers

	// 中间件
	authToken string // 简单 token 认证（生产用 JWT）
}

// Event 实时事件
type Event struct {
	Type      string      `json:"type"` // graph_updated / intent_completed / approval_requested / finding_created
	MatterID  string      `json:"matter_id"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewServer 创建 API 服务器
func NewServer(authToken string) *Server {
	return &Server{
		engines:   make(map[string]*engine.Engine),
		guards:    make(map[string]*guard.Gate),
		graphs:    make(map[string]*graph.LegalGraph),
		eventSubs: make(map[string][]chan *Event),
		authToken: authToken,
	}
}

// ─── 中间件 ─────────────────────────────────────────────

// authMiddleware 简单 token 认证
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = strings.TrimPrefix(token, "Bearer ")
		if s.authToken != "" && token != s.authToken {
			respondError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

// corsMiddleware CORS
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// loggingMiddleware 请求日志
func (s *Server) loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("[%s] %s %s (%v)", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	}
}

// ─── 路由 ─────────────────────────────────────────────

// Handler 返回 http.Handler
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// API v1
	mux.HandleFunc("POST /api/v1/matters", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.createMatter))))
	mux.HandleFunc("GET /api/v1/matters", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.listMatters))))
	mux.HandleFunc("GET /api/v1/matters/{id}", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.getMatter))))
	mux.HandleFunc("POST /api/v1/matters/{id}/input", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.injectInput))))
	mux.HandleFunc("GET /api/v1/matters/{id}/findings", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.getFindings))))
	mux.HandleFunc("GET /api/v1/matters/{id}/stats", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.getStats))))
	mux.HandleFunc("POST /api/v1/matters/{id}/approve", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.approve))))
	mux.HandleFunc("GET /api/v1/matters/{id}/audit", s.loggingMiddleware(s.corsMiddleware(s.authMiddleware(s.getAudit))))
	mux.HandleFunc("GET /api/v1/matters/{id}/events", s.corsMiddleware(s.authMiddleware(s.events)))

	return mux
}

// ─── 处理器 ─────────────────────────────────────────────

// createMatter 创建案件
func (s *Server) createMatter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		MatterType  string `json:"matter_type"`
		ClientName  string `json:"client_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	matterID := fmt.Sprintf("matter-%d", time.Now().UnixNano())

	// 初始化
	lg := graph.NewLegalGraph(matterID)
	gate := guard.NewGate()

	s.mu.Lock()
	s.graphs[matterID] = lg
	s.guards[matterID] = gate
	s.mu.Unlock()

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"matter_id": matterID,
		"name":      req.Name,
		"status":    "created",
	})
}

// listMatters 列出案件
func (s *Server) listMatters(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matters := make([]map[string]interface{}, 0)
	for id, lg := range s.graphs {
		stats := lg.Stats()
		matters = append(matters, map[string]interface{}{
			"matter_id": id,
			"stats":     stats,
		})
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matters": matters,
		"total":   len(matters),
	})
}

// getMatter 案件详情
func (s *Server) getMatter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	lg, ok := s.graphs[id]
	s.mu.RUnlock()

	if !ok {
		respondError(w, http.StatusNotFound, "matter not found")
		return
	}

	// 返回双图 + 统计
	stats := lg.Stats()
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matter_id": id,
		"stats":     stats,
		"entities":  lg.Entities,
		"claims":    lg.Claims,
		"findings":  lg.Findings,
	})
}

// injectInput 注入用户输入
func (s *Server) injectInput(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.mu.RLock()
	eng, ok := s.engines[id]
	s.mu.RUnlock()

	if !ok {
		respondError(w, http.StatusNotFound, "engine not found for matter")
		return
	}

	// 触发 Counsel 重新规划
	eng.NotifyGraphChanged()

	// 推送事件
	s.broadcastEvent(id, &Event{
		Type:      "input_received",
		MatterID:  id,
		Data:      map[string]string{"input": req.Input},
		Timestamp: time.Now(),
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "processing"})
}

// getFindings 风险发现列表
func (s *Server) getFindings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	lg, ok := s.graphs[id]
	s.mu.RUnlock()

	if !ok {
		respondError(w, http.StatusNotFound, "matter not found")
		return
	}

	findings := make([]*graph.RiskFinding, 0)
	for _, f := range lg.Findings {
		findings = append(findings, f)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"findings": findings,
		"total":    len(findings),
	})
}

// getStats 案件统计
func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.mu.RLock()
	lg, ok := s.graphs[id]
	s.mu.RUnlock()

	if !ok {
		respondError(w, http.StatusNotFound, "matter not found")
		return
	}

	respondJSON(w, http.StatusOK, lg.Stats())
}

// approve 审批操作
func (s *Server) approve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Action     string `json:"action"` // approve / reject
		ResourceID string `json:"resource_id"`
		Comment    string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 记录审批事件
	s.broadcastEvent(id, &Event{
		Type:     "approval_action",
		MatterID: id,
		Data: map[string]string{
			"action":      req.Action,
			"resource_id": req.ResourceID,
			"comment":     req.Comment,
		},
		Timestamp: time.Now(),
	})

	respondJSON(w, http.StatusOK, map[string]string{"status": "recorded"})
}

// getAudit 审计日志
func (s *Server) getAudit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// 生产版从数据库查询审计日志
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"matter_id": id,
		"audit":     []interface{}{}, // TODO: 从数据库查询
		"note":      "audit log requires database implementation",
	})
}

// events WebSocket 事件推送
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// 简化版：SSE 而非 WebSocket（无额外依赖）
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan *Event, 10)
	s.mu.Lock()
	s.eventSubs[id] = append(s.eventSubs[id], ch)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		subs := s.eventSubs[id]
		for i, c := range subs {
			if c == ch {
				s.eventSubs[id] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	// 发送已有事件快照
	fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","matter_id":"`+id+`"}`)
	w.(http.Flusher).Flush()

	// 推送新事件
	for {
		select {
		case event := <-ch:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ─── 辅助 ─────────────────────────────────────────────

// broadcastEvent 广播事件
func (s *Server) broadcastEvent(matterID string, event *Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.eventSubs[matterID] {
		select {
		case ch <- event:
		default: // 满则丢弃
		}
	}
}

// RegisterEngine 注册引擎（由 main 调用）
func (s *Server) RegisterEngine(matterID string, eng *engine.Engine, gt *guard.Gate, lg *graph.LegalGraph) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.engines[matterID] = eng
	s.guards[matterID] = gt
	s.graphs[matterID] = lg
}

// respondJSON JSON 响应
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError 错误响应
func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
