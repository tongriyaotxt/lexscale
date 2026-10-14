'use client';

import { useState, useEffect, useCallback } from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { AlertTriangle, CheckCircle, Clock, Shield, RefreshCw } from 'lucide-react';
import { api, DashboardStats } from '@/lib/api';
import { useSSE } from '@/hooks/useSSE';

export default function DashboardPage() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    try {
      setError(null);
      const data = await api.getStats();
      setStats(data);
    } catch {
      // 演示数据
      setStats({
        total_matters: 12,
        active_matters: 5,
        total_findings: 48,
        red_findings: 3,
        orange_findings: 8,
        yellow_findings: 15,
        green_findings: 22,
        pending_approvals: 4,
      });
      setError('API 未连接 — 显示演示数据');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  // SSE 实时更新
  useSSE((event) => {
    if (event.type === 'graph_updated' || event.type === 'finding_created') {
      fetchStats();
    }
  });

  if (loading) return <div className="flex justify-center py-12">加载中...</div>;
  if (!stats) return null;

  const riskData = [
    { name: 'RED', value: stats.red_findings, color: '#ef4444' },
    { name: 'ORANGE', value: stats.orange_findings, color: '#f97316' },
    { name: 'YELLOW', value: stats.yellow_findings, color: '#eab308' },
    { name: 'GREEN', value: stats.green_findings, color: '#22c55e' },
  ];

  const matterData = [
    { name: '进行中', value: stats.active_matters },
    { name: '已归档', value: stats.total_matters - stats.active_matters },
  ];

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">风险仪表盘</h1>
          <p className="text-sm text-gray-500 mt-1">实时法律风险监控总览</p>
        </div>
        <button
          onClick={fetchStats}
          className="flex items-center px-3 py-2 text-sm text-gray-600 hover:text-legal-600 border border-gray-300 rounded-lg hover:border-legal-300 transition"
        >
          <RefreshCw className="w-4 h-4 mr-1" />
          刷新
        </button>
      </div>

      {error && (
        <div className="mb-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-800 text-sm">
          ⚠️ {error}
        </div>
      )}

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
        <StatCard title="总案件" value={stats.total_matters} icon={<Shield className="w-6 h-6" />} bgColor="bg-legal-50" textColor="text-legal-600" />
        <StatCard title="进行中" value={stats.active_matters} icon={<Clock className="w-6 h-6" />} bgColor="bg-blue-50" textColor="text-blue-600" />
        <StatCard title="风险发现" value={stats.total_findings} icon={<AlertTriangle className="w-6 h-6" />} bgColor="bg-yellow-50" textColor="text-yellow-600" />
        <StatCard title="待审批" value={stats.pending_approvals} icon={<CheckCircle className="w-6 h-6" />} bgColor="bg-red-50" textColor="text-red-600" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* 风险分布饼图 */}
        <div className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow border border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold mb-4 dark:text-white">风险等级分布</h2>
          <ResponsiveContainer width="100%" height={300}>
            <PieChart>
              <Pie data={riskData} cx="50%" cy="50%" innerRadius={60} outerRadius={100} paddingAngle={5} dataKey="value">
                {riskData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
          <div className="flex justify-center flex-wrap gap-4 mt-4">
            {riskData.map(item => (
              <div key={item.name} className="flex items-center">
                <div className="w-3 h-3 rounded-full mr-2" style={{ backgroundColor: item.color }} />
                <span className="text-sm text-gray-600 dark:text-gray-400">{item.name}: {item.value}</span>
              </div>
            ))}
          </div>
        </div>

        {/* 案件状态柱状图 */}
        <div className="bg-white dark:bg-gray-800 p-6 rounded-xl shadow border border-gray-200 dark:border-gray-700">
          <h2 className="text-lg font-semibold mb-4 dark:text-white">案件状态</h2>
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={matterData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" />
              <YAxis />
              <Tooltip />
              <Bar dataKey="value" fill="#0284c7" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}

function StatCard({ title, value, icon, bgColor, textColor }: {
  title: string;
  value: number;
  icon: React.ReactNode;
  bgColor: string;
  textColor: string;
}) {
  return (
    <div className={`${bgColor} dark:bg-opacity-10 rounded-xl p-6 border border-transparent`}>
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-600 dark:text-gray-400">{title}</p>
          <p className={`text-3xl font-bold mt-1 ${textColor}`}>{value}</p>
        </div>
        <div className={textColor}>{icon}</div>
      </div>
    </div>
  );
}
