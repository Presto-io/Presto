/**
 * Network state management.
 * Uses navigator.onLine API for immediate offline detection.
 */

interface NetworkState {
  isOnline: boolean;
  lastOnlineTime?: Date;
}

let _state = $state<NetworkState>({
  isOnline: typeof navigator !== 'undefined' ? navigator.onLine : true,
});

export const networkStore = {
  get isOnline() { return _state.isOnline; },
  get lastOnlineTime() { return _state.lastOnlineTime; },

  init() {
    if (typeof window === 'undefined') return;

    console.log('[network] initializing, current status:', navigator.onLine);
    _state.isOnline = navigator.onLine;

    window.addEventListener('online', () => {
      console.log('[network] online event received');
      _state.isOnline = true;
      _state.lastOnlineTime = new Date();
    });

    window.addEventListener('offline', () => {
      console.log('[network] offline event received');
      _state.isOnline = false;
    });
  },

  // Manual refresh (for testing)
  refresh() {
    if (typeof navigator !== 'undefined') {
      _state.isOnline = navigator.onLine;
    }
  },
};

// Auto-initialize in browser environment
if (typeof window !== 'undefined') {
  console.log('[network] auto-initializing');
  networkStore.init();
}
