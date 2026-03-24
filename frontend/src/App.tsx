import React, { useState, useEffect } from 'react';
import { CoverageTree } from './CoverageTree';
import { SourceView } from './SourceView';
import { Layout, ShieldCheck, Database, Calendar, ChevronRight, Activity, Search, RefreshCw, Layers, ChevronDown, Maximize2, Minimize2, PanelLeftClose, PanelLeftOpen } from 'lucide-react';
import { getReportInfo, getTreeNodes, getFileCoverage, getMetadataList } from './api';
import { NodeType, type ReportInfo, type TreeNode, type FileCoverage } from './types';

const TEST_TYPES = [
  { id: 'unittest', label: '单元测试', icon: ShieldCheck },
  { id: 'systest', label: '集成测试', icon: Layers },
  { id: 'online', label: '线上测试', icon: Database },
  { id: 'auto', label: '自动化测试', icon: Activity },
];

function App() {
  const [testType, setTestType] = useState(TEST_TYPES[0].id);
  const [isIncrement, setIsIncrement] = useState(false);
  const [module, setModule] = useState("");
  const [branch, setBranch] = useState("");
  const [commit, setCommit] = useState("");

  const [metadata, setMetadata] = useState<{ modules: string[], branches: string[], commits: string[] }>({
    modules: [],
    branches: [],
    commits: []
  });

  const [reportInfo, setReportInfo] = useState<ReportInfo | null>(null);
  const [rootNode, setRootNode] = useState<TreeNode | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<TreeNode | null>(null);
  const [fileData, setFileData] = useState<FileCoverage | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isMaximized, setIsMaximized] = useState(false);

  const fetchMetadata = async (type: string) => {
    try {
      setLoading(true);
      // 重置之前的状态
      setMetadata({ modules: [], branches: [], commits: [] });
      setModule("");
      setBranch("");
      setCommit("");
      setReportInfo(null);
      setRootNode(null);
      setSelectedPath(null);
      setSelectedNode(null);
      setFileData(null);

      const data = await getMetadataList(type);
      setMetadata(data);
      if (data.modules.length > 0) setModule(data.modules[0]);
      if (data.branches.length > 0) setBranch(data.branches[0]);
      if (data.commits.length > 0) setCommit(data.commits[0]);
    } catch (err) {
      console.error("Failed to fetch metadata:", err);
      setError("获取元数据失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMetadata(testType);
  }, [testType]);

  const fetchReport = async () => {
    if (!module || !branch || !commit) return;
    try {
      setLoading(true);
      setError(null);
      const info = await getReportInfo(testType, module, branch, commit);
      setReportInfo(info);

      const nodes = await getTreeNodes(info.report_id, "*", isIncrement);
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
            incr_coverage: 0,
            has_increment: acc.has_increment || node.stat.has_increment
          }), {
            total_lines: 0, instr_lines: 0, cover_lines: 0, coverage: 0,
            add_lines: 0, delete_lines: 0, incr_instr_lines: 0, incr_cover_lines: 0, incr_coverage: 0,
            has_increment: false
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
        if (!selectedNode) {
          setSelectedNode(root);
        }
      } else {
        setRootNode(null);
        setSelectedNode(null);
        setError("未找到该报告的树节点数据。");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取报告失败');
      setReportInfo(null);
      setRootNode(null);
      setSelectedNode(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (module && branch && commit) {
      fetchReport();
    }
  }, [module, branch, commit, isIncrement]);

  const handleNodeClick = async (node: TreeNode) => {
    setSelectedNode(node);
    if (node.type === NodeType.File) {
      setSelectedPath(node.path);
      if (reportInfo) {
        try {
          const data = await getFileCoverage(reportInfo.report_id, node.path);
          setFileData(data);
        } catch (err) {
          console.error("Failed to fetch file coverage:", err);
          setFileData(null);
        }
      }
    } else {
      setSelectedPath(null);
      setFileData(null);
    }
  };

  const StatItem = ({ label, value, subLabel, colorClass = "text-gray-300", large = false }: any) => (
    <div 
      className={`flex flex-col border-r border-gray-800/50 px-5 last:border-r-0 ${large ? 'min-w-[140px]' : 'min-w-[90px]'} group/stat`}
      title={`${label}: ${value}${subLabel ? ` (${subLabel})` : ''}`}
    >
      <div className="text-[9px] text-gray-500 uppercase tracking-[0.2em] mb-1.5 font-black group-hover/stat:text-gray-400 transition-colors">{label}</div>
      <div className="flex items-baseline space-x-2">
        <span className={`${large ? 'text-3xl' : 'text-xl'} font-black font-mono tracking-tighter ${colorClass} drop-shadow-sm`}>{value}</span>
        {subLabel && <span className="text-[10px] text-gray-500 font-bold font-mono opacity-80">{subLabel}</span>}
      </div>
    </div>
  );

  const displayNode = selectedNode || rootNode;

  return (
    <div className="flex flex-col h-screen w-screen bg-[#0d1117] text-gray-200 overflow-hidden">
      <header className="flex items-center h-12 px-5 border-b border-gray-800 bg-[#0d1117] shrink-0 justify-between z-10">
        <div className="flex items-center space-x-6">
          <div className="flex items-center space-x-2.5 group cursor-default">
            <div className="bg-green-600/20 p-1.5 rounded-lg border border-green-600/30 group-hover:bg-green-600/30 transition-all">
              <ShieldCheck className="text-green-500" size={18} />
            </div>
            <h1 className="text-[13px] font-black tracking-widest hidden md:block uppercase text-gray-100">代码覆盖率 <span className="text-green-500">分析系统</span></h1>
          </div>

          <div className="flex space-x-1 bg-[#161b22] p-1 rounded-lg border border-gray-800 shadow-inner">
            {TEST_TYPES.map(type => (
              <button
                key={type.id}
                onClick={() => setTestType(type.id)}
                className={`flex items-center space-x-2 px-3 py-1.5 rounded-md transition-all text-xs font-bold ${
                  testType === type.id 
                  ? "bg-green-600 text-white shadow-lg transform scale-[1.02]" 
                  : "text-gray-500 hover:text-gray-300 hover:bg-gray-800"
                }`}
              >
                <type.icon size={14} />
                <span>{type.label}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <div className="flex items-center bg-[#161b22] border border-gray-700/50 rounded-lg px-2.5 py-1.5 relative hover:border-gray-600 transition-colors group">
              <span className="text-[9px] text-gray-500 mr-2 uppercase font-black group-hover:text-gray-400">模块</span>
              <select 
                className="bg-transparent border-none outline-none text-xs w-32 text-gray-300 appearance-none pr-5 font-bold cursor-pointer" 
                value={module} onChange={e => setModule(e.target.value)}
              >
                {metadata.modules.map(m => <option key={m} value={m} className="bg-[#161b22]">{m}</option>)}
              </select>
              <ChevronDown size={10} className="absolute right-2.5 text-gray-500 pointer-events-none group-hover:text-gray-400" />
            </div>
            <div className="flex items-center bg-[#161b22] border border-gray-700/50 rounded-lg px-2.5 py-1.5 relative hover:border-gray-600 transition-colors group">
              <span className="text-[9px] text-gray-500 mr-2 uppercase font-black group-hover:text-gray-400">分支</span>
              <select 
                className="bg-transparent border-none outline-none text-xs w-20 text-gray-300 appearance-none pr-5 font-bold cursor-pointer" 
                value={branch} onChange={e => setBranch(e.target.value)}
              >
                {metadata.branches.map(b => <option key={b} value={b} className="bg-[#161b22]">{b}</option>)}
              </select>
              <ChevronDown size={10} className="absolute right-2.5 text-gray-500 pointer-events-none group-hover:text-gray-400" />
            </div>
            <div className="flex items-center bg-[#161b22] border border-gray-700/50 rounded-lg px-2.5 py-1.5 relative hover:border-gray-600 transition-colors group">
              <span className="text-[9px] text-gray-500 mr-2 uppercase font-black group-hover:text-gray-400">提交</span>
              <select 
                className="bg-transparent border-none outline-none text-xs w-20 text-gray-300 appearance-none pr-5 font-bold cursor-pointer" 
                value={commit} onChange={e => setCommit(e.target.value)}
              >
                {metadata.commits.map(c => <option key={c} value={c} className="bg-[#161b22]">{c}</option>)}
              </select>
              <ChevronDown size={10} className="absolute right-2.5 text-gray-500 pointer-events-none group-hover:text-gray-400" />
            </div>
            <button 
              onClick={fetchReport}
              className="bg-[#161b22] border border-gray-700/50 hover:bg-green-600 hover:border-green-500 text-gray-400 hover:text-white p-2 rounded-lg transition-all shadow-md active:scale-95"
              title="刷新数据"
            >
              <RefreshCw size={14} className={loading ? "animate-spin" : ""} />
            </button>
          </div>
        </div>
      </header>

      {/* 覆盖率概览通栏 - 大尺寸展示 */}
      {!isMaximized && (
        <div className="bg-[#161b22] border-b border-gray-800 px-6 py-5 shrink-0 shadow-2xl relative overflow-hidden flex items-center justify-between">
          <div className="absolute top-0 right-0 p-8 opacity-5 pointer-events-none">
            <Activity size={140} />
          </div>
          
          {displayNode ? (
            <>
              <div className="flex items-center flex-grow">
                <div className="flex flex-col pr-8 mr-8 border-r border-gray-700/50 shrink-0 min-w-[220px]">
                  <div className="text-[10px] text-gray-500 uppercase font-black mb-1.5 tracking-widest flex items-center">
                    <Layers size={10} className="mr-1.5" />
                    {displayNode.name === "root" ? "统计范围" : "当前选中节点"}
                  </div>
                  <div className={`text-xl font-black truncate max-w-[300px] tracking-tight ${isIncrement ? 'text-blue-400' : 'text-green-500'}`} title={displayNode.path}>
                    {displayNode.name === "root" ? "项目全域统计" : displayNode.name}
                  </div>
                  <div className="text-[10px] text-gray-500 mt-1.5 font-mono truncate max-w-[300px] bg-[#0d1117] px-2 py-0.5 rounded border border-gray-800 w-fit">{displayNode.path}</div>
                </div>

                <div className="flex items-center flex-grow overflow-x-auto no-scrollbar space-x-2">
                  <StatItem 
                    label="全量覆盖率" 
                    value={`${displayNode.stat.coverage}%`} 
                    subLabel={`${displayNode.stat.cover_lines} / ${displayNode.stat.instr_lines}`}
                    colorClass="text-green-500"
                    large={true}
                  />
                  <StatItem 
                    label="增量覆盖率" 
                    value={`${displayNode.stat.incr_coverage}%`} 
                    subLabel={`${displayNode.stat.incr_cover_lines} / ${displayNode.stat.incr_instr_lines}`}
                    colorClass="text-blue-400"
                    large={true}
                  />
                  
                  <div className="w-px h-12 bg-gray-800 mx-6"></div>
                  
                  <div className="grid grid-cols-4 gap-x-2 gap-y-1">
                    <StatItem label="总行数" value={displayNode.stat.total_lines.toLocaleString()} />
                    <StatItem label="指令行" value={displayNode.stat.instr_lines.toLocaleString()} />
                    <StatItem label="增量指令" value={displayNode.stat.incr_instr_lines.toLocaleString()} />
                    <StatItem 
                      label="代码变更" 
                      value={`+${displayNode.stat.add_lines}`} 
                      subLabel={`-${displayNode.stat.delete_lines}`}
                      colorClass="text-yellow-500"
                    />
                  </div>
                </div>
              </div>

              {/* 视图切换按钮搬迁至此 */}
              <div className="flex flex-col space-y-3 pl-8 ml-8 border-l border-gray-700/50 shrink-0">
                <div className="flex bg-[#0d1117] p-1 rounded-lg border border-gray-800 shadow-inner">
                  <button
                    onClick={() => setIsIncrement(false)}
                    className={`px-4 py-1.5 text-xs font-black rounded-md transition-all ${
                      !isIncrement 
                      ? "bg-gray-700 text-white shadow-lg" 
                      : "text-gray-500 hover:text-gray-300"
                    }`}
                  >
                    全量
                  </button>
                  <button
                    onClick={() => setIsIncrement(true)}
                    className={`px-4 py-1.5 text-xs font-black rounded-md transition-all ${
                      isIncrement 
                      ? "bg-blue-600 text-white shadow-lg" 
                      : "text-gray-500 hover:text-gray-300"
                    }`}
                  >
                    增量
                  </button>
                </div>
                <div className="text-[9px] text-gray-500 text-center uppercase font-black tracking-widest">视图模式切换</div>
              </div>
            </>
          ) : (
            <div className="h-16 w-full flex items-center justify-center text-gray-500 space-x-3">
              <RefreshCw size={24} className="animate-spin opacity-30" />
              <span className="text-base font-black tracking-[0.3em] uppercase italic opacity-30">报告数据分析中...</span>
            </div>
          )}
        </div>
      )}

      <main className="flex flex-grow overflow-hidden bg-[#0d1117]">
        {isSidebarOpen && !isMaximized && (
          <aside className="w-[340px] shrink-0 h-full flex flex-col border-r border-gray-800 bg-[#0d1117] z-10 transition-all duration-300">
            <CoverageTree 
              rootNode={rootNode} 
              onNodeClick={handleNodeClick} 
              selectedPath={selectedNode?.path || null}
              reportId={reportInfo?.report_id || ""}
              isIncrement={isIncrement}
            />
          </aside>
        )}

        <section className="flex-grow h-full overflow-hidden flex flex-col relative">
          {/* 工具栏：收起侧边栏和最大化 - 换成缩放图标和面板图标 */}
          <div className="absolute top-3 right-6 z-20 flex space-x-2">
            <button 
              onClick={() => setIsSidebarOpen(!isSidebarOpen)}
              className="p-2 bg-[#161b22]/80 backdrop-blur-sm border border-gray-700/50 rounded-lg hover:bg-gray-700 text-gray-400 transition-all shadow-xl active:scale-95 group"
              title={isSidebarOpen ? "收起侧边栏" : "展开侧边栏"}
            >
              {isSidebarOpen ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
            </button>
            <button 
              onClick={() => setIsMaximized(!isMaximized)}
              className={`p-2 backdrop-blur-sm border rounded-lg transition-all shadow-xl active:scale-95 ${
                isMaximized 
                ? "bg-blue-600 border-blue-500 text-white" 
                : "bg-[#161b22]/80 border-gray-700/50 text-gray-400 hover:bg-gray-700"
              }`}
              title={isMaximized ? "恢复布局" : "全屏代码模式"}
            >
              {isMaximized ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
            </button>
          </div>

          <div className="flex-grow overflow-hidden relative">
            {selectedPath ? (
              fileData ? (
                <SourceView 
                  filePath={selectedPath} 
                  content={fileData.content} 
                  coverage={fileData.lines} 
                  isIncrement={isIncrement}
                />
              ) : (
                <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center bg-[#0d1117]">
                  <div className="animate-pulse flex flex-col items-center">
                    <Layout size={32} className="mb-4 opacity-20" />
                    <p className="text-sm">正在加载 {selectedPath} ...</p>
                  </div>
                </div>
              )
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center bg-[#0d1117]">
                <Search size={48} className="mb-4 opacity-10" />
                <h2 className="text-lg font-light mb-1">选择文件查看源码</h2>
                <p className="max-w-xs text-[11px] text-gray-600 leading-relaxed">
                  在左侧点击文件节点，即可在此处查看行级染色及统计详情。
                  点击目录节点可查看该目录的汇总统计信息。
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
