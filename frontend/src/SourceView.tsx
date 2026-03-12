import React from 'react';
import { parseLine } from './types';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

interface SourceViewProps {
  filePath: string;
  content: string; // 修改为 string
  coverage: number[];
}

export const SourceView: React.FC<SourceViewProps> = ({ filePath, content, coverage }) => {
  const lines = content.split('\n'); // 分割行
  return (
    <div className="flex flex-col h-full bg-[#1e1e1e] overflow-hidden">
      <div className="p-3 border-b border-gray-800 bg-[#252526] text-sm text-gray-300 font-mono">
        {filePath}
      </div>
      <div className="flex-grow overflow-auto font-mono text-sm leading-6">
        <table className="w-full border-collapse">
          <tbody>
            {lines.map((line, index) => {
              const { isInstruction, isCovered, count } = parseLine(coverage[index] || 0);
              
              let bgColor = "";
              let textColor = "text-gray-400"; // Default non-instruction
              let indicatorColor = "bg-transparent";

              if (isInstruction) {
                if (isCovered) {
                  bgColor = "bg-green-900/20";
                  textColor = "text-green-400"; // 覆盖为绿色
                  indicatorColor = "bg-green-500";
                } else {
                  bgColor = "bg-red-900/20";
                  textColor = "text-red-400"; // 未覆盖为红色
                  indicatorColor = "bg-red-500";
                }
              } else {
                textColor = "text-gray-500"; // 非指令行（其余）为灰色
              }

              return (
                <tr key={index} className={cn("hover:bg-white/5", bgColor)}>
                  <td className="w-12 text-right pr-4 text-gray-600 select-none bg-[#1e1e1e] border-r border-gray-800">
                    {index + 1}
                  </td>
                  <td className="w-4 px-0 bg-[#1e1e1e]">
                    <div className={cn("w-1 h-full", indicatorColor)} />
                  </td>
                  <td className="w-12 text-right px-2 text-gray-700 bg-[#1e1e1e] border-r border-gray-800 italic select-none">
                    {isInstruction ? count : ""}
                  </td>
                  <td className={cn("pl-4 whitespace-pre", textColor)}>
                    {line}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
