import { getOutputInfo } from '$lib/api/client';
import type { OutputInfo } from '$lib/api/types';
import { editor } from '$lib/stores/editor.svelte';
import { extractFrontmatterField } from '$lib/utils/frontmatter';

const genericBaseNames = new Set(['output', 'untitled', '未命名']);

export function outputInfoCacheKey(markdown: string, templateId: string): string {
  return `${templateId}\n${markdown}`;
}

export function isGenericBaseName(value: string | null | undefined): boolean {
  const normalized = value?.trim().toLowerCase();
  return !normalized || genericBaseNames.has(normalized);
}

export function cleanFilenameBase(value: string | null | undefined, fallback = 'presto-document'): string {
  const cleaned = (value ?? '')
    .replace(/[<>:"/\\|?*\u0000-\u001F]+/g, '_')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/^[.\s_]+|[.\s_]+$/g, '');

  if (!cleaned || isGenericBaseName(cleaned)) return fallback;
  return cleaned;
}

function cleanCandidate(value: string | null | undefined): string {
  const cleaned = cleanFilenameBase(value, '');
  return cleaned && !isGenericBaseName(cleaned) ? cleaned : '';
}

function stripExtension(filename: string): string {
  const dot = filename.lastIndexOf('.');
  return dot > 0 ? filename.slice(0, dot) : filename;
}

function currentFileBaseName(): string {
  const filename = editor.currentFilePath?.split(/[/\\]/).pop() || '';
  return stripExtension(filename);
}

function markdownHeadingTitle(markdown: string): string {
  for (const line of markdown.split(/\r?\n/)) {
    const match = line.match(/^#\s+(.+)$/);
    if (match?.[1]) return match[1].trim();
  }
  return '';
}

function markdownTitle(markdown: string): string {
  return extractFrontmatterField(markdown, 'title') || markdownHeadingTitle(markdown);
}

function firstMeaningfulBase(candidates: Array<string | null | undefined>, fallback = 'presto-document'): string {
  for (const candidate of candidates) {
    const cleaned = cleanCandidate(candidate);
    if (cleaned) return cleaned;
  }
  return fallback;
}

export async function refreshOutputInfo(
  markdown: string,
  templateId: string,
  shouldApply: () => boolean = () => true,
): Promise<void> {
  const info = typeof window !== 'undefined' && window.go?.main?.App?.GetOutputInfo
    ? await window.go.main.App.GetOutputInfo(markdown, templateId)
    : await getOutputInfo(markdown, templateId);

  if (!shouldApply()) return;
  editor.outputInfo = info;
  editor.outputInfoCacheKey = outputInfoCacheKey(markdown, templateId);
  editor.documentTitle = info.previewTitle || info.document?.title || '';
}

export async function ensureCurrentOutputInfo(): Promise<void> {
  if (!editor.selectedTemplate || !editor.markdown.trim()) return;
  const key = outputInfoCacheKey(editor.markdown, editor.selectedTemplate);
  if (editor.outputInfo && editor.outputInfoCacheKey === key) return;
  await refreshOutputInfo(editor.markdown, editor.selectedTemplate);
}

export function currentDocumentBaseName(): string {
  const key = outputInfoCacheKey(editor.markdown, editor.selectedTemplate);
  const info: OutputInfo | null = editor.outputInfoCacheKey === key ? editor.outputInfo : null;
  return firstMeaningfulBase([
    info?.outputBaseName,
    info?.previewTitle,
    info?.document?.title,
    currentFileBaseName(),
    markdownTitle(editor.markdown),
  ]);
}

export function currentDocumentDisplayTitle(): string {
  const key = outputInfoCacheKey(editor.markdown, editor.selectedTemplate);
  const info: OutputInfo | null = editor.outputInfoCacheKey === key ? editor.outputInfo : null;
  return firstMeaningfulBase([
    info?.previewTitle,
    info?.document?.title,
    info?.outputBaseName,
    currentFileBaseName(),
    markdownTitle(editor.markdown),
  ], '');
}

export async function outputBaseNameForCurrentDocument(): Promise<string> {
  await ensureCurrentOutputInfo();
  return currentDocumentBaseName();
}

export async function markdownDefaultFilenameForCurrentDocument(): Promise<string> {
  await ensureCurrentOutputInfo();
  return `${currentDocumentBaseName()}.md`;
}
