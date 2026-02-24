import type { Template, Manifest, GitHubRepo, BatchImportResult, ImportResult, RegistryTemplate, PlatformInfo } from './types';

const BASE = import.meta.env.VITE_API_URL || '';

function getApiKey(): string {
  if (typeof document === 'undefined') return '';
  const meta = document.querySelector('meta[name="api-key"]');
  return meta?.getAttribute('content') || '';
}

function authFetch(url: string, init?: RequestInit): Promise<Response> {
  const key = getApiKey();
  if (key) {
    const headers = new Headers(init?.headers);
    headers.set('Authorization', `Bearer ${key}`);
    return fetch(url, { ...init, headers });
  }
  return fetch(url, init);
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await authFetch(`${BASE}${path}`, init);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function listTemplates(): Promise<Template[]> {
  return api('/api/templates');
}

/** @deprecated 前端已改用 registryStore，不再调用此函数 */
export async function discoverTemplates(): Promise<GitHubRepo[]> {
  return api('/api/templates/discover');
}

export async function getManifest(id: string): Promise<Manifest> {
  return api(`/api/templates/${id}/manifest`);
}

export async function getExample(id: string): Promise<string> {
  const data = await api<{ example: string }>(`/api/templates/${id}/example`);
  return data.example;
}

export async function convert(markdown: string, templateId: string): Promise<string> {
  const res = await authFetch(`${BASE}/api/convert`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ markdown, templateId })
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Convert failed (${res.status}): ${body}`);
  }
  const data = await res.json();
  return data.typst;
}

export async function compile(typstSource: string, workDir?: string): Promise<Blob> {
  const headers: Record<string, string> = { 'Content-Type': 'text/plain' };
  if (workDir) headers['X-Work-Dir'] = workDir;
  const url = workDir
    ? `${BASE}/api/compile?workDir=${encodeURIComponent(workDir)}`
    : `${BASE}/api/compile`;
  const res = await authFetch(url, {
    method: 'POST',
    headers,
    body: typstSource
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Compile failed (${res.status}): ${body}`);
  }
  return res.blob();
}

export async function compileSvg(typstSource: string, workDir?: string): Promise<string[]> {
  const headers: Record<string, string> = { 'Content-Type': 'text/plain' };
  if (workDir) headers['X-Work-Dir'] = workDir;
  const url = workDir
    ? `${BASE}/api/compile-svg?workDir=${encodeURIComponent(workDir)}`
    : `${BASE}/api/compile-svg`;
  const res = await authFetch(url, {
    method: 'POST',
    headers,
    body: typstSource
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`SVG compile failed (${res.status}): ${body}`);
  }
  const data = await res.json();
  return data.pages;
}

export async function convertAndCompile(
  markdown: string,
  templateId: string,
  workDir?: string
): Promise<Blob> {
  const res = await authFetch(`${BASE}/api/convert-and-compile`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ markdown, templateId, workDir })
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Compile failed (${res.status}): ${body}`);
  }
  return res.blob();
}

export async function installTemplate(
  owner: string,
  repo: string,
  platforms?: Record<string, PlatformInfo>
): Promise<void> {
  const res = await authFetch(
    `${BASE}/api/templates/${encodeURIComponent(owner + '/' + repo)}/install`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ owner, repo, platforms })
    }
  );
  if (!res.ok) throw new Error(`Install failed: ${res.status}`);
}

export function installFromRegistry(template: RegistryTemplate): Promise<void> {
  let owner: string;
  let repo: string;

  if (template.repo) {
    // v2 format: "owner/repo"
    const parts = template.repo.split('/');
    owner = parts[0];
    repo = parts[1];
  } else if (template.repository) {
    // v1 format: full URL
    const url = new URL(template.repository);
    const parts = url.pathname.slice(1).split('/');
    owner = parts[0];
    repo = parts[1];
  } else {
    return Promise.reject(new Error('No repository info'));
  }

  return installTemplate(owner, repo, template.platforms);
}

export async function deleteTemplate(id: string): Promise<void> {
  const res = await authFetch(`${BASE}/api/templates/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
}

export async function renameTemplate(id: string, displayName: string): Promise<void> {
  const res = await authFetch(`${BASE}/api/templates/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ displayName }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(body.error || `Rename failed: ${res.status}`);
  }
}

export interface ImportConflictError {
  error: 'conflict';
  conflicts: string[];
}

export async function importTemplateZip(
  file: File,
  onConflict?: 'overwrite' | 'skip' | 'rename',
): Promise<ImportResult[]> {
  const formData = new FormData();
  formData.append('file', file);
  const params = onConflict ? `?onConflict=${onConflict}` : '';
  const res = await authFetch(`${BASE}/api/templates/import${params}`, {
    method: 'POST',
    body: formData,
  });
  if (res.status === 409) {
    const body = await res.json();
    const err = new Error(body.error || 'conflict') as Error & { conflicts?: string[] };
    err.conflicts = body.conflicts;
    throw err;
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(body.error || `Import failed: ${res.status}`);
  }
  return res.json();
}

export async function importBatchZip(file: File): Promise<BatchImportResult> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await authFetch(`${BASE}/api/batch/import-zip`, {
    method: 'POST',
    body: formData,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
    throw new Error(body.error || `Batch import failed: ${res.status}`);
  }
  return res.json();
}
