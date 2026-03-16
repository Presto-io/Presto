import { listTemplates } from '$lib/api/client';
import type { Template } from '$lib/api/types';

let _templates = $state<Template[]>([]);
let _loading = $state(false);
let _loaded = $state(false);

export const templateStore = {
  get templates() {
    return _templates;
  },
  get loading() {
    return _loading;
  },
  get loaded() {
    return _loaded;
  },

  async load(force = false) {
    if (_loaded && !force) {
      console.log('[template-store] already loaded, skipping (force=', force, ')');
      return;
    }
    console.log('[template-store] loading templates (force=', force, ')');
    _loading = true;
    try {
      _templates = (await listTemplates()) ?? [];
      console.log('[template-store] loaded', _templates.length, 'templates');
    } catch (err) {
      console.error('[template-store] failed to load templates:', err);
    } finally {
      _loading = false;
      _loaded = true;
    }
  },

  async refresh() {
    console.log('[template-store] refresh() called');
    return this.load(true);
  },
};

// Auto-refresh when templates change (ZIP import, etc.)
if (typeof window !== 'undefined') {
  window.addEventListener('templates-changed', () => {
    console.log('[template-store] received templates-changed event, refreshing');
    templateStore.refresh();
  });
}
