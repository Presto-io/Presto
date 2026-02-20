/**
 * Wizard / onboarding state store.
 * Persisted to localStorage under key 'presto-wizard'.
 * Uses Svelte 5 module-level $state rune (same pattern as editor.svelte.ts).
 */

export interface WizardPointState {
  seen: boolean;
  dismissed: boolean;
}

interface WizardPersisted {
  points: Record<string, WizardPointState>;
  disabled: boolean;
  firstVisit: number;
}

const STORAGE_KEY = 'presto-wizard';

function loadState(): WizardPersisted {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw) return JSON.parse(raw);
  } catch {}
  return { points: {}, disabled: false, firstVisit: Date.now() };
}

function saveState() {
  try {
    const data: WizardPersisted = {
      points: wizard.points,
      disabled: wizard.disabled,
      firstVisit: wizard.firstVisit,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch {}
}

const initial = loadState();

export const wizard = $state({
  points: initial.points as Record<string, WizardPointState>,
  disabled: initial.disabled,
  firstVisit: initial.firstVisit,
});

/** Currently active hint ID (only one at a time) */
export const activeHint = $state<{ id: string | null }>({ id: null });

/** The nudge (pre-hint breathing dot) — which point is in nudge mode */
export const activeNudge = $state<{ id: string | null }>({ id: null });

// --- Public API ---

export function shouldShowPoint(id: string): boolean {
  if (wizard.disabled) return false;
  const point = wizard.points[id];
  if (!point) return true;
  return !point.dismissed;
}

export function dismissPoint(id: string) {
  if (!wizard.points[id]) {
    wizard.points[id] = { seen: false, dismissed: false };
  }
  wizard.points[id].seen = true;
  wizard.points[id].dismissed = true;
  if (activeHint.id === id) activeHint.id = null;
  if (activeNudge.id === id) activeNudge.id = null;
  saveState();
}

export function showHint(id: string) {
  if (!shouldShowPoint(id)) return;
  activeNudge.id = null;
  activeHint.id = id;
  if (!wizard.points[id]) {
    wizard.points[id] = { seen: false, dismissed: false };
  }
  wizard.points[id].seen = true;
  saveState();
}

export function showNudge(id: string) {
  if (!shouldShowPoint(id)) return;
  activeNudge.id = id;
}

export function clearActive() {
  activeHint.id = null;
  activeNudge.id = null;
}

/**
 * Generic trigger for first-action points.
 * Components call this when a relevant user action occurs.
 * Shows nudge first, then auto-expands to hint after 2s.
 * Returns the timeout ID so callers can cancel if needed.
 */
export function triggerAction(pointId: string): ReturnType<typeof setTimeout> | undefined {
  if (!shouldShowPoint(pointId)) return;
  if (activeHint.id || activeNudge.id) return; // don't interrupt
  showNudge(pointId);
  return setTimeout(() => {
    if (activeNudge.id === pointId) {
      showHint(pointId);
    }
  }, 2000);
}

export function disableWizard() {
  wizard.disabled = true;
  clearActive();
  saveState();
}

export function enableWizard() {
  wizard.disabled = false;
  saveState();
}

export function resetWizard() {
  wizard.points = {};
  wizard.disabled = false;
  clearActive();
  saveState();
}
