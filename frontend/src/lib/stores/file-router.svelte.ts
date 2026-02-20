/**
 * Unified file processing logic for drag-drop and Open button.
 * Handles ZIP template installation, markdown routing, toast + confirm UI state.
 */
import { importBatchZip } from '$lib/api/client';
import type { BatchImportResult } from '$lib/api/types';
import { templateStore } from '$lib/stores/templates.svelte';
import { editor } from '$lib/stores/editor.svelte';
import { pendingDrop } from '$lib/stores/pending-drop.svelte';
import { goto } from '$app/navigation';

// --- Toast state ---
let _toast = $state<{ message: string; type: 'success' | 'error' } | null>(null);
let _toastTimer: ReturnType<typeof setTimeout>;

// --- Confirm replace dialog state ---
let _confirmVisible = $state(false);
let _confirmResolve: ((v: boolean) => void) | null = null;

// --- Processing state ---
let _processing = $state(false);

function isMarkdown(name: string): boolean {
  const lower = name.toLowerCase();
  return lower.endsWith('.md') || lower.endsWith('.markdown') || lower.endsWith('.txt');
}

function isZip(name: string): boolean {
  return name.toLowerCase().endsWith('.zip');
}

export const fileRouter = {
  // Toast
  get toast() {
    return _toast;
  },
  showToast(message: string, type: 'success' | 'error') {
    clearTimeout(_toastTimer);
    _toast = { message, type };
    _toastTimer = setTimeout(() => {
      _toast = null;
    }, 2500);
  },

  // Confirm replace dialog
  get confirmVisible() {
    return _confirmVisible;
  },
  confirmAccept() {
    _confirmVisible = false;
    _confirmResolve?.(true);
    _confirmResolve = null;
  },
  confirmCancel() {
    _confirmVisible = false;
    _confirmResolve?.(false);
    _confirmResolve = null;
  },

  // Processing indicator
  get processing() {
    return _processing;
  },

  /**
   * Prompt the user to confirm replacing editor content.
   * Returns true immediately if the editor is empty.
   */
  async promptReplace(): Promise<boolean> {
    if (!editor.markdown.trim()) return true;
    _confirmVisible = true;
    return new Promise((resolve) => {
      _confirmResolve = resolve;
    });
  },

  /**
   * Core file processing function.
   * Called by both layout drag-drop and the Open button.
   *
   * @param files - File objects (markdown only when zipResults provided)
   * @param currentPath - Current route path (e.g. '/', '/batch', '/settings')
   * @param documentDirs - Optional map of filename → directory (desktop mode)
   * @param zipResults - Pre-processed ZIP results from Wails binding (desktop mode)
   */
  async processFiles(
    files: File[],
    currentPath: string,
    documentDirs?: Map<string, string>,
    zipResults?: BatchImportResult[],
  ): Promise<void> {
    if (files.length === 0 && (!zipResults || zipResults.length === 0)) return;
    _processing = true;

    try {
      const zipFiles = files.filter((f) => isZip(f.name));
      const mdFiles = files.filter((f) => isMarkdown(f.name));

      // Step 1: Process ZIPs — install templates first, then collect markdown
      const allMarkdown: File[] = [...mdFiles];
      let workDir: string | undefined;
      const importedTemplates: { name: string; displayName: string; status: string }[] = [];
      // Per-file working directories (from ZIP per-file workDir + desktop documentDirs)
      const perFileDirs = new Map<string, string>();

      // Carry over desktop documentDirs for directly dropped files
      if (documentDirs) {
        for (const [name, dir] of documentDirs) {
          if (isMarkdown(name)) perFileDirs.set(name, dir);
        }
      }

      // Use pre-processed ZIP results if provided (Wails binding, bypasses HTTP)
      if (zipResults && zipResults.length > 0) {
        for (const result of zipResults) {
          importedTemplates.push(...result.templates);
          if (result.workDir) workDir = result.workDir;
          for (const md of result.markdownFiles) {
            const blob = new Blob([md.content], { type: 'text/markdown' });
            allMarkdown.push(new File([blob], md.name, { type: 'text/markdown' }));
            const dir = md.workDir || result.workDir;
            if (dir) perFileDirs.set(md.name, dir);
          }
        }
      } else {
        // Browser mode: upload ZIPs via FormData (HTTP)
        for (const zip of zipFiles) {
          try {
            const result = await importBatchZip(zip);

            // Collect imported templates
            importedTemplates.push(...result.templates);

            // Record workDir for image resolution (fallback)
            if (result.workDir) workDir = result.workDir;

            // Create File objects from extracted markdown
            for (const md of result.markdownFiles) {
              const blob = new Blob([md.content], { type: 'text/markdown' });
              allMarkdown.push(new File([blob], md.name, { type: 'text/markdown' }));
              // Per-file workDir (nested ZIP dirs) takes priority over top-level workDir
              const dir = md.workDir || result.workDir;
              if (dir) perFileDirs.set(md.name, dir);
            }
          } catch (err) {
            console.error('ZIP import failed:', err);
            fileRouter.showToast(
              err instanceof Error ? err.message : 'ZIP 导入失败',
              'error',
            );
          }
        }
      }

      // Step 2: Refresh templates if any were imported
      if (importedTemplates.length > 0) {
        await templateStore.refresh();
        const installed = importedTemplates
          .filter((t) => t.status !== 'skipped')
          .map((t) => t.displayName || t.name);
        if (installed.length > 0) {
          fileRouter.showToast(`模板 "${installed.join('、')}" 导入成功`, 'success');
        }
      }

      // Step 3: Route markdown files
      if (allMarkdown.length === 0) {
        // Templates only — toast already shown above
        if (importedTemplates.length === 0) {
          fileRouter.showToast('未找到可导入的文件', 'error');
        }
        return;
      }

      if (currentPath === '/batch') {
        // On batch page: add all to batch
        pendingDrop.set({
          files: allMarkdown,
          workDir,
          documentDirs: perFileDirs.size > 0 ? perFileDirs : undefined,
        });
        return;
      }

      if (allMarkdown.length === 1) {
        // Single file → editor (with confirmation if editor has content)
        const confirmed = await fileRouter.promptReplace();
        if (!confirmed) return;

        const file = allMarkdown[0];
        const content = await file.text();
        const dir = perFileDirs.get(file.name) || workDir || '';
        editor.markdown = content;
        editor.documentDir = dir;
        editor.pendingExternalLoad = true;
        if (currentPath !== '/') await goto('/');
      } else {
        // Multiple files → batch
        pendingDrop.set({
          files: allMarkdown,
          workDir,
          documentDirs: perFileDirs.size > 0 ? perFileDirs : undefined,
        });
        if (currentPath !== '/batch') await goto('/batch');
      }
    } finally {
      _processing = false;
    }
  },
};
