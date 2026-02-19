<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import WizardOverlay from '$lib/components/wizard/WizardOverlay.svelte';

	let { children } = $props();

	onMount(() => {
		if (!window.runtime?.EventsOn) return;
		// Forward edit menu events to native document commands
		// (Wails custom menus intercept Cmd+C/V/X/Z, so we re-dispatch via JS)
		window.runtime.EventsOn('menu:undo', () => document.execCommand('undo'));
		window.runtime.EventsOn('menu:redo', () => document.execCommand('redo'));
		window.runtime.EventsOn('menu:cut', () => document.execCommand('cut'));
		window.runtime.EventsOn('menu:copy', () => document.execCommand('copy'));
		window.runtime.EventsOn('menu:paste', () => document.execCommand('paste'));
		window.runtime.EventsOn('menu:selectAll', () => document.execCommand('selectAll'));
	});
</script>

<div class="app">
	<main id="main-content">
		{@render children()}
	</main>
	<WizardOverlay />
</div>

<style>
	.app {
		display: flex;
		flex-direction: column;
		height: 100vh;
		overflow: hidden;
	}
	main {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
</style>
