/**
 * Install state machine for template installation UI.
 * Manages per-template install state: idle, installing, installed.
 * Error handling is via Toast, not persistent state.
 *
 * Uses a $state version counter + plain Map to guarantee Svelte 5 reactivity.
 * ($state<Map> proxy interception is unreliable for .get()/.set() in templates.)
 */

type InstallStatus = 'idle' | 'installing' | 'installed';

export interface DownloadProgress {
  downloaded: number;  // bytes
  total: number;       // bytes
  percent: number;     // 0-100
}

interface InstallStateEntry {
  status: InstallStatus;
  progress?: DownloadProgress;
}

export interface ActiveDownloadEntry {
  name: string;
  progress: DownloadProgress;
}

let _version = $state(0);
const _map = new Map<string, InstallStateEntry>();

export const installState = {
  get version() { return _version; },

  get(templateName: string): InstallStatus {
    void _version;
    return _map.get(templateName)?.status ?? 'idle';
  },

  getProgress(templateName: string): DownloadProgress | undefined {
    void _version;
    return _map.get(templateName)?.progress;
  },

  getActiveDownloads(): ActiveDownloadEntry[] {
    void _version;
    return Array.from(_map.entries()).flatMap(([name, entry]) => {
      if (entry.status !== 'installing' || !entry.progress) {
        return [];
      }
      return [{ name, progress: entry.progress }];
    });
  },

  setInstalling(templateName: string): void {
    _map.set(templateName, { status: 'installing', progress: undefined });
    _version++;
  },

  setInstalled(templateName: string): void {
    _map.set(templateName, { status: 'installed' });
    _version++;
  },

  updateProgress(templateName: string, progress: DownloadProgress): void {
    const entry = _map.get(templateName);
    if (entry && entry.status === 'installing') {
      _map.set(templateName, { ...entry, progress });
      _version++;
    }
  },

  reset(templateName: string): void {
    _map.set(templateName, { status: 'idle' });
    _version++;
  },

  isInstalling(templateName: string): boolean {
    void _version;
    return this.get(templateName) === 'installing';
  },

  isInstalled(templateName: string): boolean {
    void _version;
    return this.get(templateName) === 'installed';
  },
};
