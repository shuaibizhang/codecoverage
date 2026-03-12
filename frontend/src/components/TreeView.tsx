import React, { useState } from 'react';
import { ChevronRight, ChevronDown, Folder, FileCode } from 'lucide-react';
import { type TreeNode, NodeType } from '../types';

interface TreeItemProps {
  node: TreeNode;
  level: number;
  onFileClick: (path: string) => void;
}

const TreeItem: React.FC<TreeItemProps> = ({ node, level, onFileClick }) => {
  const [isOpen, setIsOpen] = useState(false);
  const isDir = node.type === NodeType.Dir;

  const toggleOpen = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isDir) {
      setIsOpen(!isOpen);
    } else {
      onFileClick(node.path);
    }
  };

  const coveragePercent = node.stat.coverage > 0 ? (node.stat.coverage * 100).toFixed(1) : '0.0';

  return (
    <div className="select-none">
      <div 
        className={`flex items-center py-1 px-2 hover:bg-gray-700 cursor-pointer rounded transition-colors`}
        style={{ paddingLeft: `${level * 16}px` }}
        onClick={toggleOpen}
      >
        <span className="mr-1 text-gray-400">
          {isDir ? (
            isOpen ? <ChevronDown size={16} /> : <ChevronRight size={16} />
          ) : (
            <span className="w-4" />
          )}
        </span>
        <span className="mr-2">
          {isDir ? <Folder size={18} className="text-blue-400" /> : <FileCode size={18} className="text-gray-300" />}
        </span>
        <span className="flex-grow text-sm truncate">{node.name}</span>
        <span className="text-xs font-mono ml-4 text-gray-400 w-16 text-right">
          {coveragePercent}%
        </span>
      </div>
      
      {isDir && isOpen && node.children && (
        <div>
          {node.children.map((child, idx) => (
            <TreeItem key={idx} node={child} level={level + 1} onFileClick={onFileClick} />
          ))}
        </div>
      )}
    </div>
  );
};

interface TreeViewProps {
  tree: TreeNode;
  onFileClick: (path: string) => void;
}

export const TreeView: React.FC<TreeViewProps> = ({ tree, onFileClick }) => {
  return (
    <div className="bg-[#1e1e1e] text-gray-200 h-full overflow-y-auto border-r border-gray-700 p-2">
      <div className="text-xs font-bold text-gray-500 uppercase tracking-wider mb-4 px-2">
        Project Explorer
      </div>
      <TreeItem node={tree} level={0} onFileClick={onFileClick} />
    </div>
  );
};
