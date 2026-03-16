/**
 * First launch state management.
 * Listens to Wails events for template download progress.
 */

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
    // Dynamic import to avoid bundling @wailsio/runtime in showcase builds
    const { Events } = await import('@wailsio/runtime');

    Events.On('first-launch:start', (data: any) => {
      _state.total = data.total ?? 0;
      _state.downloaded = 0;
      _state.failed = 0;
      _state.templates = new Map();
      _state.isActive = true;
      _state.errorMessage = undefined;
      console.log('[first-launch] started, total:', _state.total);
    });

    Events.On('first-launch:progress', (data: any) => {
      const { name, status, error } = data;
      _state.templates.set(name, { name, status, error });

      if (status === 'success') {
        _state.downloaded++;
      } else if (status === 'error') {
        _state.failed++;
      }
      console.log('[first-launch] progress:', name, status);
    });

    Events.On('first-launch:complete', (data: any) => {
      const { success, failed } = data;
      _state.isActive = false;
      console.log('[first-launch] complete:', success, 'success,', failed, 'failed');

      // If all failed, show manual download option
      if (success === 0 && failed > 0) {
        _state.errorMessage = '所有模板下载失败，请手动下载';
      }
    });

    Events.On('first-launch:error', (data: any) => {
      _state.isActive = false;
      _state.errorMessage = data.message;
      console.error('[first-launch] error:', data.message);
    });
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
if (typeof window !== 'undefined' && (window as any).runtime?.EventsOn) {
  firstLaunchStore.init();
}
