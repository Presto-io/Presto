<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { Package } from 'lucide-svelte';
	import WizardOverlay from '$lib/components/wizard/WizardOverlay.svelte';
	import { importTemplateZip } from '$lib/api/client';

	let { children } = $props();

	// --- Drag-drop ZIP import ---
	let dragOver = $state(false);
	let dragCounter = 0;
	let toast = $state<{ message: string; type: 'success' | 'error' } | null>(null);
	let toastTimer: ReturnType<typeof setTimeout>;

	function showToast(message: string, type: 'success' | 'error') {
		clearTimeout(toastTimer);
		toast = { message, type };
		toastTimer = setTimeout(() => { toast = null; }, 2500);
	}

	function hasZipFile(e: DragEvent): boolean {
		if (!e.dataTransfer?.types.includes('Files')) return false;
		const items = e.dataTransfer.items;
		for (let i = 0; i < items.length; i++) {
			if (items[i].kind === 'file') {
				const entry = items[i].webkitGetAsEntry?.();
				if (entry && entry.name.toLowerCase().endsWith('.zip')) return true;
				// Fallback: check type
				if (items[i].type === 'application/zip' || items[i].type === 'application/x-zip-compressed') return true;
			}
		}
		return false;
	}

	function handleDragEnter(e: DragEvent) {
		dragCounter++;
		if (hasZipFile(e)) {
			e.preventDefault();
			dragOver = true;
		}
	}

	function handleDragOver(e: DragEvent) {
		if (dragOver) e.preventDefault();
	}

	function handleDragLeave() {
		dragCounter--;
		if (dragCounter <= 0) {
			dragCounter = 0;
			dragOver = false;
		}
	}

	async function handleDrop(e: DragEvent) {
		dragCounter = 0;
		dragOver = false;
		if (!e.dataTransfer?.files) return;

		const file = Array.from(e.dataTransfer.files).find(f =>
			f.name.toLowerCase().endsWith('.zip')
		);
		if (!file) return;
		e.preventDefault();

		try {
			const tpls = await importTemplateZip(file);
			const names = tpls.map(t => t.displayName || t.name).join('、');
			showToast(`模板 "${names}" 导入成功`, 'success');
			window.dispatchEvent(new CustomEvent('templates-changed'));
		} catch (err: any) {
			if (err.conflicts) {
				// Conflict: auto-rename and retry
				try {
					const tpls = await importTemplateZip(file, 'rename');
					const names = tpls.map(t => t.displayName || t.name).join('、');
					showToast(`模板 "${names}" 导入成功（已自动重命名）`, 'success');
					window.dispatchEvent(new CustomEvent('templates-changed'));
				} catch (retryErr) {
					showToast(retryErr instanceof Error ? retryErr.message : String(retryErr), 'error');
				}
			} else {
				showToast(err instanceof Error ? err.message : String(err), 'error');
			}
		}
	}

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

<div
	class="app"
	ondragenter={handleDragEnter}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
>
	<main id="main-content">
		{@render children()}
	</main>
	<WizardOverlay />

	{#if dragOver}
		<div class="drop-overlay">
			<div class="drop-content">
				<Package size={32} />
				<span>释放以导入模板</span>
			</div>
		</div>
	{/if}

	{#if toast}
		<div class="toast" class:toast-error={toast.type === 'error'}>
			{toast.message}
		</div>
	{/if}
</div>

<style>
	.app {
		display: flex;
		flex-direction: column;
		height: 100vh;
		overflow: hidden;
		position: relative;
	}
	main {
		flex: 1;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
	.drop-overlay {
		position: fixed;
		inset: 0;
		z-index: 9000;
		background: rgba(26, 27, 38, 0.85);
		display: flex;
		align-items: center;
		justify-content: center;
		pointer-events: none;
	}
	.drop-content {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-md);
		padding: var(--space-2xl);
		border: 2px dashed var(--color-accent);
		border-radius: var(--radius-lg);
		color: var(--color-accent);
		font-size: 1rem;
		font-weight: 500;
	}
	.toast {
		position: fixed;
		bottom: var(--space-xl);
		left: 50%;
		transform: translateX(-50%);
		z-index: 9001;
		padding: var(--space-sm) var(--space-lg);
		background: var(--color-success);
		color: var(--color-bg);
		border-radius: var(--radius-md);
		font-size: 0.8125rem;
		font-weight: 500;
		pointer-events: none;
		animation: toast-in 200ms ease-out;
	}
	.toast-error {
		background: var(--color-danger);
	}
	@keyframes toast-in {
		from { opacity: 0; transform: translateX(-50%) translateY(8px); }
		to { opacity: 1; transform: translateX(-50%) translateY(0); }
	}
</style>
