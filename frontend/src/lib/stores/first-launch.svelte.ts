/**
 * First launch state management.
 * Listens to Wails events for template download progress.
 */

import { templateStore } from './templates.svelte';
import { installState } from './install-state.svelte';
import { editor } from './editor.svelte';
import { getExample, convertAndCompile } from '$lib/api/client';

interface TemplateProgress {
  name: string;
  status: 'pending' | 'downloading' | 'success' | 'error';
  error?: string;
}

interface FirstLaunchState {
  isActive: boolean;
  total: number;
  downloaded: number;
  failed: number;
  templates: Map<string, TemplateProgress>;
  errorMessage?: string;
  manualDownloadUrl?: string;
}

let _state = $state<FirstLaunchState>({
  isActive: false,
  total: 0,
  downloaded: 0,
  failed: 0,
  templates: new Map(),
});

// CDN zip download URLs (per user decision)
const CDN_ZIP_BASE = 'https://presto.c-1o.top/templates/packages/presto-template-{name}.zip';

export const firstLaunchStore = {
  get state() { return _state; },

  get isActive() { return _state.isActive; },
  get total() { return _state.total; },
  get downloaded() { return _state.downloaded; },
  get failed() { return _state.failed; },
  get templates() { return _state.templates; },
  get errorMessage() { return _state.errorMessage; },

  getManualDownloadUrl(templateName: string): string {
    return CDN_ZIP_BASE.replace('{name}', templateName);
  },

  async init() {
    console.log('[first-launch] init() called');

    // Check if running in Wails desktop environment
    const rt = (window as any).runtime;
    if (!rt?.EventsOn) {
      console.log('[first-launch] not in Wails environment (no window.runtime.EventsOn)');
      return;
    }

    console.log('[first-launch] Wails runtime detected, registering event listeners');

    rt.EventsOn('first-launch:start', (data: any) => {
      console.log('[first-launch] received start event:', data);
      _state.total = data.total ?? 0;
      _state.downloaded = 0;
      _state.failed = 0;
      _state.templates = new Map();
      _state.isActive = true;
      _state.errorMessage = undefined;
      console.log('[first-launch] started, total:', _state.total);

      // Mark all templates as installing to show breathing animation
      if (data.templates) {
        console.log('[first-launch] marking templates as installing:', data.templates);
        data.templates.forEach((name: string) => {
          installState.setInstalling(name);
        });
      } else {
        console.warn('[first-launch] no templates list in start event');
      }
    });

    rt.EventsOn('first-launch:progress', (data: any) => {
      console.log('[first-launch] received progress event:', data);
      const { name, status, error } = data;
      _state.templates.set(name, { name, status, error });

      if (status === 'success') {
        _state.downloaded++;
        installState.setInstalled(name);
        console.log('[first-launch] template installed:', name);

        // Auto-select and load example for first successfully downloaded template
        // Prefer 'gongwen' if it's among the downloads
        const canSeedEditor = !editor.pendingExternalLoad &&
          !editor.currentFilePath &&
          !editor.markdown.trim();

        if (!editor.selectedTemplate && canSeedEditor) {
          const templates = Array.from(_state.templates.keys());
          const successTemplates = templates.filter((t: string) =>
            _state.templates.get(t)?.status === 'success'
          );

          // Prefer gongwen, otherwise use first successful template
          const templateToSelect = successTemplates.includes('gongwen')
            ? 'gongwen'
            : successTemplates[0];

          if (templateToSelect) {
            editor.selectedTemplate = templateToSelect;
            console.log('[first-launch] auto-selected template:', templateToSelect);

            // Load example document into editor
            getExample(templateToSelect).then(example => {
              if (example) {
                editor.markdown = example;
                console.log('[first-launch] loaded example document for:', templateToSelect);
                // Trigger conversion to show preview
                convertAndCompile(example, templateToSelect).catch(err => {
                  console.error('[first-launch] failed to compile example:', err);
                });
              }
            }).catch(err => {
              console.error('[first-launch] failed to load example:', err);
            });
          }
        }
      } else if (status === 'error') {
        _state.failed++;
        installState.reset(name);
        console.log('[first-launch] template failed:', name, error);
      }
    });

    rt.EventsOn('first-launch:complete', (data: any) => {
      console.log('[first-launch] received complete event:', data);
      const { success, failed } = data;
      _state.isActive = false;
      console.log('[first-launch] complete:', success, 'success,', failed, 'failed');

      // Refresh template list after download completes
      if (success > 0) {
        console.log('[first-launch] refreshing template list');
        templateStore.refresh().catch(err => {
          console.error('[first-launch] failed to refresh templates:', err);
        });
      }

      // If all failed, show manual download option
      if (success === 0 && failed > 0) {
        _state.errorMessage = '所有模板下载失败，请手动下载';
      }
    });

    rt.EventsOn('first-launch:error', (data: any) => {
      console.log('[first-launch] received error event:', data);
      _state.isActive = false;
      _state.errorMessage = data.message;
      console.error('[first-launch] error:', data.message);
    });

    // Listen for template download progress events
    rt.EventsOn('template-download:progress', (data: any) => {
      const { template, downloaded, total, percent } = data;
      installState.updateProgress(template, { downloaded, total, percent });
    });

    console.log('[first-launch] all event listeners registered successfully');

    // Signal Go backend that frontend is ready to receive events
    rt.EventsEmit('frontend:ready');
    console.log('[first-launch] emitted frontend:ready signal');
  },

  reset() {
    _state = {
      isActive: false,
      total: 0,
      downloaded: 0,
      failed: 0,
      templates: new Map(),
    };
  },
};

// Auto-initialize on import (only in Wails desktop environment)
// Check if running in Wails by trying to import the runtime
if (typeof window !== 'undefined') {
  console.log('[first-launch] auto-initializing in browser environment');
  firstLaunchStore.init().catch(err => {
    // Silently fail if not in Wails environment (e.g., showcase builds)
    console.debug('[first-launch] not initializing:', err.message);
  });
} else {
  console.log('[first-launch] not in browser environment, skipping auto-init');
}
