/**
 * First launch state management.
 * Listens to Wails events for template download progress.
 */

import { templateStore } from './templates.svelte';
import { installState } from './install-state.svelte';
import { editor } from './editor.svelte';
import { notificationStore } from './notification.svelte';
import { getExample, convertAndCompile } from '$lib/api/client';

interface TemplateProgress {
  name: string;
  status: 'pending' | 'downloading' | 'success' | 'error';
  error?: string;
  manualDownloadUrl?: string;
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

function templateNamesFrom(items: any[] | undefined): string[] {
  if (!Array.isArray(items)) return [];
  return items
    .map((item) => typeof item === 'string' ? item : item?.name)
    .filter((name): name is string => typeof name === 'string' && name.length > 0);
}

export const firstLaunchStore = {
  get state() { return _state; },

  get isActive() { return _state.isActive; },
  get total() { return _state.total; },
  get downloaded() { return _state.downloaded; },
  get failed() { return _state.failed; },
  get templates() { return _state.templates; },
  get errorMessage() { return _state.errorMessage; },

  getManualDownloadUrl(templateName: string): string | undefined {
    return _state.templates.get(templateName)?.manualDownloadUrl;
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
      const templateNames = templateNamesFrom(data.templates);
      if (templateNames.length > 0) {
        console.log('[first-launch] marking templates as installing:', templateNames);
        templateNames.forEach((name: string) => {
          installState.setInstalling(name);
        });
      } else {
        console.warn('[first-launch] no templates list in start event');
      }
    });

    rt.EventsOn('first-launch:progress', (data: any) => {
      console.log('[first-launch] received progress event:', data);
      const { name, status, error, manualDownloadUrl } = data;
      _state.templates.set(name, { name, status, error, manualDownloadUrl });

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
                editor.savedContent = example;
                editor.exampleContent = example;
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

    rt.EventsOn('template-update:start', (data: any) => {
      console.log('[template-update] received start event:', data);
      const templateNames = templateNamesFrom(data.templates);
      notificationStore.dismissGroup('template-update');
      notificationStore.show({
        message: `检测到 ${templateNames.length} 个模板需要更新，开始自动更新`,
        type: 'info',
        source: 'template-update',
        groupKey: 'template-update',
        durationMs: 4200,
      });
      templateNames.forEach((name) => {
        _state.templates.set(name, { name, status: 'downloading' });
        installState.setInstalling(name);
      });
    });

    rt.EventsOn('template-update:progress', (data: any) => {
      console.log('[template-update] received progress event:', data);
      const { name, status, error } = data;
      if (!name) return;
      _state.templates.set(name, { name, status, error });
      if (status === 'success') {
        installState.setInstalled(name);
      } else if (status === 'error') {
        installState.reset(name);
      }
    });

    rt.EventsOn('template-update:complete', (data: any) => {
      console.log('[template-update] received complete event:', data);
      const updated = Array.isArray(data.updated) ? data.updated : [];
      if ((data.success ?? 0) > 0) {
        templateStore.refresh().catch(err => {
          console.error('[template-update] failed to refresh templates:', err);
        });
      }
      const success = data.success ?? 0;
      const failed = data.failed ?? 0;
      notificationStore.dismissGroup('template-update');
      if (success === 0 && failed > 0) {
        notificationStore.show({
          message: '模板更新失败，已保留现有模板',
          type: 'warning',
          source: 'template-update',
          groupKey: 'template-update',
          durationMs: 5600,
        });
      } else if (failed > 0) {
        notificationStore.show({
          message: `已更新 ${success} 个模板，${failed} 个失败，失败项已保留旧版本`,
          type: 'warning',
          source: 'template-update',
          groupKey: 'template-update',
          durationMs: 5600,
        });
      } else {
        notificationStore.show({
          message: `模板已更新：${success} 个模板`,
          type: 'success',
          source: 'template-update',
          groupKey: 'template-update',
          durationMs: 3600,
        });
      }
      window.dispatchEvent(new CustomEvent('presto:templates-updated', {
        detail: { updated, success, failed },
      }));
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
