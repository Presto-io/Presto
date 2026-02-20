/**
 * Cross-page file transfer store.
 * Used by the layout's universal drop handler to pass files
 * to the batch page (via $effect) or the editor page.
 */

export interface PendingDropData {
  files: File[];
  workDir?: string;
}

let _pending = $state<PendingDropData | null>(null);

export const pendingDrop = {
  get data() {
    return _pending;
  },
  set(data: PendingDropData) {
    _pending = data;
  },
  clear() {
    _pending = null;
  },
};
