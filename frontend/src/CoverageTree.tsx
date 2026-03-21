import React, { useState, useEffect } from 'react';
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
}

const TreeItem: React.FC<TreeItemProps> = ({ node, level, onNodeClick, selectedPath, reportId, isIncrement }) => {
  const [isOpen, setIsOpen] = useState(level === 0);
  const [children, setChildren] = useState<TreeNode[]>(node.children || []);
  const [loading, setLoading] = useState(false);
  
  const isDir = node.type === NodeType.Dir || node.type === undefined;
  const isSelected = selectedPath === node.path;

  useEffect(() => {
    if (isDir && isOpen && children.length === 0) {
      async function fetchChildren() {
        setLoading(true);
        try {
          const fetchedChildren = await getTreeNodes(reportId, node.path, isIncrement);
          setChildren(fetchedChildren);
        } catch (err) {
          console.error("Failed to fetch children:", err);
        } finally {
          setLoading(false);
        }
      }
      fetchChildren();
    }
  }, [isDir, isOpen, node.path, reportId, isIncrement]);

  useEffect(() => {
    // 当切换增量模式时，重新获取子节点
    if (isDir && isOpen) {
      async function fetchChildren() {
        setLoading(true);
        try {
          const fetchedChildren = await getTreeNodes(reportId, node.path, isIncrement);
          setChildren(fetchedChildren);
        } catch (err) {
          console.error("Failed to fetch children:", err);
        } finally {
          setLoading(false);
        }
      }
      fetchChildren();
    }
  }, [isIncrement]);

  const toggle = (e: React.MouseEvent) => {
    onNodeClick(node);
    if (isDir) {
      setIsOpen(!isOpen);
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
        <span className="flex-grow truncate font-medium">{node.name}</span>
        
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
            children.map((child) => (
              <TreeItem
                key={child.path}
                node={child}
                level={level + 1}
                onNodeClick={onNodeClick}
                selectedPath={selectedPath}
                reportId={reportId}
                isIncrement={isIncrement}
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
}

export const CoverageTree: React.FC<CoverageTreeProps> = ({ rootNode, onNodeClick, selectedPath, reportId, isIncrement }) => {
  if (!rootNode) return null;

  return (
    <div className="flex flex-col h-full bg-[#0d1117]">
      <div className="px-3 py-1.5 border-b border-gray-800 bg-[#161b22] text-[10px] font-bold uppercase tracking-wider text-gray-500 flex items-center">
        <Folder size={12} className="mr-2 text-gray-500" />
        文件目录树
      </div>
      <div className="flex-grow overflow-y-auto custom-scrollbar">
        <TreeItem
          node={rootNode}
          level={0}
          onNodeClick={onNodeClick}
          selectedPath={selectedPath}
          reportId={reportId}
          isIncrement={isIncrement}
        />
      </div>
    </div>
  );
};
