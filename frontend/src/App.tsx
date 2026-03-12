import React, { useState, useEffect } from 'react';
import { CoverageTree } from './CoverageTree';
import { SourceView } from './SourceView';
import { Layout, ShieldCheck, Database, Calendar } from 'lucide-react';
import { getReportInfo, getTreeNodes, getFileCoverage } from './api';
import { NodeType, type ReportInfo, type TreeNode, type FileCoverage } from './types';

function App() {
  const [reportInfo, setReportInfo] = useState<ReportInfo | null>(null);
  const [rootNode, setRootNode] = useState<TreeNode | null>(null);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [fileData, setFileData] = useState<FileCoverage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function init() {
      try {
        setLoading(true);
        // 获取报告信息 (这里暂时硬编码一些参数，实际应该从 URL 或配置获取)
        // 匹配 uint_cover_test.go 中生成的参数: type="unittest", module="github.com/shuaibizhang/codecoverage", branch="main", commit="latest"
        const info = await getReportInfo("unittest", "github.com/shuaibizhang/codecoverage", "main", "latest");
        setReportInfo(info);

        // 获取根节点
        const nodes = await getTreeNodes(info.report_id, "*");
        if (nodes.length > 0) {
          // 包装成一个虚拟根节点，或者直接使用第一个节点（如果后端返回的是单个根节点）
          // 假设后端返回的是根目录下的所有子节点
          const root: TreeNode = {
            name: "root",
            path: "*",
            type: NodeType.Dir,
            stat: {
              total_lines: 0,
              instr_lines: 0,
              cover_lines: 0,
              coverage: 0,
              add_lines: 0,
              delete_lines: 0,
              incr_instr_lines: 0,
              incr_cover_lines: 0,
              incr_coverage: 0
            },
            children: nodes
          };
          setRootNode(root);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    }
    init();
  }, []);

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

  if (loading) {
    return (
      <div className="h-screen w-screen flex items-center justify-center bg-[#0d1117] text-gray-200">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-green-500"></div>
      </div>
    );
  }

  if (error || !reportInfo || !rootNode) {
    return (
      <div className="h-screen w-screen flex flex-col items-center justify-center bg-[#0d1117] text-gray-200 p-8">
        <ShieldCheck className="text-red-500 mb-4" size={48} />
        <h1 className="text-xl font-bold mb-2">加载失败</h1>
        <p className="text-gray-400">{error || "无法获取报告数据，请确保后端服务已启动并已上传数据。"}</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen w-screen bg-[#0d1117] text-gray-200 overflow-hidden">
      {/* Header */}
      <header className="h-16 flex items-center justify-between px-6 border-b border-gray-800 bg-[#161b22]">
        <div className="flex items-center space-x-3">
          <ShieldCheck className="text-green-500" size={28} />
          <div>
            <h1 className="text-lg font-bold tracking-tight">代码覆盖率</h1>
            <div className="text-xs text-gray-400 flex items-center">
              <span className="bg-gray-800 px-1.5 py-0.5 rounded text-gray-300 font-mono">{reportInfo.meta.module}</span>
              <span className="mx-2">/</span>
              <span className="text-gray-500">{reportInfo.meta.branch} @ {reportInfo.meta.commit}</span>
            </div>
          </div>
        </div>
        <div className="flex items-center space-x-6 text-xs text-gray-400">
          <div className="flex items-center">
            <Database size={14} className="mr-1.5" />
            <span>ID: {reportInfo.report_id}</span>
          </div>
          <div className="flex items-center">
            <Calendar size={14} className="mr-1.5" />
            <span>更新时间: {reportInfo.meta.last_update}</span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex flex-grow overflow-hidden">
        {/* Sidebar */}
        <aside className="w-1/3 min-w-[320px] max-w-md h-full flex flex-col">
          <CoverageTree 
            tree={rootNode as any} 
            onFileClick={handleFileClick} 
            selectedPath={selectedPath}
            reportId={reportInfo.report_id}
          />
        </aside>

        {/* Source View */}
        <section className="flex-grow h-full bg-[#0d1117] overflow-hidden">
          {selectedPath ? (
            fileData ? (
              <SourceView 
                filePath={selectedPath} 
                content={fileData.content} 
                coverage={fileData.lines} 
              />
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center">
                <Layout size={48} className="mb-4 opacity-20" />
                <p className="text-lg">加载中...</p>
              </div>
            )
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-gray-500 p-8 text-center bg-[#1e1e1e]/50">
              <Layout size={64} className="mb-6 opacity-10" />
              <h2 className="text-2xl font-light mb-2">选择一个文件查看覆盖率</h2>
              <p className="max-w-md text-sm leading-relaxed">
                从左侧目录结构中选择一个文件，以检查详细的行级覆盖率指标。
              </p>
              
              <div className="mt-12 grid grid-cols-3 gap-8 w-full max-w-lg">
                <div className="flex flex-col items-center">
                  <div className="w-3 h-3 rounded-full bg-green-500 mb-2"></div>
                  <span className="text-xs uppercase tracking-widest">已覆盖</span>
                </div>
                <div className="flex flex-col items-center">
                  <div className="w-3 h-3 rounded-full bg-red-500 mb-2"></div>
                  <span className="text-xs uppercase tracking-widest">未覆盖</span>
                </div>
                <div className="flex flex-col items-center">
                  <div className="w-3 h-3 rounded-full bg-gray-600 mb-2"></div>
                  <span className="text-xs uppercase tracking-widest">非指令行</span>
                </div>
              </div>
            </div>
          )}
        </section>
      </main>
    </div>
  );
}

export default App;
