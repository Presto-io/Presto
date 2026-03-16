/**
 * Install state machine for template installation UI.
 * Manages per-template install state: idle, installing, installed.
 * Error handling is via Toast, not persistent state.
 */

type InstallStatus = 'idle' | 'installing' | 'installed';

interface InstallStateEntry {
  status: InstallStatus;
}

let _states = $state<Map<string, InstallStateEntry>>(new Map());

export const installState = {
  get(templateName: string): InstallStatus {
    return _states.get(templateName)?.status ?? 'idle';
  },

  setInstalling(templateName: string): void {
    _states.set(templateName, { status: 'installing' });
  },

  setInstalled(templateName: string): void {
    _states.set(templateName, { status: 'installed' });
  },

  reset(templateName: string): void {
    // After toast dismisses, reset to idle (not error)
    _states.set(templateName, { status: 'idle' });
  },

  isInstalling(templateName: string): boolean {
    return this.get(templateName) === 'installing';
  },

  isInstalled(templateName: string): boolean {
    return this.get(templateName) === 'installed';
  },
};
