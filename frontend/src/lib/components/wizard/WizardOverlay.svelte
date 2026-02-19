<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/state';
  import WizardHint from './WizardHint.svelte';
  import WizardNudge from './WizardNudge.svelte';
  import { WIZARD_POINTS, type WizardPointDef } from './wizard-definitions';
  import {
    wizard, activeHint, activeNudge,
    shouldShowPoint, showNudge, showHint, clearActive,
  } from '$lib/stores/wizard.svelte';

  // --- Timer tracking ---
  let mountTimers: ReturnType<typeof setTimeout>[] = [];
  let idleTimerMap = new Map<string, ReturnType<typeof setTimeout>>();
  let nudgeExpandTimers = new Map<string, ReturnType<typeof setTimeout>>();
  let lastActivity = Date.now();
  let currentPath = $state('');

  function clearAllTimers() {
    for (const t of mountTimers) clearTimeout(t);
    mountTimers = [];
    for (const t of idleTimerMap.values()) clearTimeout(t);
    idleTimerMap.clear();
    for (const t of nudgeExpandTimers.values()) clearTimeout(t);
    nudgeExpandTimers.clear();
  }

  // --- Idle detection ---
  function resetIdle() {
    lastActivity = Date.now();
    // If a nudge is showing due to idle, clear it on activity
    if (activeNudge.id) {
      const def = WIZARD_POINTS.find(p => p.id === activeNudge.id);
      if (def?.trigger === 'on-idle') {
        activeNudge.id = null;
        // Cancel the nudge->hint expand timer
        const expandTimer = nudgeExpandTimers.get(def.id);
        if (expandTimer) {
          clearTimeout(expandTimer);
          nudgeExpandTimers.delete(def.id);
        }
      }
    }
    // Restart idle timers
    for (const [id, timer] of idleTimerMap) {
      clearTimeout(timer);
      idleTimerMap.delete(id);
    }
    scheduleIdlePoints();
  }

  function scheduleIdlePoints() {
    const path = currentPath;
    const idlePoints = WIZARD_POINTS.filter(
      p => p.trigger === 'on-idle' && p.route === path && shouldShowPoint(p.id)
    );
    for (const point of idlePoints) {
      if (idleTimerMap.has(point.id)) continue;
      const timer = setTimeout(() => {
        idleTimerMap.delete(point.id);
        if (!document.querySelector(point.anchorSelector)) return;
        if (activeHint.id || activeNudge.id) return;
        showNudge(point.id);
        // Auto-expand nudge → hint after 3s
        const expandTimer = setTimeout(() => {
          nudgeExpandTimers.delete(point.id);
          if (activeNudge.id === point.id) {
            showHint(point.id);
          }
        }, 3000);
        nudgeExpandTimers.set(point.id, expandTimer);
      }, (point.idleSeconds ?? 30) * 1000);
      idleTimerMap.set(point.id, timer);
    }
  }

  // --- Mount-triggered points ---
  function scheduleMountPoints() {
    const path = currentPath;
    const points = WIZARD_POINTS
      .filter(p => p.trigger === 'on-mount' && p.route === path && shouldShowPoint(p.id))
      .sort((a, b) => (a.triggerDelay ?? 0) - (b.triggerDelay ?? 0));

    for (const point of points) {
      const timer = setTimeout(() => {
        if (!document.querySelector(point.anchorSelector)) return;
        if (activeHint.id) return;
        showNudge(point.id);
        const expandTimer = setTimeout(() => {
          nudgeExpandTimers.delete(point.id);
          if (activeNudge.id === point.id) {
            showHint(point.id);
          }
        }, 2500);
        nudgeExpandTimers.set(point.id, expandTimer);
      }, point.triggerDelay ?? 1000);
      mountTimers.push(timer);
    }
  }

  // --- Navigate-triggered points ---
  function scheduleNavigatePoints() {
    const path = currentPath;
    const points = WIZARD_POINTS.filter(
      p => p.trigger === 'on-navigate' && p.route === path && shouldShowPoint(p.id)
    );
    for (const point of points) {
      const timer = setTimeout(() => {
        if (!document.querySelector(point.anchorSelector)) return;
        if (activeHint.id || activeNudge.id) return;
        showNudge(point.id);
        const expandTimer = setTimeout(() => {
          nudgeExpandTimers.delete(point.id);
          if (activeNudge.id === point.id) {
            showHint(point.id);
          }
        }, 2000);
        nudgeExpandTimers.set(point.id, expandTimer);
      }, point.triggerDelay ?? 1500);
      mountTimers.push(timer);
    }
  }

  // --- Route change detection ---
  $effect(() => {
    const path = page.url?.pathname ?? '/';
    if (path !== currentPath) {
      currentPath = path;
      clearActive();
      clearAllTimers();
      // Give DOM time to render new route
      setTimeout(() => {
        scheduleMountPoints();
        scheduleNavigatePoints();
        scheduleIdlePoints();
      }, 300);
    }
  });

  // --- Resolve active definitions ---
  let activeHintDef = $derived(
    activeHint.id ? WIZARD_POINTS.find(p => p.id === activeHint.id) : undefined
  );
  let activeNudgeDef = $derived(
    activeNudge.id ? WIZARD_POINTS.find(p => p.id === activeNudge.id) : undefined
  );

  function handleNudgeActivate() {
    if (activeNudge.id) {
      showHint(activeNudge.id);
    }
  }

  onMount(() => {
    if (wizard.disabled) return;

    const events = ['mousemove', 'mousedown', 'keydown', 'touchstart', 'scroll'] as const;
    for (const event of events) {
      document.addEventListener(event, resetIdle, { passive: true });
    }

    return () => {
      for (const event of events) {
        document.removeEventListener(event, resetIdle);
      }
      clearAllTimers();
    };
  });
</script>

{#if !wizard.disabled}
  {#if activeNudgeDef && !activeHintDef}
    <WizardNudge
      anchorSelector={activeNudgeDef.anchorSelector}
      position={activeNudgeDef.position}
      onactivate={handleNudgeActivate}
    />
  {/if}

  {#if activeHintDef}
    <WizardHint
      point={activeHintDef}
      ondismiss={() => clearActive()}
    />
  {/if}
{/if}
