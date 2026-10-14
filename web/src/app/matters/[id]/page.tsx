'use client';

import { useState, useEffect, useCallback } from 'react';
import ReactFlow, {
  Node, Edge, Background, Controls, MiniMap,
  useNodesState, useEdgesState, MarkerType,
} from 'reactflow';
import 'reactflow/dist/style.css';
import { useParams } from 'next/navigation';
import { AlertTriangle, Shield, FileText, Users, Upload } from 'lucide-react';
import { api, MatterDetail } from '@/lib/api';
import { useSSE } from '@/hooks/useSSE';

export default function MatterPage() {
  const params = useParams();
  const matterId = params.id as string;

  const [matter, setMatter] = useState<MatterDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [selectedNode, setSelectedNode] = useState<Node | null>(null);
  const [activeTab, setActiveTab] = useState<'graph' | 'findings' | 'entities'>('graph');
  const [showUpload, setShowUpload] = useState(false);
  const [contractText, setContractText] = useState('');
  const [uploading, setUploading] = useState(false);

  const fetchMatterDetail = useCallback(async () => {
    try {
      const data = await api.getMatter(matterId);
      setMatter(data);
      buildGraph(data);
    } catch {
      // 演示模式：构造空数据
      setMatter({
        matter_id: matterId,
        stats: { entity_count: 0, claim_count: 0, finding_count: 0, red_count: 0, orange_count: 0 },
        entities: {}, claims: {}, findings: {},
      });
    } finally {
      setLoading(false);
    }
  }, [matterId]);

  useEffect(() => {
    fetchMatterDetail();
  }, [fetchMatterDetail]);

  // SSE 实时更新替代轮询
  useSSE((event) => {
    if (event.type === 'graph_updated' || event.type === 'finding_created') {
      const data = event.data as { matter_id?: string };
      if (!data.matter_id || data.matter_id === matterId) {
        fetchMatterDetail();
      }
    }
  });

  const buildGraph = useCallback((data: MatterDetail) => {
    const newNodes: Node[] = [];
    const newEdges: Edge[] = [];
    let x = 0, y = 0;

    Object.values(data.entities || {}).forEach((entity) => {
      newNodes.push({
        id: entity.id,
        type: 'entity',
        position: { x: x * 250, y: y * 100 },
        data: { label: entity.name, ...entity },
        className: 'react-flow__node-entity',
      });
      x++;
      if (x > 3) { x = 0; y++; }
    });

    Object.values(data.findings || {}).forEach((finding, idx) => {
      const nodeId = `finding-${finding.id}`;
      newNodes.push({
        id: nodeId,
        type: 'finding',
        position: { x: 400, y: idx * 150 },
        data: { label: finding.title, ...finding },
        className: 'react-flow__node-finding',
      });

      finding.entity_refs?.forEach(refId => {
        newEdges.push({
          id: `edge-${refId}-${nodeId}`,
          source: refId,
          target: nodeId,
          animated: finding.level === 'RED',
          style: { stroke: getRiskColor(finding.level), strokeWidth: 2 },
          markerEnd: { type: MarkerType.ArrowClosed },
        });
      });
    });

    setNodes(newNodes);
    setEdges(newEdges);
  }, [setNodes, setEdges]);

  const getRiskColor = (level: string) => {
    switch (level) {
      case 'RED': return '#ef4444';
      case 'ORANGE': return '#f97316';
      case 'YELLOW': return '#eab308';
      default: return '#22c55e';
    }
  };

  const handleUpload = async () => {
    if (!contractText.trim()) return;
    setUploading(true);
    try {
      await api.uploadContract(matterId, contractText);
      setShowUpload(false);
      setContractText('');
      fetchMatterDetail();
    } catch { /* 演示模式忽略 */ } finally {
      setUploading(false);
    }
  };

  const onNodeClick = useCallback((_event: React.MouseEvent, node: Node) => {
    setSelectedNode(node);
  }, []);

  if (loading) return <div className="flex justify-center py-12">加载中...</div>;
  if (!matter) return null;

  return (
    <div className="min-h-screen">
      {/* 头部统计 */}
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">案件详情</h1>
          <p className="text-sm text-gray-500 font-mono mt-1">{matterId}</p>
        </div>
        <div className="flex items-center space-x-6">
          <MiniStat icon={<Users className="w-4 h-4" />} label="主体" value={matter.stats.entity_count} color="text-legal-600" />
          <MiniStat icon={<FileText className="w-4 h-4" />} label="主张" value={matter.stats.claim_count} color="text-yellow-600" />
          <MiniStat icon={<AlertTriangle className="w-4 h-4" />} label="风险" value={matter.stats.finding_count} color="text-red-600" />
          <MiniStat icon={<Shield className="w-4 h-4" />} label="红旗" value={matter.stats.red_count} color="text-red-700" />
          <button
            onClick={() => setShowUpload(true)}
            className="flex items-center px-4 py-2 bg-legal-600 text-white rounded-lg hover:bg-legal-700 transition text-sm"
          >
            <Upload className="w-4 h-4 mr-2" />
            上传合同
          </button>
        </div>
      </div>

      {/* 上传弹窗 */}
      {showUpload && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-white dark:bg-gray-800 rounded-xl p-6 w-full max-w-2xl shadow-2xl">
            <h2 className="text-lg font-bold mb-4 dark:text-white">上传合同文本</h2>
            <textarea
              value={contractText}
              onChange={e => setContractText(e.target.value)}
              placeholder="粘贴合同文本内容，AI Agent 将自动分析风险..."
              rows={12}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg dark:bg-gray-700 dark:text-white focus:ring-2 focus:ring-legal-500 font-mono text-sm"
            />
            <div className="flex space-x-3 mt-4">
              <button
                onClick={handleUpload}
                disabled={uploading || !contractText.trim()}
                className="flex-1 px-4 py-2 bg-legal-600 text-white rounded-lg hover:bg-legal-700 disabled:opacity-50"
              >
                {uploading ? '分析中...' : '开始分析'}
              </button>
              <button
                onClick={() => setShowUpload(false)}
                className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-600 rounded-lg hover:bg-gray-300"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Tab 切换 */}
      <div className="mb-4">
        <div className="flex space-x-1 bg-gray-100 dark:bg-gray-800 p-1 rounded-lg w-fit">
          {(['graph', 'findings', 'entities'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 rounded-md transition text-sm ${
                activeTab === tab
                  ? 'bg-white dark:bg-gray-700 shadow text-legal-600 dark:text-legal-400'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900'
              }`}
            >
              {tab === 'graph' ? '双图可视化' : tab === 'findings' ? '风险发现' : '实体列表'}
            </button>
          ))}
        </div>
      </div>

      {/* 内容区 */}
      <div className="pb-6">
        {activeTab === 'graph' && (
          <div className="h-[600px] bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-200 dark:border-gray-700 overflow-hidden">
            <ReactFlow
              nodes={nodes} edges={edges}
              onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
              onNodeClick={onNodeClick}
              fitView attributionPosition="bottom-left"
            >
              <Background />
              <Controls />
              <MiniMap />
            </ReactFlow>
          </div>
        )}

        {activeTab === 'findings' && (
          <div className="bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-200 dark:border-gray-700 overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-900">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">等级</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">标题</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">评分</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {Object.values(matter.findings || {}).map(finding => (
                  <tr key={finding.id} className="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className="inline-flex px-2 py-1 text-xs font-semibold rounded-full"
                        style={{
                          backgroundColor: getRiskColor(finding.level) + '20',
                          color: getRiskColor(finding.level),
                        }}
                      >
                        {finding.level}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <div className="text-sm font-medium text-gray-900 dark:text-white">{finding.title}</div>
                      <div className="text-sm text-gray-500 dark:text-gray-400 truncate max-w-md">{finding.description}</div>
                      {finding.recommendation && (
                        <div className="text-xs text-legal-600 dark:text-legal-400 mt-1">💡 {finding.recommendation}</div>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                      {finding.severity}×{finding.likelihood}={finding.score}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            {Object.keys(matter.findings || {}).length === 0 && (
              <div className="text-center py-12 text-gray-500">暂无风险发现，上传合同开始分析</div>
            )}
          </div>
        )}

        {activeTab === 'entities' && (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Object.values(matter.entities || {}).map(entity => (
              <div key={entity.id} className="bg-white dark:bg-gray-800 rounded-xl shadow border border-gray-200 dark:border-gray-700 p-4">
                <span className="text-xs font-medium text-gray-500 bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                  {entity.type}
                </span>
                <h3 className="font-semibold text-gray-900 dark:text-white mt-2">{entity.name}</h3>
                <div className="mt-2 text-sm text-gray-600 dark:text-gray-400">
                  {Object.entries(entity.attributes || {}).slice(0, 3).map(([k, v]) => (
                    <div key={k} className="flex justify-between">
                      <span className="text-gray-400">{k}:</span>
                      <span className="truncate max-w-[200px]">{v}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
            {Object.keys(matter.entities || {}).length === 0 && (
              <div className="col-span-full text-center py-12 text-gray-500">暂无实体数据</div>
            )}
          </div>
        )}
      </div>

      {/* 节点详情侧边栏 */}
      {selectedNode && (
        <div className="fixed right-0 top-0 h-full w-96 bg-white dark:bg-gray-800 shadow-xl border-l border-gray-200 dark:border-gray-700 p-6 overflow-y-auto z-50">
          <div className="flex justify-between items-start mb-4">
            <h2 className="text-lg font-bold dark:text-white">节点详情</h2>
            <button onClick={() => setSelectedNode(null)} className="text-gray-400 hover:text-gray-600">✕</button>
          </div>
          <pre className="bg-gray-50 dark:bg-gray-900 rounded p-4 text-sm overflow-auto dark:text-gray-300">
            {JSON.stringify(selectedNode.data, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function MiniStat({ icon, label, value, color }: { icon: React.ReactNode; label: string; value: number; color: string }) {
  return (
    <div className="flex items-center space-x-1.5">
      <div className={color}>{icon}</div>
      <div>
        <div className={`text-lg font-bold ${color}`}>{value}</div>
        <div className="text-xs text-gray-500">{label}</div>
      </div>
    </div>
  );
}
