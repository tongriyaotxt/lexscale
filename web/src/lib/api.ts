// ─── 律衡 LexScale API Client ─────────────────────────────
// 连接 Go 后端的 REST API + SSE

const BASE = '/api/v1';

export interface Matter {
  matter_id: string;
  name: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface DashboardStats {
  total_matters: number;
  active_matters: number;
  total_findings: number;
  red_findings: number;
  orange_findings: number;
  yellow_findings: number;
  green_findings: number;
  pending_approvals: number;
}

export interface MatterDetail {
  matter_id: string;
  stats: {
    entity_count: number;
    claim_count: number;
    finding_count: number;
    red_count: number;
    orange_count: number;
  };
  entities: Record<string, EntityNode>;
  claims: Record<string, ClaimNode>;
  findings: Record<string, RiskFinding>;
}

export interface EntityNode {
  id: string;
  type: string;
  name: string;
  attributes: Record<string, string>;
}

export interface ClaimNode {
  id: string;
  statement: string;
  status: string;
  evidence_tier: 'FACT' | 'INFERENCE' | 'UNKNOWN';
  citations: Citation[];
}

export interface RiskFinding {
  id: string;
  title: string;
  description: string;
  severity: number;
  likelihood: number;
  score: number;
  level: 'RED' | 'ORANGE' | 'YELLOW' | 'GREEN';
  entity_refs: string[];
  clause_refs: string[];
  recommendation: string;
}

export interface Citation {
  source: string;
  reference: string;
  quote: string;
}

export interface ApprovalItem {
  id: string;
  type: 'intent' | 'finding' | 'artifact';
  title: string;
  description: string;
  requester: string;
  requested_at: string;
  status: 'pending' | 'approved' | 'rejected';
  risk_level?: string;
}

export interface AuditEntry {
  id: string;
  action: string;
  actor: string;
  timestamp: string;
  hash: string;
  prev_hash: string;
  detail: string;
}

// ─── Helpers ──────────────────────────────────────────────

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    ...options,
  });
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res.status, body || res.statusText);
  }
  return res.json();
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

// ─── API Methods ──────────────────────────────────────────

export const api = {
  // Health
  health: () => request<{ status: string }>('/healthz'),

  // Matters
  listMatters: () => request<Matter[]>('/matters'),
  getMatter: (id: string) => request<MatterDetail>(`/matters/${id}`),
  createMatter: (data: { name: string; description?: string }) =>
    request<Matter>('/matters', { method: 'POST', body: JSON.stringify(data) }),
  deleteMatter: (id: string) =>
    request<void>(`/matters/${id}`, { method: 'DELETE' }),

  // Dashboard
  getStats: () => request<DashboardStats>('/stats'),

  // Approvals
  listApprovals: (status?: string) =>
    request<ApprovalItem[]>(`/approvals${status ? `?status=${status}` : ''}`),
  approve: (id: string, comment?: string) =>
    request<void>(`/approvals/${id}/approve`, {
      method: 'POST',
      body: JSON.stringify({ comment }),
    }),
  reject: (id: string, reason: string) =>
    request<void>(`/approvals/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),

  // Audit
  getAuditLog: (matterId?: string) =>
    request<AuditEntry[]>(`/audit${matterId ? `?matter_id=${matterId}` : ''}`),

  // Contract upload & analysis
  uploadContract: async (matterId: string, text: string) => {
    const res = await fetch(`${BASE}/matters/${matterId}/input`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ type: 'contract', content: text }),
    });
    if (!res.ok) throw new ApiError(res.status, await res.text());
    return res.json();
  },
};

// ─── SSE ─────────────────────────────────────────────────

export interface SSEEvent {
  type: 'graph_updated' | 'intent_completed' | 'approval_requested' | 'finding_created' | 'connected';
  matter_id?: string;
  data: Record<string, unknown>;
  timestamp: string;
}

export function connectSSE(
  onEvent: (event: SSEEvent) => void,
  onError?: (err: Event) => void
): () => void {
  const source = new EventSource(`${BASE}/events`);

  const handlers: Record<string, (e: MessageEvent) => void> = {
    graph_updated: (e) => onEvent({ type: 'graph_updated', data: JSON.parse(e.data), timestamp: new Date().toISOString() }),
    intent_completed: (e) => onEvent({ type: 'intent_completed', data: JSON.parse(e.data), timestamp: new Date().toISOString() }),
    approval_requested: (e) => onEvent({ type: 'approval_requested', data: JSON.parse(e.data), timestamp: new Date().toISOString() }),
    finding_created: (e) => onEvent({ type: 'finding_created', data: JSON.parse(e.data), timestamp: new Date().toISOString() }),
    connected: (e) => onEvent({ type: 'connected', data: JSON.parse(e.data), timestamp: new Date().toISOString() }),
  };

  Object.entries(handlers).forEach(([event, handler]) => {
    source.addEventListener(event, handler as EventListener);
  });

  if (onError) source.onerror = onError;

  return () => source.close();
}
