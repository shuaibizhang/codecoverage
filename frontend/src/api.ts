import { type TreeNode, type ReportInfo, type FileCoverage } from './types';

const BASE_URL = 'http://localhost:8080';

export async function getReportInfo(type: string, module: string, branch: string, commit: string): Promise<ReportInfo> {
  const url = `${BASE_URL}/api/v1/coverage/report?type=${type}&module=${module}&branch=${branch}&commit=${commit}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch report info');
  }
  return response.json();
}

export async function getTreeNodes(reportId: string, path: string): Promise<TreeNode[]> {
  const url = `${BASE_URL}/api/v1/coverage/tree?report_id=${encodeURIComponent(reportId)}&path=${encodeURIComponent(path)}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch tree nodes');
  }
  const data = await response.json();
  return data.nodes || [];
}

export async function getFileCoverage(reportId: string, path: string): Promise<FileCoverage> {
  const url = `${BASE_URL}/api/v1/coverage/file?report_id=${encodeURIComponent(reportId)}&path=${encodeURIComponent(path)}`;
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch file coverage');
  }
  return response.json();
}
