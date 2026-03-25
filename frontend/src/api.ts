import { type TreeNode, type ReportInfo, type FileCoverage, type MergeReportsRequest, type MergeReportsResponse } from './types';

const BASE_URL = 'http://localhost:28080';

export async function getReportInfo(type: string, module: string, branch: string, commit: string): Promise<ReportInfo> {
  const url = `${BASE_URL}/api/v1/coverage/report`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ type, module, branch, commit }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch report info');
  }
  return response.json();
}

export async function getMetadataList(type: string, module?: string, branch?: string): Promise<{ modules: string[], branches: string[], commits: string[] }> {
  let url = `${BASE_URL}/api/v1/coverage/metadata?type=${type}`;
  if (module) {
    url += `&module=${encodeURIComponent(module)}`;
  }
  if (branch) {
    url += `&branch=${encodeURIComponent(branch)}`;
  }
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error('Failed to fetch metadata list');
  }
  return response.json();
}

export async function getTreeNodes(reportId: string, path: string, isIncrement: boolean = false): Promise<TreeNode[]> {
  const url = `${BASE_URL}/api/v1/coverage/tree`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId, path, is_increment: isIncrement }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch tree nodes');
  }
  const data = await response.json();
  return data.nodes || [];
}

export async function getFileCoverage(reportId: string, path: string): Promise<FileCoverage> {
  const url = `${BASE_URL}/api/v1/coverage/file`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId, path }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch file coverage');
  }
  return response.json();
}

export async function getRootCoverage(reportId: string): Promise<TreeNode> {
  const url = `${BASE_URL}/api/v1/coverage/root`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch root coverage');
  }
  const data = await response.json();
  return data.root_node;
}

export async function searchNodes(reportId: string, keyword: string, isIncrement: boolean = false): Promise<TreeNode[]> {
  const url = `${BASE_URL}/api/v1/coverage/search`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId, keyword, is_increment: isIncrement }),
  });
  if (!response.ok) {
    throw new Error('Failed to search nodes');
  }
  const data = await response.json();
  return data.nodes || [];
}

export async function mergeReports(req: MergeReportsRequest): Promise<MergeReportsResponse> {
  const url = `${BASE_URL}/api/v1/coverage/merge`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(req),
  });
  if (!response.ok) {
    throw new Error('Failed to merge reports');
  }
  return response.json();
}

export async function createSnapshot(reportId: string): Promise<{ snapshot_report_id: string, success: boolean, message: string }> {
  const url = `${BASE_URL}/api/v1/coverage/snapshot`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId }),
  });
  if (!response.ok) {
    throw new Error('Failed to create snapshot');
  }
  return response.json();
}

export async function getReportInfoById(reportId: string): Promise<ReportInfo> {
  const url = `${BASE_URL}/api/v1/coverage/report_by_id`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch report info by id');
  }
  return response.json();
}
