/**
 * Install state machine for template installation UI.
 * Manages per-template install state: idle, installing, installed.
 * Error handling is via Toast, not persistent state.
 */

type InstallStatus = 'idle' | 'installing' | 'installed';

interface DownloadProgress {
  downloaded: number;  // bytes
  total: number;       // bytes
  percent: number;     // 0-100
}

interface InstallStateEntry {
  status: InstallStatus;
  progress?: DownloadProgress;  // undefined when not downloading
}

let _states = $state<Map<string, InstallStateEntry>>(new Map());

export const installState = {
  get(templateName: string): InstallStatus {
    return _states.get(templateName)?.status ?? 'idle';
  },

  getProgress(templateName: string): DownloadProgress | undefined {
    return _states.get(templateName)?.progress;
  },

  setInstalling(templateName: string): void {
    _states.set(templateName, { status: 'installing', progress: undefined });
  },

  setInstalled(templateName: string): void {
    _states.set(templateName, { status: 'installed' });  // Remove progress on completion
  },

  updateProgress(templateName: string, progress: DownloadProgress): void {
    const entry = _states.get(templateName);
    if (entry && entry.status === 'installing') {
      _states.set(templateName, { ...entry, progress });
    }
  },

  reset(templateName: string): void {
    // After toast dismisses, reset to idle (not error)
    _states.set(templateName, { status: 'idle' });  // Clear progress
  },

  isInstalling(templateName: string): boolean {
    return this.get(templateName) === 'installing';
  },

  isInstalled(templateName: string): boolean {
    return this.get(templateName) === 'installed';
  },
};
