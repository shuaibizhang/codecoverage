import { useState, useEffect } from 'react';
import { X, Plus, GitMerge, AlertCircle, CheckCircle2 } from 'lucide-react';
import { getMetadataList, mergeReports } from './api';
import { type ReportSelector } from './types';

interface MergeModalProps {
  isOpen: boolean;
  onClose: () => void;
  baseReport: ReportSelector;
  testTypes: { id: string, label: string }[];
  onMergeSuccess: (mergedReportId: string) => void;
}

export function MergeModal({ isOpen, onClose, baseReport, testTypes, onMergeSuccess }: MergeModalProps) {
  const [otherReports, setOtherReports] = useState<ReportSelector[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const [availableTestTypes, setAvailableTestTypes] = useState<{ id: string, label: string }[]>([]);

  // 为每个待合并报告维护独立的元数据列表
  const [metadataMap, setMetadataMap] = useState<Record<number, { branches: string[], commits: string[] }>>({});

  // 初始加载：过滤掉当前模块下没有数据的测试类型
  useEffect(() => {
    if (isOpen) {
      const checkAvailableTypes = async () => {
        setLoading(true);
        try {
          const checks = await Promise.all(
            testTypes.map(async (t) => {
              try {
                // 明确传递当前模块进行过滤
                const data = await getMetadataList(t.id, baseReport.module);
                // 只有当该模块下确实有分支数据时，才认为该类型可用
                return (data.branches && data.branches.length > 0) ? t : null;
              } catch (e) {
                console.error(`Error checking metadata for ${t.id}:`, e);
                return null;
              }
            })
          );
          const filtered = checks.filter((t): t is { id: string, label: string } => t !== null);
          setAvailableTestTypes(filtered);

          // 初始添加一个报告，使用第一个可用的测试类型
          if (otherReports.length === 0 && filtered.length > 0) {
            const newReport: ReportSelector = {
              module: baseReport.module,
              branch: "", // 初始为空，由 fetchMetadataForIndex 填充
              commit: "",
              type: filtered[0].id
            };
            setOtherReports([newReport]);
            fetchMetadataForIndex(0, newReport.type);
          } else if (filtered.length === 0) {
            setError("当前模块在其他测试类型下暂无数据，无法进行合并。");
          }
        } catch {
          setError("获取可用测试类型失败");
        } finally {
          setLoading(false);
        }
      };
      checkAvailableTypes();
    } else {
       // 关闭时重置状态
       setOtherReports([]);
       setMetadataMap({});
       setAvailableTestTypes([]);
       setError(null);
     }
  }, [isOpen, baseReport.module]);

  const addOtherReport = () => {
    if (availableTestTypes.length === 0) return;
    
    const newReport: ReportSelector = {
      module: baseReport.module,
      branch: "",
      commit: "",
      type: availableTestTypes[0].id
    };
    const newIndex = otherReports.length;
    setOtherReports([...otherReports, newReport]);
    fetchMetadataForIndex(newIndex, newReport.type);
  };

  const removeOtherReport = (index: number) => {
    const newList = [...otherReports];
    newList.splice(index, 1);
    setOtherReports(newList);
    
    const newMetadataMap = { ...metadataMap };
    delete newMetadataMap[index];
    setMetadataMap(newMetadataMap);
  };

  const fetchMetadataForIndex = async (index: number, type: string) => {
    try {
      // 1. 先获取该类型下的所有分支
      const data = await getMetadataList(type, baseReport.module);
      
      let initialBranch = "";
      let initialCommits: string[] = [];
      
      if (data.branches.length > 0) {
        initialBranch = data.branches[0];
        // 2. 级联获取第一个分支下的提交
        const commitData = await getMetadataList(type, baseReport.module, initialBranch);
        initialCommits = commitData.commits;
      }

      setMetadataMap(prev => ({
        ...prev,
        [index]: { branches: data.branches, commits: initialCommits }
      }));
      
      // 默认选择第一个分支和提交，并确保 module 字段被设置
      if (initialBranch) {
        setOtherReports(prev => {
          const newList = [...prev];
          if (newList[index]) {
            newList[index] = { 
              ...newList[index], 
              module: baseReport.module, // 确保 module 字段始终与基准报告一致
              branch: initialBranch,
              commit: initialCommits.length > 0 ? initialCommits[0] : "" 
            };
          }
          return newList;
        });
      }
    } catch (err) {
      console.error(`Failed to fetch metadata for index ${index}:`, err);
    }
  };

  const updateOtherReport = async (index: number, field: keyof ReportSelector, value: string) => {
    const newList = [...otherReports];
    // 每次更新时都强制同步 module 字段
    newList[index] = { ...newList[index], [field]: value, module: baseReport.module };
    setOtherReports(newList);

    if (field === 'type') {
      fetchMetadataForIndex(index, value);
    } else if (field === 'branch') {
      // 级联刷新：当分支改变时，刷新该分支下的提交列表
      try {
        const data = await getMetadataList(newList[index].type, baseReport.module, value);
        setMetadataMap(prev => ({
          ...prev,
          [index]: { ...prev[index], commits: data.commits }
        }));
        // 默认选择第一个提交
        if (data.commits.length > 0) {
          setOtherReports(prev => {
            const upList = [...prev];
            if (upList[index]) {
              upList[index] = { ...upList[index], commit: data.commits[0] };
            }
            return upList;
          });
        }
      } catch (err) {
        console.error(`Failed to fetch commits for branch ${value}:`, err);
      }
    }
  };

  const handleMerge = async () => {
    if (otherReports.length === 0) {
      setError("请至少添加一个待合并报告");
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const resp = await mergeReports({
        base_report: { selector: baseReport },
        other_reports: otherReports.map(r => ({ selector: r }))
      });
      
      if (resp.success) {
        setSuccess(true);
        setTimeout(() => {
          onMergeSuccess(resp.merged_report_id);
          onClose();
          setSuccess(false);
          setOtherReports([]);
        }, 1500);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '合并失败');
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
      <div className="bg-[#161b22] border border-gray-800 rounded-xl shadow-2xl w-full max-w-2xl overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800">
          <div className="flex items-center space-x-3">
            <div className="bg-blue-600/20 p-2 rounded-lg border border-blue-600/30">
              <GitMerge className="text-blue-500" size={20} />
            </div>
            <div>
              <h2 className="text-lg font-bold text-gray-100">合并覆盖率报告</h2>
              <p className="text-xs text-gray-500">将多个报告的覆盖率数据合并到基准报告中</p>
            </div>
          </div>
          <button onClick={onClose} className="text-gray-500 hover:text-gray-300 transition-colors">
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Base Report Info */}
          <div className="bg-blue-600/5 border border-blue-600/20 rounded-lg p-4">
            <h3 className="text-[10px] font-black uppercase tracking-widest text-blue-500 mb-3">基准报告 (目标)</h3>
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div>
                <div className="text-[9px] text-gray-500 uppercase font-bold mb-1">模块</div>
                <div className="text-xs font-mono text-gray-300 truncate" title={baseReport.module}>{baseReport.module}</div>
              </div>
              <div>
                <div className="text-[9px] text-gray-500 uppercase font-bold mb-1">类型</div>
                <div className="text-xs font-bold text-gray-300 capitalize">{baseReport.type}</div>
              </div>
              <div>
                <div className="text-[9px] text-gray-500 uppercase font-bold mb-1">分支</div>
                <div className="text-xs font-mono text-gray-300 truncate" title={baseReport.branch}>{baseReport.branch}</div>
              </div>
              <div>
                <div className="text-[9px] text-gray-500 uppercase font-bold mb-1">提交</div>
                <div className="text-xs font-mono text-gray-300 truncate" title={baseReport.commit}>{baseReport.commit.substring(0, 7)}</div>
              </div>
            </div>
          </div>

          {/* Other Reports */}
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-[10px] font-black uppercase tracking-widest text-gray-500">待合并报告</h3>
              <button 
                onClick={addOtherReport}
                className="flex items-center space-x-1.5 text-[10px] font-bold text-blue-500 hover:text-blue-400 transition-colors px-2 py-1 rounded bg-blue-500/10 border border-blue-500/20"
              >
                <Plus size={12} />
                <span>添加报告</span>
              </button>
            </div>

            {otherReports.map((report, index) => (
              <div key={index} className="bg-[#0d1117] border border-gray-800 rounded-lg p-4 relative group">
                <button 
                  onClick={() => removeOtherReport(index)}
                  className="absolute -top-2 -right-2 bg-red-600/20 border border-red-600/30 text-red-500 p-1 rounded-full opacity-0 group-hover:opacity-100 transition-all hover:bg-red-600 hover:text-white"
                >
                  <X size={12} />
                </button>
                
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="space-y-1.5">
                    <label className="text-[9px] text-gray-500 uppercase font-black">测试类型</label>
                    <select 
                      className="w-full bg-[#161b22] border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 outline-none focus:border-blue-500 transition-colors"
                      value={report.type}
                      onChange={(e) => updateOtherReport(index, 'type', e.target.value)}
                    >
                      {availableTestTypes.map(t => <option key={t.id} value={t.id}>{t.label}</option>)}
                    </select>
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-[9px] text-gray-500 uppercase font-black">分支</label>
                    <select 
                      className="w-full bg-[#161b22] border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 outline-none focus:border-blue-500 transition-colors"
                      value={report.branch}
                      onChange={(e) => updateOtherReport(index, 'branch', e.target.value)}
                    >
                      {metadataMap[index]?.branches.length === 0 && <option value="">暂无数据</option>}
                      {metadataMap[index]?.branches.map(b => <option key={b} value={b}>{b}</option>)}
                    </select>
                  </div>
                  <div className="space-y-1.5">
                    <label className="text-[9px] text-gray-500 uppercase font-black">提交</label>
                    <select 
                      className="w-full bg-[#161b22] border border-gray-700 rounded px-2 py-1.5 text-xs text-gray-300 outline-none focus:border-blue-500 transition-colors"
                      value={report.commit}
                      onChange={(e) => updateOtherReport(index, 'commit', e.target.value)}
                    >
                      {metadataMap[index]?.commits.length === 0 && <option value="">暂无数据</option>}
                      {metadataMap[index]?.commits.map(c => <option key={c} value={c}>{c}</option>)}
                    </select>
                  </div>
                </div>
              </div>
            ))}

            {otherReports.length === 0 && (
              <div className="text-center py-8 border-2 border-dashed border-gray-800 rounded-xl text-gray-600 italic text-sm">
                点击上方按钮添加待合并的报告
              </div>
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-4 border-t border-gray-800 bg-[#0d1117] flex items-center justify-between">
          <div className="flex-1 mr-4">
            {error && (
              <div className="flex items-center space-x-2 text-red-500 text-xs font-bold animate-pulse">
                <AlertCircle size={14} />
                <span>{error}</span>
              </div>
            )}
            {success && (
              <div className="flex items-center space-x-2 text-green-500 text-xs font-bold">
                <CheckCircle2 size={14} />
                <span>合并成功，正在刷新...</span>
              </div>
            )}
          </div>
          <div className="flex items-center space-x-3">
            <button 
              onClick={onClose}
              className="px-4 py-2 text-xs font-bold text-gray-400 hover:text-gray-200 transition-colors"
            >
              取消
            </button>
            <button 
              onClick={handleMerge}
              disabled={loading || success || otherReports.length === 0}
              className={`flex items-center space-x-2 px-6 py-2 rounded-lg text-xs font-black uppercase tracking-widest transition-all ${
                loading || success || otherReports.length === 0
                ? 'bg-gray-800 text-gray-600 cursor-not-allowed'
                : 'bg-blue-600 text-white hover:bg-blue-500 shadow-lg shadow-blue-600/20 active:scale-95'
              }`}
            >
              {loading ? (
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              ) : (
                <GitMerge size={14} />
              )}
              <span>{loading ? '处理中...' : '开始合并'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}