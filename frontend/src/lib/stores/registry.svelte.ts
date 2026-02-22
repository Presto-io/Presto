import type { Registry } from '$lib/api/types';

const useMock = import.meta.env.DEV || import.meta.env.VITE_MOCK === '1';

const REGISTRY_URL = useMock
  ? '/mock/registry.json'
  : 'https://registry.presto.app/templates/registry.json';

let _registry = $state<Registry | null>(null);
let _loading = $state(false);
let _error = $state<string | null>(null);

export const registryStore = {
  get registry() { return _registry; },
  get loading() { return _loading; },
  get error() { return _error; },

  async load(force = false) {
    if (_registry && !force) return;
    _loading = true;
    _error = null;
    try {
      const res = await fetch(REGISTRY_URL);
      if (!res.ok) throw new Error(`${res.status}`);
      _registry = await res.json();
    } catch (e) {
      _error = e instanceof Error ? e.message : String(e);
    } finally {
      _loading = false;
    }
  },

  async refresh() { return this.load(true); },
};
