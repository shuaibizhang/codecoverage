export interface MetaInfo {
  module: string;
  branch: string;
  commit: string;
  total_files: number;
  last_update: string;
}

export interface TreeNodeData {
  total_lines: number;
  instr_lines: number;
  cover_lines: number;
  coverage: number;
  add_lines: number;
  delete_lines: number;
  incr_instr_lines: number;
  incr_cover_lines: number;
  incr_coverage: number;
}

export const NodeType = {
  Dir: 0,
  File: 1,
} as const;

export type NodeType = typeof NodeType[keyof typeof NodeType];

export interface TreeNode {
  name: string;
  path: string;
  type: NodeType;
  stat: TreeNodeData;
  children?: TreeNode[];
}

export interface ReportInfo {
  report_id: string;
  meta: MetaInfo;
}

export interface FileCoverage {
  path: string;
  lines: number[]; // 数组，每个元素表示该行的覆盖情况
  content: string; // 源码内容
}

// 覆盖行标识位解析
export const MaskInstrLine = 0x80000000; // 1 << 31 位为是否是指令行的标识位 (1表示非指令行，0表示指令行)
export const MaskIncrLine = 0x40000000;  // 1 << 30 位为是否是增量行的标识位
export const MaskCoverCount = 0x3FFFFFFF; // 低30位为覆盖次数掩码

export function parseLine(val: number) {
  const isInstruction = (val & MaskInstrLine) === 0;
  const isIncr = (val & MaskIncrLine) !== 0;
  const count = val & MaskCoverCount;
  const isCovered = count > 0;
  
  return {
    isInstruction,
    isIncr,
    isCovered,
    count
  };
}
