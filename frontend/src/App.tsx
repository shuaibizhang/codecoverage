import React, { useState, useEffect, useCallback } from 'react';
import { CoverageTree } from './CoverageTree';
import { SourceView } from './SourceView';
import { MergeModal } from './MergeModal';
import { Layout, ShieldCheck, Database, Activity, Search, RefreshCw, Layers, ChevronDown, Maximize2, Minimize2, PanelLeftClose, PanelLeftOpen, GitMerge } from 'lucide-react';
import { getReportInfo, getFileCoverage, getMetadataList, getRootCoverage, searchNodes } from './api';
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
  const [_error, setError] = useState<string | null>(null);
  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [isMaximized, setIsMaximized] = useState(false);
  const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);

  const [searchQuery, setSearchQuery] = useState("");
  const [matchedPaths, setMatchedPaths] = useState<Set<string> | undefined>(undefined);
  const [ancestorPaths, setAncestorPaths] = useState<Set<string> | undefined>(undefined);
  const [searchChildrenMap, setSearchChildrenMap] = useState<Map<string, TreeNode[]> | undefined>(undefined);
  const [isSearching, setIsSearching] = useState(false);

  // 搜索防抖
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchQuery) {
        performSearch(searchQuery);
      } else {
        setMatchedPaths(undefined);
        setAncestorPaths(undefined);
        setSearchChildrenMap(undefined);
      }
    }, 300); // 300ms 防抖

    return () => clearTimeout(timer);
  }, [searchQuery, isIncrement]);

  const performSearch = useCallback(async (query: string) => {
    if (!query.trim() || !reportInfo || !rootNode) return;

    setIsSearching(true);
    try {
      const results = await searchNodes(reportInfo.report_id, query, isIncrement);

      const matched = new Set<string>();
      const ancestors = new Set<string>();
      const childrenMap = new Map<string, TreeNode[]>();

      // 确保根节点始终作为祖先被展开
      ancestors.add(rootNode.path);

      results.forEach(node => {
        matched.add(node.path);

        if (node.path === rootNode.path) return;

        let parentPath = "";
        const lastSlashIndex = node.path.lastIndexOf("/");
        if (lastSlashIndex === -1) {
            // 没有斜杠，说明它是根节点的直接子节点
            parentPath = rootNode.path;
        } else {
            // 有斜杠，截取最后一个斜杠前面的部分作为父节点路径
            parentPath = node.path.substring(0, lastSlashIndex);
        }

        if (!childrenMap.has(parentPath)) {
            childrenMap.set(parentPath, []);
        }
        
        // 避免重复添加
        if (!childrenMap.get(parentPath)!.some(c => c.path === node.path)) {
            childrenMap.get(parentPath)!.push(node);
        }
        
        ancestors.add(parentPath);
      });

      setMatchedPaths(matched);
      setAncestorPaths(ancestors);
      setSearchChildrenMap(childrenMap);
    } catch (err) {
      console.error("Search failed:", err);
    } finally {
      setIsSearching(false);
    }
  }, [isIncrement, reportInfo, rootNode]);

  const handleSearch = (query: string) => {
    setSearchQuery(query);
  };

  // 1. 当测试类型改变时，获取模块列表
  useEffect(() => {
    let active = true;
    const fetchModules = async () => {
      try {
        setLoading(true);
        const data = await getMetadataList(testType);
        if (!active) return;
        setMetadata(prev => ({ ...prev, modules: data.modules }));
        if (data.modules.length > 0) {
          if (!module || !data.modules.includes(module)) {
            setModule(data.modules[0]);
          }
        } else {
          setModule("");
          setBranch("");
          setCommit("");
          setMetadata({ modules: [], branches: [], commits: [] });
        }
      } catch (err) {
        console.error("Failed to fetch modules:", err);
        setError("获取模块列表失败");
      } finally {
        if (active) setLoading(false);
      }
    };
    fetchModules();
    return () => { active = false; };
  }, [testType]);

  // 2. 当模块改变时，刷新该模块下的分支列表
  useEffect(() => {
    if (!module) return;
    let active = true;
    const fetchBranches = async () => {
      try {
        setLoading(true);
        const data = await getMetadataList(testType, module);
        if (!active) return;
        setMetadata(prev => ({ ...prev, branches: data.branches }));
        if (data.branches.length > 0) {
          if (!branch || !data.branches.includes(branch)) {
            setBranch(data.branches[0]);
          }
        } else {
          setBranch("");
          setCommit("");
          setMetadata(prev => ({ ...prev, branches: [], commits: [] }));
        }
      } catch (err) {
        console.error("Failed to fetch branches:", err);
        setError("获取分支列表失败");
      } finally {
        if (active) setLoading(false);
      }
    };
    fetchBranches();
    return () => { active = false; };
  }, [testType, module]);

  // 3. 当分支改变时，刷新该分支下的提交列表
  useEffect(() => {
    if (!module || !branch) return;
    let active = true;
    const fetchCommits = async () => {
      try {
        setLoading(true);
        const data = await getMetadataList(testType, module, branch);
        if (!active) return;
        setMetadata(prev => ({ ...prev, commits: data.commits }));
        if (data.commits.length > 0) {
          if (!commit || !data.commits.includes(commit)) {
            setCommit(data.commits[0]);
          }
        } else {
          setCommit("");
          setMetadata(prev => ({ ...prev, commits: [] }));
        }
      } catch (err) {
        console.error("Failed to fetch commits:", err);
        setError("获取提交列表失败");
      } finally {
        if (active) setLoading(false);
      }
    };
    fetchCommits();
    return () => { active = false; };
  }, [testType, module, branch]);

  const handleNodeClick = useCallback(async (node: TreeNode) => {
    setSelectedNode(node);
    if (node.type === NodeType.File) {
      setSelectedPath(node.path);
      try {
        setLoading(true);
        const data = await getFileCoverage(reportInfo?.report_id || "", node.path);
        setFileData(data);
      } catch (err) {
        setError("获取文件详情失败");
      } finally {
        setLoading(false);
      }
    } else {
      setSelectedPath(null);
      setFileData(null);
    }
  }, [reportInfo?.report_id]);

  const selectedNodeRef = React.useRef(selectedNode);
  const selectedPathRef = React.useRef(selectedPath);
  selectedNodeRef.current = selectedNode;
  selectedPathRef.current = selectedPath;

  const fetchReportRef = React.useRef<string>("");
  const fetchReport = useCallback(async () => {
    const reportKey = `${testType}-${module}-${branch}-${commit}`;
    if (reportKey === fetchReportRef.current) return;
    if (!module || !branch || !commit) return;

    fetchReportRef.current = reportKey;
    try {
      setLoading(true);
      setError(null);
      const info = await getReportInfo(testType, module, branch, commit);
      setReportInfo(info);

      const root = await getRootCoverage(info.report_id);
      if (root) {
        setRootNode(root);
        if (!selectedNodeRef.current || (selectedPathRef.current && !selectedPathRef.current.startsWith(root.path))) {
          setSelectedNode(root);
          setSelectedPath(root.path);
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
  }, [testType, module, branch, commit]);

  useEffect(() => {
    if (module && branch && commit) {
      fetchReport();
    }
  }, [module, branch, commit, fetchReport]);

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
            <button 
              onClick={() => setIsMergeModalOpen(true)}
              disabled={!module || !branch || !commit}
              className={`bg-[#161b22] border border-gray-700/50 p-2 rounded-lg transition-all shadow-md active:scale-95 flex items-center space-x-2 ${
                !module || !branch || !commit 
                ? 'opacity-50 cursor-not-allowed' 
                : 'hover:bg-blue-600 hover:border-blue-500 text-gray-400 hover:text-white'
              }`}
              title="合并报告"
            >
              <GitMerge size={14} />
              <span className="text-[10px] font-black uppercase tracking-widest hidden lg:block">合并报告</span>
            </button>
          </div>
        </div>
      </header>

      <MergeModal 
        isOpen={isMergeModalOpen}
        onClose={() => setIsMergeModalOpen(false)}
        baseReport={{ module, branch, commit, type: testType }}
        testTypes={TEST_TYPES}
        onMergeSuccess={() => {
          fetchReport();
        }}
      />

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
                    {displayNode.name === "root" ? "root" : displayNode.name}
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
            {/* 搜索框 */}
            <div className="p-3 border-b border-gray-800 bg-[#161b22]">
              <div className="relative group">
                <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 group-focus-within:text-blue-400 transition-colors" />
                <input 
                  type="text"
                  placeholder="搜索文件或目录..."
                  value={searchQuery}
                  onChange={(e) => handleSearch(e.target.value)}
                  className="w-full bg-[#0d1117] border border-gray-700 rounded-md py-1.5 pl-9 pr-3 text-xs text-gray-300 focus:outline-none focus:border-blue-500 focus:ring-1 focus:ring-blue-500/30 transition-all"
                />
                {isSearching && (
                  <RefreshCw size={12} className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 animate-spin" />
                )}
              </div>
            </div>

            <CoverageTree
                rootNode={rootNode}
                onNodeClick={handleNodeClick}
                selectedPath={selectedPath}
                reportId={reportInfo?.report_id || ""}
                isIncrement={isIncrement}
                searchQuery={searchQuery}
                matchedPaths={matchedPaths}
                ancestorPaths={ancestorPaths}
                searchChildrenMap={searchChildrenMap}
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
