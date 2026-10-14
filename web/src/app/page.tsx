'use client';

import { useState, useEffect, useCallback } from 'react';
import { api, Matter } from '@/lib/api';
import { Plus, FileText, Trash2, Clock } from 'lucide-react';
import { format } from 'date-fns';

export default function MattersPage() {
  const [matters, setMatters] = useState<Matter[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');

  const fetchMatters = useCallback(async () => {
    try {
      setError(null);
      const data = await api.listMatters();
      setMatters(data);
    } catch (err) {
      // API 未连接时使用演示数据
      setMatters([
        {
          matter_id: 'demo-001',
          name: '示例：保密协议审查',
          status: 'active',
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
        {
          matter_id: 'demo-002',
          name: '示例：劳动合同审查',
          status: 'active',
          created_at: new Date(Date.now() - 86400000).toISOString(),
          updated_at: new Date(Date.now() - 86400000).toISOString(),
        },
        {
          matter_id: 'demo-003',
          name: '示例：采购合同审查',
          status: 'archived',
          created_at: new Date(Date.now() - 172800000).toISOString(),
          updated_at: new Date(Date.now() - 172800000).toISOString(),
        },
      ]);
      setError('API 未连接 — 显示演示数据');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMatters();
  }, [fetchMatters]);

  const handleCreate = async () => {
    if (!newName.trim()) return;
    try {
      const matter = await api.createMatter({ name: newName, description: newDesc });
      setMatters(prev => [matter, ...prev]);
    } catch {
      // 演示模式
      const fake: Matter = {
        matter_id: `local-${Date.now()}`,
        name: newName,
        status: 'active',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };
      setMatters(prev => [fake, ...prev]);
    }
    setNewName('');
    setNewDesc('');
    setShowCreate(false);
  };

  const handleDelete = async (id: string) => {
    try {
      await api.deleteMatter(id);
    } catch { /* 演示模式 */ }
    setMatters(prev => prev.filter(m => m.matter_id !== id));
  };

  if (loading) return <LoadingSkeleton />;

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">案件管理</h1>
          <p className="text-sm text-gray-500 mt-1">管理法律事务案件，上传合同进行 AI 审查</p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="flex items-center px-4 py-2 bg-legal-600 text-white rounded-lg hover:bg-legal-700 transition"
        >
          <Plus className="w-4 h-4 mr-2" />
          新建案件
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-800 text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* 创建弹窗 */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-md shadow-2xl">
            <h2 className="text-lg font-bold mb-4">新建案件</h2>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  案件名称 *
                </label>
                <input
                  type="text"
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  placeholder="例如：某某公司采购合同审查"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white focus:ring-2 focus:ring-legal-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  描述
                </label>
                <textarea
                  value={newDesc}
                  onChange={e => setNewDesc(e.target.value)}
                  placeholder="案件背景描述（可选）"
                  rows={3}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white focus:ring-2 focus:ring-legal-500"
                />
              </div>
              <div className="flex space-x-3 pt-2">
                <button
                  onClick={handleCreate}
                  className="flex-1 px-4 py-2 bg-legal-600 text-white rounded-lg hover:bg-legal-700"
                >
                  创建
                </button>
                <button
                  onClick={() => setShowCreate(false)}
                  className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-600 rounded-lg hover:bg-gray-300"
                >
                  取消
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 案件列表 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {matters.map(matter => (
          <div
            key={matter.matter_id}
            className="bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-200 dark:border-gray-700 p-5 hover:shadow-lg transition group"
          >
            <div className="flex items-start justify-between">
              <div className="flex items-center space-x-3">
                <div className="p-2 bg-legal-50 dark:bg-legal-900/30 rounded-lg">
                  <FileText className="w-5 h-5 text-legal-600 dark:text-legal-400" />
                </div>
                <div>
                  <h3 className="font-semibold text-gray-900 dark:text-white group-hover:text-legal-600 transition">
                    {matter.name}
                  </h3>
                  <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium mt-1 ${
                    matter.status === 'active'
                      ? 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
                      : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
                  }`}>
                    {matter.status === 'active' ? '进行中' : '已归档'}
                  </span>
                </div>
              </div>
              <button
                onClick={() => handleDelete(matter.matter_id)}
                className="opacity-0 group-hover:opacity-100 text-gray-400 hover:text-red-500 transition"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            </div>
            <div className="mt-4 flex items-center justify-between">
              <span className="text-xs text-gray-400 flex items-center">
                <Clock className="w-3 h-3 mr-1" />
                {format(new Date(matter.created_at), 'yyyy-MM-dd HH:mm')}
              </span>
              <a
                href={`/matters/${matter.matter_id}`}
                className="text-sm text-legal-600 hover:text-legal-700 font-medium"
              >
                查看详情 →
              </a>
            </div>
          </div>
        ))}
      </div>

      {matters.length === 0 && (
        <div className="text-center py-12 text-gray-500">
          <FileText className="w-12 h-12 mx-auto mb-4 text-gray-300" />
          <p>暂无案件，点击「新建案件」开始</p>
        </div>
      )}
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="animate-pulse">
      <div className="h-8 bg-gray-200 rounded w-1/4 mb-6" />
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {[1, 2, 3].map(i => (
          <div key={i} className="bg-white rounded-xl shadow p-5 h-40" />
        ))}
      </div>
    </div>
  );
}
