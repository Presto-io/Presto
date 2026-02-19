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
    if (_loaded && !force) return;
    _loading = true;
    try {
      _templates = (await listTemplates()) ?? [];
    } catch {
      /* silent */
    } finally {
      _loading = false;
      _loaded = true;
    }
  },

  async refresh() {
    return this.load(true);
  },
};

// Auto-refresh when templates change (ZIP import, etc.)
if (typeof window !== 'undefined') {
  window.addEventListener('templates-changed', () => {
    templateStore.refresh();
  });
}
