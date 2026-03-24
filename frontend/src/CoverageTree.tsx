import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { type TreeNode, NodeType } from './types';
import { ChevronRight, ChevronDown, Folder, FileCode } from 'lucide-react';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { getTreeNodes } from './api';

function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

interface TreeItemProps {
  node: TreeNode;
  level: number;
  onNodeClick: (node: TreeNode) => void;
  selectedPath: string | null;
  reportId: string;
  isIncrement: boolean;
  searchQuery: string;
  matchedPaths?: Set<string>;
  ancestorPaths?: Set<string>;
  searchChildrenMap?: Map<string, TreeNode[]>;
  parentIsMatched?: boolean;
}

const TreeItem: React.FC<TreeItemProps> = ({ 
  node, 
  level, 
  onNodeClick, 
  selectedPath, 
  reportId, 
  isIncrement, 
  searchQuery, 
  matchedPaths, 
  ancestorPaths, 
  searchChildrenMap,
  parentIsMatched = false
}) => {
  const isSearchActive = searchQuery.trim().length > 0;
  const isMatched = matchedPaths?.has(node.path);
  const isAncestor = ancestorPaths?.has(node.path);
  
  const [isOpen, setIsOpen] = useState(level === 0 || (isSearchActive && isAncestor));
  const isDir = node.type === NodeType.Dir || node.type === undefined;
  const isSelected = selectedPath === node.path;

  // 可见性逻辑：
  // 1. 非搜索模式：始终可见
  // 2. 搜索模式下：
  //    - 节点本身匹配 (isMatched)
  //    - 节点是匹配项的祖先 (isAncestor)
  //    - 特殊情况：如果用户手动展开了一个匹配的目录，我们允许看到它的直接子节点 (即使不匹配)
  //      但为了解决您说的“展示了不匹配节点”，我们将 parentIsMatched 限制为仅在目录已手动展开时才生效
  const isVisible = !isSearchActive || isMatched || isAncestor || (parentIsMatched && isOpen);

  // 状态同步逻辑：
  // 核心优化：避免在搜索切换时频繁 setChildren 导致闪烁
  // null 表示尚未获取，[] 表示已获取但内容为空
  const [children, setChildren] = useState<TreeNode[] | null>(node.children && node.children.length > 0 ? node.children : null);
  const [loading, setLoading] = useState(false);

  // 当 node 改变时（例如由于 App 重新获取报告），同步更新本地 children 状态
  // 这能解决“闪烁”以及“旧数据残留”问题
  useEffect(() => {
    setChildren(node.children && node.children.length > 0 ? node.children : null);
  }, [node.path, node.children]);

  // 计算当前应该显示的子节点，避免通过 useEffect 异步设置 state 导致的闪烁
  const displayChildren = useMemo(() => {
    if (isSearchActive && searchChildrenMap) {
      return searchChildrenMap.get(node.path) || [];
    }
    // 非搜索模式下，如果 children state 有值（来自懒加载），优先使用，否则用 node.children
    return children !== null ? children : (node.children || []);
  }, [isSearchActive, searchChildrenMap, node.path, node.children, children]);

  // 异步获取子节点 - 使用 ref 来追踪 children 避免循环依赖
  const childrenRef = React.useRef(children);
  childrenRef.current = children;
  const loadingRef = React.useRef(false);

  const fetchChildren = useCallback(async (force = false) => {
    if (loadingRef.current || (!force && childrenRef.current !== null)) return;
    loadingRef.current = true;
    setLoading(true);
    try {
      const fetchedChildren = await getTreeNodes(reportId, node.path, isIncrement);
      setChildren(fetchedChildren || []);
    } catch (err) {
      console.error("Failed to fetch children:", err);
      setChildren([]);
    } finally {
      loadingRef.current = false;
      setLoading(false);
    }
  }, [reportId, node.path, isIncrement]);

  // 当搜索状态改变时，如果是祖先节点，则自动展开
  useEffect(() => {
    if (isSearchActive && isAncestor) {
      setIsOpen(true);
    }
  }, [isSearchActive, isAncestor]);

  // 核心优化：禁止搜索模式下的任何自动异步请求 (Waterfall)
  // 只有在非搜索模式下，展开目录才自动触发获取
  useEffect(() => {
    const shouldAutoFetch = !isSearchActive && isDir && isOpen && children === null;
    if (shouldAutoFetch) {
      fetchChildren();
    }
  }, [isDir, isOpen, children, isSearchActive, fetchChildren]);

  const prevIsIncrementRef = React.useRef(isIncrement);
  useEffect(() => {
    if (prevIsIncrementRef.current !== isIncrement) {
      prevIsIncrementRef.current = isIncrement;
      // 当且仅当增量模式切换时，如果目录已展开且不在搜索模式，重新获取子节点
      if (isDir && isOpen && !isSearchActive) {
        fetchChildren(true);
      }
    }
  }, [isIncrement, isDir, isOpen, isSearchActive, fetchChildren]);

  if (!isVisible) return null;

  const toggle = (e: React.MouseEvent) => {
    onNodeClick(node);
    if (isDir) {
      const nextOpen = !isOpen;
      setIsOpen(nextOpen);
      // 如果手动点击展开，且没有子节点，则触发加载
      if (nextOpen && children === null) {
        fetchChildren();
      }
    }
    e.stopPropagation();
  };

  const getCoverageColor = (coverage: number) => {
    if (coverage >= 80) return 'text-green-500';
    if (coverage >= 50) return 'text-yellow-500';
    return 'text-red-500';
  };

  const coverage = isIncrement ? (node.stat?.incr_coverage || 0) : (node.stat?.coverage || 0);
  const coverLines = isIncrement ? (node.stat?.incr_cover_lines || 0) : (node.stat?.cover_lines || 0);
  const instrLines = isIncrement ? (node.stat?.incr_instr_lines || 0) : (node.stat?.instr_lines || 0);

  return (
    <div className="select-none group">
      <div
        className={cn(
          "flex items-center py-0.5 px-2 hover:bg-gray-800/50 cursor-pointer text-[11px] border-l-2 transition-colors",
          isSelected ? "bg-blue-600/10 border-blue-500 text-blue-400" : "border-transparent text-gray-400 hover:text-gray-200"
        )}
        style={{ paddingLeft: `${level * 12 + 4}px` }}
        onClick={toggle}
      >
        <span className="w-4 flex items-center justify-center mr-0.5 text-gray-500">
          {isDir ? (
            isOpen ? <ChevronDown size={12} /> : <ChevronRight size={12} />
          ) : null}
        </span>
        <span className="mr-1.5 shrink-0">
          {isDir ? (
            <Folder size={14} className={cn("transition-colors", isOpen ? "text-blue-400" : "text-gray-500 group-hover:text-blue-400")} />
          ) : (
            <FileCode size={14} className={cn("transition-colors", isSelected ? "text-blue-400" : "text-gray-500 group-hover:text-gray-300")} />
          )}
        </span>
        <span className={cn("flex-grow truncate font-medium", isMatched && !isSelected && "text-blue-300")}>
          {node.name}
        </span>
        
        <div className="flex items-center space-x-2 opacity-60 group-hover:opacity-100 transition-opacity ml-2">
          <span className={cn("font-mono font-bold text-[10px]", getCoverageColor(coverage))}>
            {coverage.toFixed(0)}%
          </span>
          <span className="text-[9px] text-gray-600 w-16 text-right font-mono hidden sm:inline">
            {coverLines}/{instrLines}
          </span>
        </div>
      </div>
      {isDir && isOpen && (
        <div className="border-l border-gray-800/30 ml-[10px]">
          {loading ? (
            <div className="py-1 text-[9px] text-gray-600 italic" style={{ paddingLeft: `${(level + 1) * 12 + 16}px` }}>
              加载中...
            </div>
          ) : (
            displayChildren.map((child) => (
              <TreeItem
                key={child.path}
                node={child}
                level={level + 1}
                onNodeClick={onNodeClick}
                selectedPath={selectedPath}
                reportId={reportId}
                isIncrement={isIncrement}
                searchQuery={searchQuery}
                matchedPaths={matchedPaths}
                ancestorPaths={ancestorPaths}
                searchChildrenMap={searchChildrenMap}
                parentIsMatched={isMatched} // 将当前匹配状态传给子节点
              />
            ))
          )}
        </div>
      )}
    </div>
  );
};

interface CoverageTreeProps {
  rootNode: TreeNode | null;
  onNodeClick: (node: TreeNode) => void;
  selectedPath: string | null;
  reportId: string;
  isIncrement: boolean;
  searchQuery: string;
  matchedPaths?: Set<string>;
  ancestorPaths?: Set<string>;
  searchChildrenMap?: Map<string, TreeNode[]>;
}

export const CoverageTree: React.FC<CoverageTreeProps> = React.memo(({ 
  rootNode, 
  onNodeClick, 
  selectedPath, 
  reportId, 
  isIncrement,
  searchQuery,
  matchedPaths,
  ancestorPaths,
  searchChildrenMap
}) => {
  if (!rootNode) return null;

  return (
    <div className="flex flex-col h-full bg-[#0d1117]">
      {/* 搜索模式下的提示 */}
      {searchQuery && (
        <div className="px-4 py-2 border-b border-gray-800/50 bg-[#161b22]/50">
          <div className="text-[11px] font-medium text-blue-400 flex items-center gap-2">
            <div className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse" />
            搜索模式已开启: {searchQuery}
          </div>
        </div>
      )}

      <div className="flex-grow overflow-y-auto custom-scrollbar">
        <TreeItem
          node={rootNode}
          level={0}
          onNodeClick={onNodeClick}
          selectedPath={selectedPath}
          reportId={reportId}
          isIncrement={isIncrement}
          searchQuery={searchQuery}
          matchedPaths={matchedPaths}
          ancestorPaths={ancestorPaths}
          searchChildrenMap={searchChildrenMap}
          parentIsMatched={false}
        />
      </div>
    </div>
  );
});
