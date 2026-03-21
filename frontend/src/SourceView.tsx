import React from 'react';
import { parseLine } from './types';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { FileCode } from 'lucide-react';

function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

interface SourceViewProps {
  filePath: string;
  content: string;
  coverage: number[];
  isIncrement?: boolean;
}

export const SourceView: React.FC<SourceViewProps> = ({ filePath, content, coverage, isIncrement = false }) => {
  const lines = content.split('\n');
  return (
    <div className="flex flex-col h-full bg-[#0d1117] overflow-hidden">
      <div className="px-3 py-1.5 border-b border-gray-800 bg-[#161b22] text-[11px] text-gray-400 font-mono flex justify-between items-center shrink-0">
        <div className="flex items-center">
          <FileCode size={12} className="mr-2 text-blue-400" />
          <span className="font-bold">{filePath}</span>
        </div>
        {isIncrement && (
          <span className="text-[9px] bg-blue-500/10 text-blue-400 px-1.5 py-0.5 rounded border border-blue-500/20 uppercase font-bold">
            增量视图
          </span>
        )}
      </div>
      <div className="flex-grow overflow-auto font-mono text-[12px] leading-5 custom-scrollbar">
        <table className="w-full border-separate border-spacing-0">
          <tbody className="bg-[#0d1117]">
            {lines.map((line, index) => {
              const { isInstruction, isCovered, count, isIncr } = parseLine(coverage[index] || 0);
              
              let bgColor = "";
              let textColor = "text-gray-400";
              let indicatorColor = "bg-transparent";

              // 在增量模式下，只有属于 diff 的行才被视为有效的统计行
              const shouldHighlight = isIncrement ? (isInstruction && isIncr) : isInstruction;

              if (shouldHighlight) {
                if (isCovered) {
                  bgColor = "bg-green-900/20";
                  textColor = "text-green-400";
                  indicatorColor = "bg-green-500";
                } else {
                  bgColor = "bg-red-900/20";
                  textColor = "text-red-400";
                  indicatorColor = "bg-red-500";
                }
              } else {
                // 如果是增量模式，但这一行不在增量范围内，即使它被覆盖了，也用灰色显示
                textColor = "text-gray-500";
              }

              return (
                <tr key={index} className={cn("hover:bg-white/5", bgColor)}>
                  <td className="w-10 text-right pr-3 text-gray-600 select-none bg-[#0d1117] border-r border-gray-800/50 text-[10px] align-top py-0.5">
                    {index + 1}
                  </td>
                  <td className="w-1 px-0 bg-[#0d1117] align-stretch py-0">
                    <div className={cn("w-0.5 h-full min-h-[20px]", indicatorColor)} />
                  </td>
                  <td className="w-10 text-right px-2 text-gray-700 bg-[#0d1117] border-r border-gray-800/50 italic select-none text-[10px] align-top py-0.5">
                    {shouldHighlight ? count : ""}
                  </td>
                  <td className={cn("pl-3 whitespace-pre break-all", textColor, "py-0.5")}>
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
