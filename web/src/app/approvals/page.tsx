'use client';

import { useState, useEffect, useCallback } from 'react';
import { CheckCircle, XCircle, Clock } from 'lucide-react';
import { api, ApprovalItem } from '@/lib/api';
import { useSSE } from '@/hooks/useSSE';
import { format } from 'date-fns';

export default function ApprovalsPage() {
  const [approvals, setApprovals] = useState<ApprovalItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'pending' | 'approved' | 'rejected'>('pending');
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchApprovals = useCallback(async () => {
    try {
      setError(null);
      const data = await api.listApprovals(filter === 'all' ? undefined : filter);
      setApprovals(data);
    } catch {
      // 演示数据
      setApprovals([
        {
          id: '1', type: 'intent', title: '合同条款深度审查',
          description: '对付款条款进行深度法律分析，涉及违约金计算',
          requester: 'counsel-agent', requested_at: new Date().toISOString(),
          status: 'pending', risk_level: 'ORANGE',
        },
        {
          id: '2', type: 'finding', title: '知识产权归属风险',
          description: '发现乙方保留全部知识产权条款存在重大风险',
          requester: 'analyst-agent', requested_at: new Date().toISOString(),
          status: 'pending', risk_level: 'RED',
        },
        {
          id: '3', type: 'artifact', title: '法律意见书草稿',
          description: '关于合同解除条件的法律意见书（草稿）',
          requester: 'drafter-agent', requested_at: new Date().toISOString(),
          status: 'pending',
        },
      ]);
      setError('API 未连接 — 显示演示数据');
    } finally {
      setLoading(false);
    }
  }, [filter]);

  useEffect(() => {
    fetchApprovals();
  }, [fetchApprovals]);

  // SSE: 新的审批请求实时推送
  useSSE((event) => {
    if (event.type === 'approval_requested') {
      fetchApprovals();
    }
  });

  const handleApprove = async (id: string) => {
    setActionLoading(id);
    try {
      await api.approve(id);
      setApprovals(prev => prev.map(a => a.id === id ? { ...a, status: 'approved' } : a));
    } catch {
      setApprovals(prev => prev.map(a => a.id === id ? { ...a, status: 'approved' } : a));
    } finally {
      setActionLoading(null);
    }
  };

  const handleReject = async (id: string) => {
    setActionLoading(id);
    try {
      await api.reject(id, '人工审批拒绝');
      setApprovals(prev => prev.map(a => a.id === id ? { ...a, status: 'rejected' } : a));
    } catch {
      setApprovals(prev => prev.map(a => a.id === id ? { ...a, status: 'rejected' } : a));
    } finally {
      setActionLoading(null);
    }
  };

  const filtered = approvals.filter(a => filter === 'all' || a.status === filter);

  if (loading) return <div className="flex justify-center py-12">加载中...</div>;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">审批工作台</h1>
          <p className="text-sm text-gray-500 mt-1">审批 AI Agent 提交的法律操作请求</p>
        </div>
        <div className="flex space-x-2">
          {(['all', 'pending', 'approved', 'rejected'] as const).map(f => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition ${
                filter === f
                  ? 'bg-legal-600 text-white'
                  : 'bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300 border border-gray-200 dark:border-gray-700 hover:bg-gray-50'
              }`}
            >
              {f === 'all' ? '全部' : f === 'pending' ? '待审批' : f === 'approved' ? '已批准' : '已拒绝'}
            </button>
          ))}
        </div>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-800 text-sm">
          ⚠️ {error}
        </div>
      )}

      <div className="bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-200 dark:border-gray-700 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-900">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">类型</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">标题</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">申请人</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">时间</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
            {filtered.map(item => (
              <tr key={item.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                <td className="px-6 py-4 whitespace-nowrap">{getTypeBadge(item.type)}</td>
                <td className="px-6 py-4">
                  <div className="text-sm font-medium text-gray-900 dark:text-white">{item.title}</div>
                  <div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-md">{item.description}</div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">{item.requester}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {format(new Date(item.requested_at), 'MM-dd HH:mm')}
                </td>
                <td className="px-6 py-4 whitespace-nowrap">{getStatusBadge(item.status)}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {item.status === 'pending' && (
                    <div className="flex space-x-3">
                      <button
                        onClick={() => handleApprove(item.id)}
                        disabled={actionLoading === item.id}
                        className="text-green-600 hover:text-green-700 font-medium disabled:opacity-50"
                      >
                        批准
                      </button>
                      <button
                        onClick={() => handleReject(item.id)}
                        disabled={actionLoading === item.id}
                        className="text-red-600 hover:text-red-700 font-medium disabled:opacity-50"
                      >
                        拒绝
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        {filtered.length === 0 && (
          <div className="text-center py-12 text-gray-500">
            <CheckCircle className="w-12 h-12 mx-auto mb-4 text-gray-300" />
            <p>没有{filter === 'pending' ? '待审批' : ''}事项</p>
          </div>
        )}
      </div>
    </div>
  );
}

function getStatusBadge(status: string) {
  switch (status) {
    case 'approved':
      return (
        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400">
          <CheckCircle className="w-3 h-3 mr-1" />已批准
        </span>
      );
    case 'rejected':
      return (
        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400">
          <XCircle className="w-3 h-3 mr-1" />已拒绝
        </span>
      );
    default:
      return (
        <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">
          <Clock className="w-3 h-3 mr-1" />待审批
        </span>
      );
  }
}

function getTypeBadge(type: string) {
  const colors: Record<string, string> = {
    intent: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400',
    finding: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400',
    artifact: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400',
  };
  return (
    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${colors[type] || colors.intent}`}>
      {type}
    </span>
  );
}
