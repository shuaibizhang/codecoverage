import React, { useState, useEffect } from 'react';
import { CoverageTree } from './CoverageTree';
import { SourceView } from './SourceView';
import { Layout, ShieldCheck, Database, Calendar, ChevronRight, Activity, Search, RefreshCw, Layers } from 'lucide-react';
import { getReportInfo, getTreeNodes, getFileCoverage } from './api';
import { NodeType, type ReportInfo, type TreeNode, type FileCoverage } from './types';

const TEST_TYPES = [
  { id: 'unittest', label: '单元测试', icon: ShieldCheck },
  { id: 'integrate', label: '集成测试', icon: Layers },
  { id: 'online', label: '线上测试', icon: Database },
  { id: 'auto', label: '自动化测试', icon: Activity },
];

function App() {
  const [testType, setTestType] = useState(TEST_TYPES[0].id);
  const [module, setModule] = useState("github.com/shuaibizhang/codecoverage");
  const [branch, setBranch] = useState("main");
  const [commit, setCommit] = useState("latest");

  const [reportInfo, setReportInfo] = useState<ReportInfo | null>(null);
  const [rootNode, setRootNode] = useState<TreeNode | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileData, setFileData] = useState<FileCoverage | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchReport = async () => {
    try {
      setLoading(true);
      setError(null);
      const info = await getReportInfo(testType, module, branch, commit);
      setReportInfo(info);

      const nodes = await getTreeNodes(info.report_id, "*");
      if (nodes.length > 0) {
        const root: TreeNode = {
          name: "root",
          path: "*",
          type: NodeType.Dir,
          stat: nodes[0].stat,
          children: nodes
        };

        const firstNode = nodes[0];
        if (nodes.length === 1 && firstNode.type === NodeType.Dir) {
          root.stat = firstNode.stat;
        } else {
          const aggregateStat = nodes.reduce((acc, node) => ({
            total_lines: acc.total_lines + node.stat.total_lines,
            instr_lines: acc.instr_lines + node.stat.instr_lines,
            cover_lines: acc.cover_lines + node.stat.cover_lines,
            coverage: 0,
            add_lines: acc.add_lines + node.stat.add_lines,
            delete_lines: acc.delete_lines + node.stat.delete_lines,
            incr_instr_lines: acc.incr_instr_lines + node.stat.incr_instr_lines,
            incr_cover_lines: acc.incr_cover_lines + node.stat.incr_cover_lines,
            incr_coverage: 0
          }), {
            total_lines: 0, instr_lines: 0, cover_lines: 0, coverage: 0,
            add_lines: 0, delete_lines: 0, incr_instr_lines: 0, incr_cover_lines: 0, incr_coverage: 0
          });
          if (aggregateStat.instr_lines > 0) {
            aggregateStat.coverage = Math.round((aggregateStat.cover_lines / aggregateStat.instr_lines) * 100);
          }
          if (aggregateStat.incr_instr_lines > 0) {
            aggregateStat.incr_coverage = Math.round((aggregateStat.incr_cover_lines / aggregateStat.incr_instr_lines) * 100);
          }
          root.stat = aggregateStat;
        }
        setRootNode(root);
      } else {
        setRootNode(null);
        setError("未找到该报告的树节点数据。");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取报告失败');
      setReportInfo(null);
      setRootNode(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchReport();
  }, [testType]);

  const handleFileClick = async (path: string) => {
    setSelectedPath(path);
    if (reportInfo) {
      try {
        const data = await getFileCoverage(reportInfo.report_id, path);
        setFileData(data);
      } catch (err) {
        console.error("Failed to fetch file coverage:", err);
        setFileData(null);
      }
    }
  };

  const SummaryCard = ({ title, value, subValue, colorClass }: any) => (
    <div className="bg-[#161b22] border border-gray-800 p-4 rounded-lg">
      <div className="text-xs text-gray-500 uppercase tracking-wider mb-1">{title}</div>
      <div className={`text-2xl font-bold ${colorClass}`}>{value}</div>
      {subValue && <div className="text-xs text-gray-400 mt-1">{subValue}</div>}
    </div>
  );

  return (
    <div className="flex flex-col h-screen w-screen bg-[#0d1117] text-gray-200 overflow-hidden">
      <header className="flex flex-col border-b border-gray-800 bg-[#161b22]">
        <div className="h-14 flex items-center justify-between px-6 border-b border-gray-800/50">
          <div className="flex items-center space-x-3">
            <ShieldCheck className="text-green-500" size={24} />
            <h1 className="text-base font-bold tracking-tight">代码覆盖率分析平台</h1>
          </div>
          
          <div className="flex items-center space-x-4">
            <div className="flex items-center bg-[#0d1117] border border-gray-700 rounded px-2 py-1">
              <span className="text-[10px] text-gray-500 mr-2 uppercase">Module</span>
              <input 
                className="bg-transparent border-none outline-none text-xs w-48 text-gray-300" 
                value={module} onChange={e => setModule(e.target.value)}
              />
            </div>
            <div className="flex items-center bg-[#0d1117] border border-gray-700 rounded px-2 py-1">
              <span className="text-[10px] text-gray-500 mr-2 uppercase">Branch</span>
              <input 
                className="bg-transparent border-none outline-none text-xs w-24 text-gray-300" 
                value={branch} onChange={e => setBranch(e.target.value)}
              />
            </div>
            <div className="flex items-center bg-[#0d1117] border border-gray-700 rounded px-2 py-1">
              <span className="text-[10px] text-gray-500 mr-2 uppercase">Commit</span>
              <input 
                className="bg-transparent border-none outline-none text-xs w-24 text-gray-300" 
                value={commit} onChange={e => setCommit(e.target.value)}
              />
            </div>
            <button 
              onClick={fetchReport}
              className="bg-green-600 hover:bg-green-700 text-white p-1.5 rounded transition-colors"
            >
              <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
            </button>
          </div>
        </div>

        <div className="flex px-6 space-x-8">
          {TEST_TYPES.map(type => (
            <button
              key={type.id}
              onClick={() => setTestType(type.id)}
              className={`flex items-center space-x-2 py-3 border-b-2 transition-all text-sm font-medium ${
                testType === type.id 
                ? "border-green-500 text-green-500" 
                : "border-transparent text-gray-500 hover:text-gray-300"
              }`}
            >
              <type.icon size={16} />
              <span>{type.label}</span>
            </button>
          ))}
        </div>
      </header>

      <section className="p-6 bg-[#0d1117] border-b border-gray-800">
        {rootNode ? (
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <SummaryCard 
              title="全量覆盖率" 
              value={`${rootNode.stat.coverage}%`} 
              subValue={`${rootNode.stat.cover_lines} / ${rootNode.stat.instr_lines} 指令行`}
              colorClass="text-green-500"
            />
            <SummaryCard 
              title="增量覆盖率" 
              value={`${rootNode.stat.incr_coverage}%`} 
              subValue={`${rootNode.stat.incr_cover_lines} / ${rootNode.stat.incr_instr_lines} 增量行`}
              colorClass="text-blue-500"
            />
            <SummaryCard 
              title="代码规模" 
              value={rootNode.stat.total_lines.toLocaleString()} 
              subValue="总行数"
              colorClass="text-gray-300"
            />
            <SummaryCard 
              title="变更统计" 
              value={`+${rootNode.stat.add_lines} / -${rootNode.stat.delete_lines}`} 
              subValue="新增 / 删除行"
              colorClass="text-orange-500"
            />
          </div>
        ) : (
          <div className="h-24 flex items-center justify-center text-gray-500 italic">
            {loading ? "正在加载概览数据..." : (error || "暂无报告概览数据")}
          </div>
        )}
      </section>

      <main className="flex flex-grow overflow-hidden bg-[#0d1117]">
        <aside className="w-1/3 min-w-[320px] max-w-md h-full flex flex-col border-r border-gray-800">
          <div className="p-3 bg-[#161b22] border-b border-gray-800 text-xs font-semibold text-gray-500 uppercase tracking-widest">
            目录结构
          </div>
          {rootNode ? (
            <CoverageTree 
              tree={rootNode as any} 
              onFileClick={handleFileClick} 
              selectedPath={selectedPath}
              reportId={reportInfo?.report_id || ""}
            />
          ) : (
            <div className="flex-grow flex items-center justify-center text-gray-600 text-sm">
              无目录数据
            </div>
          )}
        </aside>

        <section className="flex-grow h-full overflow-hidden flex flex-col">
          <div className="p-3 bg-[#161b22] border-b border-gray-800 text-xs font-semibold text-gray-500 flex justify-between items-center">
            <span className="uppercase tracking-widest">代码详情 {selectedPath && `: ${selectedPath}`}</span>
            {selectedPath && (
              <div className="flex items-center space-x-4 text-[10px]">
                <span className="flex items-center"><div className="w-2 h-2 rounded-full bg-green-500 mr-1"></div> 已覆盖</span>
                <span className="flex items-center"><div className="w-2 h-2 rounded-full bg-red-500 mr-1"></div> 未覆盖</span>
                <span className="flex items-center"><div className="w-2 h-2 rounded-full bg-gray-600 mr-1"></div> 非指令行</span>
              </div>
            )}
          </div>
          <div className="flex-grow overflow-hidden">
            {selectedPath ? (
              fileData ? (
                <SourceView 
                  filePath={selectedPath} 
                  content={fileData.content} 
                  coverage={fileData.lines} 
                />
              ) : (
                <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center">
                  <div className="animate-pulse flex flex-col items-center">
                    <Layout size={48} className="mb-4 opacity-20" />
                    <p className="text-lg">加载源码中...</p>
                  </div>
                </div>
              )
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center">
                <Search size={64} className="mb-6 opacity-10" />
                <h2 className="text-xl font-light mb-2">选择文件以查看详情</h2>
                <p className="max-w-xs text-xs text-gray-600 leading-relaxed">
                  在左侧树形结构中选择具体的源码文件，即可在此处查看行级覆盖率染色及统计详情。
                </p>
              </div>
            )}
          </div>
        </section>
      </main>
    </div>
  );
}

export default App;
