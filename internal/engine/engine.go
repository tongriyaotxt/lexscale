// Package engine 事件驱动调度引擎。
//
// 核心循环（仿 ARTEX）：
//
//	图变更(debounced) → 唤醒 Counsel（唯一策略生成者）→
//	Counsel 生成 Intent（分析意图）→ Workers 并发领取执行 →
//	结果写回双图 → 写回触发图变更 → 循环
//
// 关键约束（法律版）：
//   - Counsel 是唯一意图生成者——保证论证链不重复、不错序
//   - 每个 Intent 执行前过 Guard 审批门——deny 硬拦截 / ask 人工审批 / allow 放行
//   - Worker 执行完即停，不自行扩散（防权限蔓延）
//   - pause 是真取消（cancel context），不只是跳过下一轮
package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"lexscale/internal/graph"
	"lexscale/internal/guard"
)

// IntentType 分析意图类型
type IntentType string

const (
	IntentContractReview  IntentType = "CONTRACT_REVIEW"  // 合同条款审查
	IntentStatuteResearch IntentType = "STATUTE_RESEARCH" // 法条检索
	IntentPrecedentSearch IntentType = "PRECEDENT_SEARCH" // 判例检索
	IntentComplianceCheck IntentType = "COMPLIANCE_CHECK" // 合规核查
	IntentDraftDocument   IntentType = "DRAFT_DOCUMENT"   // 文书起草
	IntentVerifyFinding   IntentType = "VERIFY_FINDING"   // 风险点反证验证
	IntentRiskAssess      IntentType = "RISK_ASSESS"      // 风险评级
)

// Intent 分析意图——Counsel 生成的最小工作单元
type Intent struct {
	ID        string                 `json:"id"`
	Type      IntentType             `json:"type"`
	Goal      string                 `json:"goal"`       // 意图目标描述
	Context   map[string]string      `json:"context"`    // 上下文（条款原文/争议焦点等）
	Priority  int                    `json:"priority"`   // 优先级（1-10，大者先执行）
	CreatedAt time.Time              `json:"created_at"`

	// 执行状态
	Status    IntentStatus `json:"status"`
	WorkerID  string       `json:"worker_id,omitempty"`
	Result    *IntentResult `json:"result,omitempty"`
}

// IntentStatus 意图状态
type IntentStatus string

const (
	IntentPending   IntentStatus = "PENDING"
	IntentApproved  IntentStatus = "APPROVED"  // 已过审批门
	IntentRejected  IntentStatus = "REJECTED"  // 被审批门拦截
	IntentRunning   IntentStatus = "RUNNING"
	IntentCompleted IntentStatus = "COMPLETED"
	IntentFailed    IntentStatus = "FAILED"
)

// IntentResult 意图执行结果
type IntentResult struct {
	Summary   string                `json:"summary"`
	Findings  []*graph.RiskFinding  `json:"findings,omitempty"`
	Entities  []*graph.EntityNode   `json:"entities,omitempty"`
	Claims    []*graph.ClaimNode    `json:"claims,omitempty"`
	Documents []string              `json:"documents,omitempty"` // 生成的文书
	Error     string                `json:"error,omitempty"`
	Duration  time.Duration         `json:"duration"`
}

// Worker 分析 Worker 接口——每领一条意图执行完即停
type Worker interface {
	ID() string
	CanHandle(intentType IntentType) bool
	Execute(ctx context.Context, intent *Intent, g *graph.LegalGraph) (*IntentResult, error)
}

// Counsel 主办律师接口——唯一意图生成者（ARTEX Planner 职能）
type Counsel interface {
	// Plan 读取当前图态势，生成下一轮意图列表
	Plan(ctx context.Context, g *graph.LegalGraph, history []*Intent) ([]*Intent, error)
}

// Engine 事件驱动引擎
type Engine struct {
	matterID string
	counsel  Counsel
	workers  []Worker
	guard    *guard.Gate
	graph    *graph.LegalGraph

	// 并发控制
	intentQueue chan *Intent
	history     []*Intent
	historyMu   sync.Mutex

	// 生命周期
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 图变更去抖（debounce，仿 ARTEX 的 debounced wake）
	graphChanged chan struct{}
	debounce     time.Duration

	// 状态回调
	onIntentComplete func(intent *Intent)
	onGraphUpdate    func(g *graph.LegalGraph)
}

// Config 引擎配置
type Config struct {
	MatterID         string
	Counsel          Counsel
	Workers          []Worker
	Guard            *guard.Gate
	Debounce         time.Duration // 图变更去抖窗口，默认 2s
	MaxConcurrent    int           // 最大并发 Worker 数，默认 3
	OnIntentComplete func(intent *Intent)
	OnGraphUpdate    func(g *graph.LegalGraph)
}

// New 创建引擎
func New(cfg Config) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	if cfg.Debounce <= 0 {
		cfg.Debounce = 2 * time.Second
	}
	e := &Engine{
		matterID:         cfg.MatterID,
		counsel:          cfg.Counsel,
		workers:          cfg.Workers,
		guard:            cfg.Guard,
		graph:            graph.NewLegalGraph(cfg.MatterID),
		ctx:              ctx,
		cancel:           cancel,
		graphChanged:     make(chan struct{}, 1),
		debounce:         cfg.Debounce,
		onIntentComplete: cfg.OnIntentComplete,
		onGraphUpdate:    cfg.OnGraphUpdate,
	}
	e.intentQueue = make(chan *Intent, 100)
	return e
}

// Graph 返回当前案件的双图
func (e *Engine) Graph() *graph.LegalGraph {
	return e.graph
}

// Start 启动引擎主循环
func (e *Engine) Start() error {
	e.wg.Add(2)
	go e.counselLoop()
	go e.dispatchLoop()
	return nil
}

// Stop 停止引擎——真取消，等待所有 goroutine 退出
func (e *Engine) Stop() error {
	e.cancel()
	done := make(chan struct{})
	go func() { e.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("engine stop timeout: workers did not exit in 30s")
	}
}

// NotifyGraphChanged 通知图变更（触发 Counsel 重新规划）
func (e *Engine) NotifyGraphChanged() {
	select {
	case e.graphChanged <- struct{}{}:
	default: // 已有待处理通知，跳过（去抖）
	}
}

// counselLoop 主办律师循环：图变更 → 去抖 → 重新规划
func (e *Engine) counselLoop() {
	defer e.wg.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-e.graphChanged:
			// 去抖：等待更多变更聚合
			timer := time.NewTimer(e.debounce)
			select {
			case <-e.ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			e.planAndEnqueue()
		}
	}
}

// planAndEnqueue 调用 Counsel 规划并入队
func (e *Engine) planAndEnqueue() {
	intents, err := e.counsel.Plan(e.ctx, e.graph, e.getHistory())
	if err != nil {
		// Counsel 规划失败记录但不终止——下轮图变更会重试
		return
	}
	for _, intent := range intents {
		// 审批门：每个意图入队前检查
		decision := e.guard.CheckIntent(intent)
		switch decision {
		case guard.DecisionDeny:
			intent.Status = IntentRejected
			e.recordIntent(intent)
			continue
		case guard.DecisionAsk:
			intent.Status = IntentPending // 等待人工审批
			// 入审批队列，由 guard 异步处理
			e.guard.EnqueueForApproval(intent)
			continue
		case guard.DecisionAllow:
			intent.Status = IntentApproved
		}

		select {
		case e.intentQueue <- intent:
		case <-e.ctx.Done():
			return
		}
	}
}

// dispatchLoop Worker 分发循环
func (e *Engine) dispatchLoop() {
	defer e.wg.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case intent := <-e.intentQueue:
			worker := e.findWorker(intent.Type)
			if worker == nil {
				intent.Status = IntentFailed
				intent.Result = &IntentResult{Error: "no available worker for " + string(intent.Type)}
				e.recordIntent(intent)
				continue
			}
			e.wg.Add(1)
			go func(w Worker, it *Intent) {
				defer e.wg.Done()
				execCtx, execCancel := context.WithCancel(e.ctx)
				it.Status = IntentRunning
				it.WorkerID = w.ID()

				start := time.Now()
				result, err := w.Execute(execCtx, it, e.graph)
				execCancel()

				it.Result = result
				if err != nil {
					it.Status = IntentFailed
					if it.Result == nil {
						it.Result = &IntentResult{}
					}
					it.Result.Error = err.Error()
				} else {
					it.Status = IntentCompleted
				}
				it.Result.Duration = time.Since(start)

				e.recordIntent(it)
			}(worker, intent)
		}
	}
}

// findWorker 按意图类型找可用 Worker
func (e *Engine) findWorker(intentType IntentType) Worker {
	for _, w := range e.workers {
		if w.CanHandle(intentType) {
			return w
		}
	}
	return nil
}

// recordIntent 记录完成的意图并触发图变更
func (e *Engine) recordIntent(intent *Intent) {
	e.historyMu.Lock()
	e.history = append(e.history, intent)
	e.historyMu.Unlock()

	// 执行结果写回双图
	if intent.Result != nil {
		for _, f := range intent.Result.Findings {
			e.graph.AddFinding(f)
		}
		for _, en := range intent.Result.Entities {
			e.graph.Entities[en.ID] = en
		}
		for _, c := range intent.Result.Claims {
			e.graph.Claims[c.ID] = c
		}
	}

	if e.onIntentComplete != nil {
		e.onIntentComplete(intent)
	}
	if e.onGraphUpdate != nil {
		e.onGraphUpdate(e.graph)
	}

	// 完成的意图可能改变态势 → 触发 Counsel 重新规划
	e.NotifyGraphChanged()
}

// getHistory 返回执行历史快照
func (e *Engine) getHistory() []*Intent {
	e.historyMu.Lock()
	defer e.historyMu.Unlock()
	out := make([]*Intent, len(e.history))
	copy(out, e.history)
	return out
}

// History 返回执行历史（外部只读）
func (e *Engine) History() []*Intent {
	return e.getHistory()
}

// WaitIdle 等待意图队列清空（测试/同步用）
func (e *Engine) WaitIdle(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(e.intentQueue) == 0 {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
