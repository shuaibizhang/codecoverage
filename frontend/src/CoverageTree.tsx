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
  onFileClick: (path: string) => void;
  selectedPath: string | null;
  reportId: string;
}

const TreeItem: React.FC<TreeItemProps> = ({ node, level, onFileClick, selectedPath, reportId }) => {
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
          const fetchedChildren = await getTreeNodes(reportId, node.path);
          setChildren(fetchedChildren);
        } catch (err) {
          console.error("Failed to fetch children:", err);
        } finally {
          setLoading(false);
        }
      }
      fetchChildren();
    }
  }, [isDir, isOpen, node.path, reportId]);

  const toggle = (e: React.MouseEvent) => {
    if (isDir) {
      setIsOpen(!isOpen);
    } else {
      onFileClick(node.path);
    }
    e.stopPropagation();
  };

  const getCoverageColor = (coverage: number) => {
    if (coverage >= 80) return 'text-green-500';
    if (coverage >= 50) return 'text-yellow-500';
    return 'text-red-500';
  };

  return (
    <div className="select-none">
      <div
        className={cn(
          "flex items-center py-1 px-2 hover:bg-gray-800 cursor-pointer text-sm border-l-2",
          isSelected ? "bg-gray-700 border-blue-500" : "border-transparent"
        )}
        style={{ paddingLeft: `${level * 16 + 8}px` }}
        onClick={toggle}
      >
        <span className="w-5 flex items-center justify-center mr-1 text-gray-400">
          {isDir ? (
            isOpen ? <ChevronDown size={14} /> : <ChevronRight size={14} />
          ) : null}
        </span>
        <span className="mr-2">
          {isDir ? (
            <Folder size={16} className="text-blue-400" />
          ) : (
            <FileCode size={16} className="text-gray-300" />
          )}
        </span>
        <span className="flex-grow truncate text-gray-200">{node.name}</span>
        <span className={cn("ml-4 font-mono font-medium", getCoverageColor(node.stat?.coverage || 0))}>
          {(node.stat?.coverage || 0).toFixed(1)}%
        </span>
        <span className="ml-4 text-xs text-gray-500 w-24 text-right">
          {node.stat?.cover_lines || 0}/{node.stat?.instr_lines || 0}
        </span>
      </div>
      {isDir && isOpen && (
        <div>
          {loading ? (
            <div className="py-1 text-xs text-gray-500 italic" style={{ paddingLeft: `${(level + 1) * 16 + 24}px` }}>
              加载中...
            </div>
          ) : (
            children.map((child) => (
              <TreeItem
                key={child.path}
                node={child}
                level={level + 1}
                onFileClick={onFileClick}
                selectedPath={selectedPath}
                reportId={reportId}
              />
            ))
          )}
        </div>
      )}
    </div>
  );
};

export const CoverageTree: React.FC<{
  tree: TreeNode;
  onFileClick: (path: string) => void;
  selectedPath: string | null;
  reportId: string;
}> = ({ tree, onFileClick, selectedPath, reportId }) => {
  return (
    <div className="bg-[#1e1e1e] h-full overflow-y-auto border-r border-gray-800">
      <div className="p-3 text-xs font-bold text-gray-500 uppercase tracking-wider border-b border-gray-800">
        目录结构
      </div>
      <TreeItem 
        node={tree} 
        level={0} 
        onFileClick={onFileClick} 
        selectedPath={selectedPath} 
        reportId={reportId}
      />
    </div>
  );
};
