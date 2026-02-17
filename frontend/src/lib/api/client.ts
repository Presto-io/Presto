import type { Template, Manifest, GitHubRepo } from './types';

const BASE = import.meta.env.VITE_API_URL || '';

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) throw new Error(`API error: ${res.status}`);
  return res.json();
}

export async function listTemplates(): Promise<Template[]> {
  return api('/api/templates');
}

export async function discoverTemplates(): Promise<GitHubRepo[]> {
  return api('/api/templates/discover');
}

export async function getManifest(id: string): Promise<Manifest> {
  return api(`/api/templates/${id}/manifest`);
}

export async function convert(markdown: string, templateId: string): Promise<string> {
  const res = await fetch(`${BASE}/api/convert`, {
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

export async function compile(typstSource: string): Promise<Blob> {
  const res = await fetch(`${BASE}/api/compile`, {
    method: 'POST',
    headers: { 'Content-Type': 'text/plain' },
    body: typstSource
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Compile failed (${res.status}): ${body}`);
  }
  return res.blob();
}

export async function convertAndCompile(
  markdown: string,
  templateId: string
): Promise<Blob> {
  const res = await fetch(`${BASE}/api/convert-and-compile`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ markdown, templateId })
  });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Compile failed (${res.status}): ${body}`);
  }
  return res.blob();
}

export async function installTemplate(owner: string, repo: string): Promise<void> {
  const res = await fetch(
    `${BASE}/api/templates/${encodeURIComponent(owner + '/' + repo)}/install`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ owner, repo })
    }
  );
  if (!res.ok) throw new Error(`Install failed: ${res.status}`);
}

export async function deleteTemplate(id: string): Promise<void> {
  const res = await fetch(`${BASE}/api/templates/${id}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
}
