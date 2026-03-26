import { 
  type TreeNode, 
  type ReportInfo, 
  type FileCoverage, 
  type MergeReportsRequest, 
  type MergeReportsResponse,
  type PullRequest,
  type Commit,
  type RebaseReportResponse
} from './types';

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
  const url = `${BASE_URL}/api/v1/coverage/metadata`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ type, module, branch }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch metadata list');
  }
  return response.json();
}

export async function getTreeNodes(reportId: string, path: string, isIncrement: boolean = false, baseCommit?: string): Promise<TreeNode[]> {
  const url = `${BASE_URL}/api/v1/coverage/tree`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ 
      report_id: reportId, 
      path, 
      is_increment: isIncrement,
      base_commit: baseCommit
    }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch tree nodes');
  }
  const data = await response.json();
  return data.nodes || [];
}

export async function getFileCoverage(reportId: string, path: string, baseCommit?: string): Promise<FileCoverage> {
  const url = `${BASE_URL}/api/v1/coverage/file`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ 
      report_id: reportId, 
      path,
      base_commit: baseCommit
    }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch file coverage');
  }
  return response.json();
}

export async function getRootCoverage(reportId: string, baseCommit?: string): Promise<TreeNode> {
  const url = `${BASE_URL}/api/v1/coverage/root`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ 
      report_id: reportId,
      base_commit: baseCommit
    }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch root coverage');
  }
  const data = await response.json();
  return data.root_node;
}

export async function searchNodes(reportId: string, keyword: string, isIncrement: boolean = false, baseCommit?: string): Promise<TreeNode[]> {
  const url = `${BASE_URL}/api/v1/coverage/search`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ 
      report_id: reportId, 
      keyword, 
      is_increment: isIncrement,
      base_commit: baseCommit
    }),
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

export async function listPullRequests(module: string, state: string = 'open'): Promise<PullRequest[]> {
  const url = `${BASE_URL}/api/v1/git/pulls`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ module, state }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch pull requests');
  }
  const data = await response.json();
  return data.pull_requests || [];
}

export async function listCommits(module: string, branch: string): Promise<Commit[]> {
  const url = `${BASE_URL}/api/v1/git/commits`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ module, branch }),
  });
  if (!response.ok) {
    throw new Error('Failed to fetch commits');
  }
  const data = await response.json();
  return data.commits || [];
}

export async function rebaseReport(reportId: string, baseCommit: string): Promise<RebaseReportResponse> {
  const url = `${BASE_URL}/api/v1/coverage/rebase`;
  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ report_id: reportId, base_commit: baseCommit }),
  });
  if (!response.ok) {
    throw new Error('Failed to rebase report');
  }
  return response.json();
}

