import { useState, useEffect, useCallback, useRef } from 'react';
import { CoverageTree } from './CoverageTree';
import { SourceView } from './SourceView';
import { MergeModal } from './MergeModal';
import { Layout, ShieldCheck, Database, Search, RefreshCw, Layers, Maximize2, Minimize2, PanelLeftClose, PanelLeftOpen, GitMerge, Camera, CheckCircle2, X, GitCommit, Clock } from 'lucide-react';
import { getReportInfo, getFileCoverage, getMetadataList, getRootCoverage, searchNodes, createSnapshot, getReportInfoById, listCommits, rebaseReport } from './api';
import { NodeType, type ReportInfo, type TreeNode, type FileCoverage, type Commit } from './types';

const TEST_TYPES = [
  { id: 'unittest', label: '单元测试', icon: ShieldCheck },
  { id: 'systest', label: '集成测试', icon: Layers },
  { id: 'snapshot', label: '覆盖率快照', icon: Database },
];

function App() {
  const [testType, setTestType] = useState(TEST_TYPES[0].id);
  const [isIncrement, setIsIncrement] = useState(false);
  const [module, setModule] = useState("");
  const [branch, setBranch] = useState("");
  const [commit, setCommit] = useState("");
  
  const [baseCommit, setBaseCommit] = useState("");
  const [isModifyingBase, setIsModifyingBase] = useState(false);
  const [commitsForBranch, setCommitsForBranch] = useState<Commit[]>([]);
  const [isRebasing, setIsRebasing] = useState(false);
  const [rebaseSuccess, setRebaseSuccess] = useState<string | null>(null);

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
  const [isMergeModalOpen, setIsMergeModalOpen] = useState(false);

  const [searchQuery, setSearchQuery] = useState("");
  const [matchedPaths, setMatchedPaths] = useState<Set<string> | undefined>(undefined);
  const [ancestorPaths, setAncestorPaths] = useState<Set<string> | undefined>(undefined);
  const [searchChildrenMap, setSearchChildrenMap] = useState<Map<string, TreeNode[]> | undefined>(undefined);
  const [isSearching, setIsSearching] = useState(false);

  const [snapshotResult, setSnapshotResult] = useState<{ id: string, message: string } | null>(null);

  const performSearch = useCallback(async (query: string) => {
    if (!query.trim() || !reportInfo || !rootNode) return;

    setIsSearching(true);
    try {
      const results = await searchNodes(reportInfo.report_id, query, isIncrement, baseCommit);

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
  }, [isIncrement, reportInfo, rootNode, baseCommit]);

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
  }, [searchQuery, isIncrement, baseCommit, performSearch]);

  // 当开启修改 baseCommit 模式时，加载当前分支的提交列表
  useEffect(() => {
    if (isModifyingBase && module && branch) {
      const fetchCommits = async () => {
        try {
          const commits = await listCommits(module, branch);
          setCommitsForBranch(commits);
        } catch (err) {
          console.error("Failed to fetch commits for branch:", err);
          setError("获取分支提交列表失败");
        }
      };
      fetchCommits();
    }
  }, [isModifyingBase, module, branch]);

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

  // 4. 当进入修改模式时，获取当前分支的所有提交列表
  useEffect(() => {
    if (!module || !branch || !isModifyingBase) {
      setCommitsForBranch([]);
      return;
    }

    const fetchCommitsForBranch = async () => {
      try {
        setLoading(true);
        const commits = await listCommits(module, branch);
        setCommitsForBranch(commits);
      } catch (err) {
        console.error("Failed to fetch commits for branch:", err);
        setError("获取提交列表失败");
      } finally {
        setLoading(false);
      }
    };
    fetchCommitsForBranch();
  }, [module, branch, isModifyingBase]);

  const handleNodeClick = useCallback(async (node: TreeNode) => {
    setSelectedNode(node);
    if (node.type === NodeType.File) {
      setSelectedPath(node.path);
      try {
        setLoading(true);
        const data = await getFileCoverage(reportInfo?.report_id || "", node.path, baseCommit);
        setFileData(data);
      } catch {
        setError("获取文件详情失败");
      } finally {
        setLoading(false);
      }
    } else {
      setSelectedPath(null);
      setFileData(null);
    }
  }, [reportInfo?.report_id, baseCommit]);

  const selectedNodeRef = useRef(selectedNode);
  const selectedPathRef = useRef(selectedPath);
  selectedNodeRef.current = selectedNode;
  selectedPathRef.current = selectedPath;

  // 当开启修改 baseCommit 时，加载当前分支的历史提交
  useEffect(() => {
    const fetchCommits = async () => {
      if (isModifyingBase && module && branch) {
        try {
          const commits = await listCommits(module, branch);
          setCommitsForBranch(commits);
        } catch (err) {
          console.error("Failed to fetch commits:", err);
          setError("获取分支提交历史失败");
        }
      }
    };
    fetchCommits();
  }, [isModifyingBase, module, branch]);

  const fetchReportRef = useRef<string>("");
  const fetchReport = useCallback(async () => {
    const reportKey = `${testType}-${module}-${branch}-${commit}-${baseCommit}-${isIncrement}`;
    if (reportKey === fetchReportRef.current) return;
    if (!module || !branch || !commit) return;

    fetchReportRef.current = reportKey;
    try {
      setLoading(true);
      setError(null);
      const info = await getReportInfo(testType, module, branch, commit);
      setReportInfo(info);

      const root = await getRootCoverage(info.report_id, baseCommit);
      if (root) {
        setRootNode(root);
        // 只要刷新报告、切换 Tab 或切换增量/全量视图，都重置选中的节点为根节点，展示根节点的覆盖率概览数据
        setSelectedNode(root);
        setSelectedPath(null);
        setFileData(null);
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
  }, [testType, module, branch, commit, baseCommit, isIncrement]);

  useEffect(() => {
    if (module && branch && commit) {
      fetchReport();
    }
  }, [module, branch, commit, baseCommit, isIncrement, fetchReport]);

  const handleRebase = async () => {
    if (!reportInfo || !baseCommit) return;
    if (!window.confirm("确定要重写报告吗？这将会根据当前的 baseCommit 重新计算并保存增量覆盖率数据。")) return;

    try {
      setIsRebasing(true);
      setLoading(true);
      const res = await rebaseReport(reportInfo.report_id, baseCommit);
      if (res.success) {
        setRebaseSuccess("报告增量数据重写成功！已更新当前报告。");
        setIsModifyingBase(false);
        setBaseCommit("");
        // 强制刷新
        fetchReportRef.current = "";
        fetchReport();
        
        // 3秒后自动清除成功提示
        setTimeout(() => setRebaseSuccess(null), 5000);
      } else {
        setError(`重写报告失败: ${res.message}`);
      }
    } catch (err) {
      setError(`重写报告出错: ${err instanceof Error ? err.message : '未知错误'}`);
    } finally {
      setIsRebasing(false);
      setLoading(false);
    }
  };

  const StatItem = ({ label, value, subLabel, colorClass = "text-gray-300", large = false }: { label: string, value: string, subLabel?: string, colorClass?: string, large?: boolean }) => (
    <div 
      className={`flex flex-col border-r border-gray-800 px-6 last:border-r-0 ${large ? 'min-w-[140px]' : 'min-w-[90px]'}`}
      title={`${label}: ${value}${subLabel ? ` (${subLabel})` : ''}`}
    >
      <div className="text-[10px] text-gray-500 uppercase tracking-wider mb-1 font-bold">{label}</div>
      <div className="flex flex-col">
        <span className={`${large ? 'text-2xl' : 'text-xl'} font-bold font-mono ${colorClass}`}>{value}</span>
        {subLabel && <span className="text-[11px] text-gray-500 font-mono mt-0.5">{subLabel}</span>}
      </div>
    </div>
  );

  const loadReportById = async (reportId: string, targetTestType?: string) => {
    try {
      setLoading(true);
      setError(null);
      const info = await getReportInfoById(reportId);
      setReportInfo(info);

      // 更新状态，确保与加载的报告一致
      if (info.meta) {
        setModule(info.meta.module || "");
        setBranch(info.meta.branch || "");
        setCommit(info.meta.commit || "");
        // 更新 fetchReportRef，防止后续 fetchReport 重复请求
        const tType = targetTestType || testType;
        fetchReportRef.current = `${tType}-${info.meta.module || ""}-${info.meta.branch || ""}-${info.meta.commit || ""}`;
      }
      
      const root = await getRootCoverage(info.report_id);
      if (root) {
        setRootNode(root);
        setSelectedNode(root);
        setSelectedPath(null);
        setFileData(null);
      } else {
        setError("未找到该报告的树节点数据。");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载报告失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSnapshot = async () => {
    if (!reportInfo) return;
    try {
      setLoading(true);
      setSnapshotResult(null);
      const res = await createSnapshot(reportInfo.report_id);
      if (res.success) {
        setSnapshotResult({ id: res.snapshot_report_id, message: "快照创建成功！" });
      } else {
        setError(`快照创建失败: ${res.message}`);
      }
    } catch (err) {
      setError(`创建快照出错: ${err instanceof Error ? err.message : '未知错误'}`);
    } finally {
      setLoading(false);
    }
  };

  const displayNode = selectedNode || rootNode;

  return (
    <div className="flex flex-col h-screen w-screen bg-[#0d1117] text-gray-200 overflow-hidden">
      <header className="flex items-center h-14 px-6 border-b border-gray-800 bg-[#0d1117] shrink-0 justify-between z-10">
        <div className="flex items-center space-x-8">
          <div className="flex items-center space-x-3 cursor-default">
            <ShieldCheck className="text-green-500" size={20} />
            <h1 className="text-sm font-bold tracking-tight text-gray-100 uppercase">代码覆盖率 <span className="text-gray-500 font-normal">系统</span></h1>
          </div>

          <div className="flex bg-[#161b22] rounded-md p-1 border border-gray-800">
            {TEST_TYPES.map(type => (
              <button
                key={type.id}
                onClick={() => setTestType(type.id)}
                className={`flex items-center space-x-2 px-4 py-1.5 rounded-md transition-all text-xs font-medium ${
                  testType === type.id 
                  ? "bg-green-600 text-white shadow-sm" 
                  : "text-gray-400 hover:text-gray-200 hover:bg-gray-800"
                }`}
              >
                <type.icon size={14} />
                <span>{type.label}</span>
              </button>
            ))}
          </div>
        </div>

        <div className="flex items-center space-x-3">
          <div className="flex items-center space-x-1 bg-[#161b22] border border-gray-800 rounded-md px-1 py-1">
            {[
              { label: '模块', value: module, onChange: setModule, options: metadata.modules, width: 'w-36' },
              { label: '分支', value: branch, onChange: setBranch, options: metadata.branches, width: 'w-24' },
              { label: '提交', value: commit, onChange: setCommit, options: metadata.commits, width: 'w-24' }
            ].map((item, idx) => (
              <div key={item.label} className={`flex items-center px-2 py-1 ${idx !== 2 ? 'border-r border-gray-800' : ''}`}>
                <span className="text-[10px] text-gray-500 mr-2 uppercase font-bold">{item.label}</span>
                <select 
                  className={`bg-transparent border-none outline-none text-xs ${item.width} text-gray-300 font-medium cursor-pointer appearance-none pr-4`} 
                  value={item.value} 
                  onChange={e => item.onChange(e.target.value)}
                >
                  {item.options.map(o => <option key={o} value={o} className="bg-[#161b22]">{o}</option>)}
                </select>
              </div>
            ))}
          </div>

          <div className="flex items-center space-x-1">
            <button 
              onClick={() => {
                fetchReportRef.current = "";
                fetchReport();
              }}
              className="p-2.5 text-gray-400 hover:text-green-500 hover:bg-green-500/10 rounded-md transition-all"
              title="刷新数据"
            >
              <RefreshCw size={16} className={loading ? "animate-spin" : ""} />
            </button>
            
            <button 
              onClick={() => setIsMergeModalOpen(true)}
              disabled={!module || !branch || !commit}
              className={`flex items-center space-x-2 px-4 py-2 rounded-md text-xs font-bold uppercase tracking-tight transition-all ${
                !module || !branch || !commit 
                ? 'opacity-30 cursor-not-allowed text-gray-500' 
                : 'text-blue-400 hover:bg-blue-500/10'
              }`}
            >
              <GitMerge size={16} />
              <span className="hidden lg:block">合并</span>
            </button>

            <button 
              onClick={handleCreateSnapshot}
              disabled={!reportInfo}
              className={`flex items-center space-x-2 px-4 py-2 rounded-md text-xs font-bold uppercase tracking-tight transition-all ${
                !reportInfo 
                ? 'opacity-30 cursor-not-allowed text-gray-500' 
                : 'text-purple-400 hover:bg-purple-500/10'
              }`}
            >
              <Camera size={16} />
              <span className="hidden lg:block">快照</span>
            </button>
          </div>
        </div>
      </header>

      {/* 主体内容 */}
      <div className="flex-grow flex flex-col min-h-0 overflow-hidden relative">
        {/* 顶部报告状态 & 实时增量配置 */}
        {!isMaximized && (
          <div className="bg-[#161b22] border-b border-gray-800 shrink-0">
            {/* 上部：报告元数据与 BaseCommit 配置 */}
            <div className="px-6 py-3 flex items-center justify-between border-b border-gray-800 bg-[#0d1117]/50">
              <div className="flex items-center space-x-6">
                <div className="flex items-center space-x-3 bg-[#161b22] px-3 py-1.5 rounded-md border border-gray-800">
                  <GitCommit size={14} className="text-gray-500" />
                  <div className="flex items-center space-x-2">
                    <span className="text-[10px] text-gray-500 uppercase font-bold tracking-tight">基准版本</span>
                    <span className="text-xs font-mono font-bold text-gray-300" title={reportInfo?.meta.base_commit}>
                      {reportInfo?.meta.base_commit ? reportInfo.meta.base_commit.substring(0, 7) : "未设置"}
                    </span>
                  </div>
                  
                  {!isModifyingBase ? (
                    reportInfo && (
                      <button
                        onClick={() => setIsModifyingBase(true)}
                        className="ml-2 text-[10px] font-bold text-blue-400 hover:text-blue-300 transition-colors uppercase tracking-tight"
                      >
                        修改
                      </button>
                    )
                  ) : (
                    <div className="flex items-center ml-2 pl-2 border-l border-gray-800 space-x-2">
                      <select 
                        className="bg-transparent border-none outline-none text-xs text-gray-200 font-mono font-bold cursor-pointer pr-4" 
                        value={baseCommit} 
                        onChange={e => {
                          setBaseCommit(e.target.value);
                          setIsIncrement(true);
                        }}
                      >
                        <option value="" className="bg-[#161b22]">-- 选择提交 --</option>
                        {commitsForBranch.map(c => (
                          <option key={c.sha} value={c.sha} className="bg-[#161b22]">
                            {c.sha.substring(0, 7)} - {c.message.substring(0, 30)}
                          </option>
                        ))}
                      </select>
                      <button
                        onClick={() => {
                          setIsModifyingBase(false);
                          setBaseCommit("");
                        }}
                        className="text-gray-500 hover:text-red-400"
                      >
                        <X size={14} />
                      </button>
                    </div>
                  )}
                </div>

                {baseCommit && baseCommit !== reportInfo?.meta.base_commit && (
                  <div className="flex items-center space-x-3 animate-in fade-in slide-in-from-left-2">
                    <div className="flex items-center px-2 py-1 bg-orange-500/10 text-orange-500 rounded text-[10px] font-bold uppercase tracking-tight border border-orange-500/20">
                      预览模式
                    </div>
                    <button
                      onClick={handleRebase}
                      disabled={isRebasing}
                      className="px-3 py-1 bg-orange-600 hover:bg-orange-500 text-white rounded text-[10px] font-bold uppercase tracking-tight transition-colors disabled:opacity-50"
                    >
                      {isRebasing ? '正在重写...' : '确认重写报告'}
                    </button>
                  </div>
                )}

                <div className="h-4 w-px bg-gray-800 mx-2"></div>

                <div className="flex items-center space-x-2 text-[10px] text-gray-500 font-bold uppercase tracking-tight">
                  <Clock size={12} />
                  <span>最后更新: {reportInfo?.meta.last_update || "-"}</span>
                </div>
              </div>

              <div className="flex bg-[#161b22] p-1 rounded-md border border-gray-800">
                <button
                  onClick={() => setIsIncrement(false)}
                  className={`px-4 py-1 rounded text-[10px] font-bold transition-all ${
                    !isIncrement ? "bg-gray-700 text-white shadow-sm" : "text-gray-500 hover:text-gray-300"
                  }`}
                >
                  全量
                </button>
                <button
                  onClick={() => setIsIncrement(true)}
                  className={`px-4 py-1 rounded text-[10px] font-bold transition-all ${
                    isIncrement ? "bg-blue-600 text-white shadow-sm" : "text-gray-500 hover:text-gray-300"
                  }`}
                >
                  增量
                </button>
              </div>
            </div>

            {/* 下部：核心统计指标 */}
            <div className="px-6 py-5 flex items-center justify-between min-h-[100px]">
              {loading && baseCommit && baseCommit !== reportInfo?.meta.base_commit ? (
                <div className="flex-grow flex items-center justify-center space-x-4 animate-pulse">
                  <RefreshCw size={24} className="text-blue-500 animate-spin" />
                  <div className="flex flex-col">
                    <span className="text-sm font-black text-blue-400 uppercase tracking-[0.2em]">正在计算实时增量数据...</span>
                    <span className="text-[9px] text-gray-500 font-bold uppercase tracking-widest mt-1">对比基准: {baseCommit.substring(0, 7)}</span>
                  </div>
                </div>
              ) : displayNode ? (
                <div className="flex items-center flex-grow">
                  <div className="flex flex-col pr-8 mr-8 border-r border-gray-800/50 shrink-0 min-w-[200px]">
                    <div className="text-[9px] text-gray-500 uppercase font-black mb-1 tracking-[0.15em] flex items-center opacity-70">
                      <Layers size={10} className="mr-1.5" />
                      {displayNode.name === "root" ? "统计范围" : "当前选中节点"}
                    </div>
                    <div className={`text-xl font-black truncate max-w-[280px] tracking-tight ${isIncrement ? 'text-blue-400' : 'text-green-500'}`} title={displayNode.path}>
                      {displayNode.name === "root" ? "root" : displayNode.name}
                    </div>
                    <div className="text-[9px] text-gray-500 mt-1.5 font-mono truncate max-w-[280px] bg-[#0d1117] px-2 py-0.5 rounded border border-gray-800/50 w-fit">{displayNode.path}</div>
                  </div>

                  <div className="flex items-center flex-grow overflow-x-auto no-scrollbar space-x-2">
                    <StatItem 
                      label="全量覆盖率" 
                      value={`${displayNode.stat.coverage}%`} 
                      subLabel={`${displayNode.stat.cover_lines} / ${displayNode.stat.instr_lines}`}
                      colorClass="text-green-500"
                      large={!isIncrement}
                    />
                    <StatItem 
                      label="增量覆盖率" 
                      value={`${displayNode.stat.incr_coverage}%`} 
                      subLabel={`${displayNode.stat.incr_cover_lines} / ${displayNode.stat.incr_instr_lines}`}
                      colorClass="text-blue-400"
                      large={isIncrement}
                    />
                    
                    <div className="w-px h-10 bg-gray-800 mx-6 opacity-50"></div>
                    
                    <div className="grid grid-cols-4 gap-x-4 gap-y-1">
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
              ) : (
                <div className="h-16 w-full flex items-center justify-center text-gray-500 space-x-3">
                  <RefreshCw size={20} className="animate-spin opacity-30" />
                  <span className="text-sm font-black tracking-[0.3em] uppercase italic opacity-30">报告数据分析中...</span>
                </div>
              )}
            </div>
          </div>
        )}

        {/* 错误提示和快照结果展示保持不变，但移动到概览面板下方 */}
        <div className="flex flex-col">
          {error && (
            <div className="mx-5 mt-2 bg-red-900/20 border border-red-500/30 rounded-lg p-3 flex items-center justify-between animate-in fade-in slide-in-from-top-2">
              <div className="flex items-center space-x-3">
                <X className="text-red-500" size={18} />
                <p className="text-xs font-bold text-red-500">{error}</p>
              </div>
              <button onClick={() => setError(null)} className="text-gray-500 hover:text-gray-300">
                <X size={16} />
              </button>
            </div>
          )}

          {rebaseSuccess && (
            <div className="mx-5 mt-2 bg-blue-900/20 border border-blue-500/30 rounded-lg p-3 flex items-center justify-between animate-in fade-in slide-in-from-top-2">
              <div className="flex items-center space-x-3">
                <CheckCircle2 className="text-blue-500" size={18} />
                <p className="text-xs font-bold text-blue-400">{rebaseSuccess}</p>
              </div>
              <button onClick={() => setRebaseSuccess(null)} className="text-gray-500 hover:text-gray-300">
                <X size={16} />
              </button>
            </div>
          )}

          {snapshotResult && (
            <div className="mx-5 mt-2 bg-green-900/20 border border-green-500/30 rounded-lg p-3 flex items-center justify-between animate-in fade-in slide-in-from-top-2">
              <div className="flex items-center space-x-3">
                <CheckCircle2 className="text-green-500" size={18} />
                <div>
                  <p className="text-xs font-bold text-green-500">{snapshotResult.message}</p>
                  <p className="text-[10px] text-gray-400 font-mono mt-0.5">PartitionKey: {snapshotResult.id}</p>
                </div>
              </div>
              <div className="flex items-center space-x-2">
                <button 
                  onClick={() => {
                    navigator.clipboard.writeText(snapshotResult.id);
                    setSnapshotResult(prev => prev ? { ...prev, message: "ID已复制到剪贴板！" } : null);
                  }}
                  className="text-[10px] bg-green-600/20 hover:bg-green-600/40 text-green-400 px-2 py-1 rounded border border-green-500/30 transition-colors font-bold"
                >
                  复制 ID
                </button>
                <button 
                  onClick={() => {
                    setTestType('snapshot'); // 跳转到快照覆盖率 tab
                    loadReportById(snapshotResult.id, 'snapshot');
                    setSnapshotResult(null);
                  }}
                  className="text-[10px] bg-blue-600/20 hover:bg-blue-600/40 text-blue-400 px-2 py-1 rounded border border-blue-500/30 transition-colors font-bold"
                >
                  查看
                </button>
                <button onClick={() => setSnapshotResult(null)} className="text-gray-500 hover:text-gray-300">
                  <X size={16} />
                </button>
              </div>
            </div>
          )}
        </div>

        <MergeModal 
          isOpen={isMergeModalOpen}
          onClose={() => setIsMergeModalOpen(false)}
          baseReport={{ module, branch, commit, type: testType }}
          testTypes={TEST_TYPES}
          onMergeSuccess={() => {
            fetchReport();
          }}
        />

      <main className="flex flex-grow overflow-hidden bg-[#0d1117] relative">
        {loading && baseCommit && baseCommit !== reportInfo?.meta.base_commit && (
          <div className="absolute inset-0 z-50 bg-[#0d1117]/80 flex flex-col items-center justify-center space-y-4">
            <RefreshCw size={32} className="text-blue-500 animate-spin" />
            <div className="flex flex-col items-center">
              <h3 className="text-sm font-bold text-white uppercase tracking-widest">正在重新计算增量覆盖率</h3>
              <p className="text-[10px] text-gray-500 font-mono mt-1">Comparing with {baseCommit.substring(0, 7)}...</p>
            </div>
          </div>
        )}
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
                  baseCommit={baseCommit}
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
  </div>
);
}

export default App;
