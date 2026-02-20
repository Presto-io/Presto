<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { FileText } from 'lucide-svelte';
	import WizardOverlay from '$lib/components/wizard/WizardOverlay.svelte';
	import { fileRouter } from '$lib/stores/file-router.svelte';

	let { children } = $props();

	// --- Universal drag-drop ---
	let dragOver = $state(false);
	let dragCounter = 0;
	let confirmDialog: HTMLDialogElement;

	const markdownExts = ['.md', '.markdown', '.txt'];
	function isAcceptedFile(name: string): boolean {
		const lower = name.toLowerCase();
		return lower.endsWith('.zip') || markdownExts.some(ext => lower.endsWith(ext));
	}

	function hasDroppableFile(e: DragEvent): boolean {
		if (!e.dataTransfer?.types.includes('Files')) return false;
		// Don't intercept internal drags (file reassignment between templates)
		if (e.dataTransfer.types.includes('application/x-presto-files')) return false;
		const items = e.dataTransfer.items;
		for (let i = 0; i < items.length; i++) {
			if (items[i].kind === 'file') {
				const entry = items[i].webkitGetAsEntry?.();
				if (entry && isAcceptedFile(entry.name)) return true;
				// Fallback: check MIME type for ZIP
				if (items[i].type === 'application/zip' || items[i].type === 'application/x-zip-compressed') return true;
				// Fallback: check MIME type for text/markdown
				if (items[i].type === 'text/plain' || items[i].type === 'text/markdown') return true;
			}
		}
		return false;
	}

	function handleDragEnter(e: DragEvent) {
		dragCounter++;
		if (hasDroppableFile(e)) {
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

		const files = Array.from(e.dataTransfer.files).filter(f => isAcceptedFile(f.name));
		if (files.length === 0) return;
		e.preventDefault();
		e.stopPropagation();

		await fileRouter.processFiles(files, window.location.pathname);
	}

	// Sync confirm dialog with fileRouter state
	$effect(() => {
		if (fileRouter.confirmVisible) {
			confirmDialog?.showModal();
		} else {
			confirmDialog?.close();
		}
	});

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
				<FileText size={32} />
				<span>释放以导入文件</span>
			</div>
		</div>
	{/if}

	{#if fileRouter.toast}
		<div class="toast" class:toast-error={fileRouter.toast.type === 'error'}>
			{fileRouter.toast.message}
		</div>
	{/if}
</div>

<dialog bind:this={confirmDialog} class="confirm-dialog">
	<h3>打开文件</h3>
	<p>当前编辑器有未保存的内容，打开新文件将替换当前内容。</p>
	<div class="dialog-actions">
		<button class="dialog-btn primary" onclick={fileRouter.confirmAccept}>替换</button>
		<button class="dialog-btn" onclick={fileRouter.confirmCancel}>取消</button>
	</div>
</dialog>

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
	.confirm-dialog {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md, 8px);
		background: var(--color-surface);
		color: var(--color-text);
		padding: 24px;
		max-width: 400px;
		font-family: var(--font-ui);
	}
	.confirm-dialog::backdrop {
		background: rgba(0, 0, 0, 0.4);
	}
	.confirm-dialog h3 {
		margin: 0 0 8px;
		font-size: 16px;
		font-weight: 600;
	}
	.confirm-dialog p {
		margin: 0 0 20px;
		font-size: 13px;
		color: var(--color-muted);
		line-height: 1.5;
	}
	.dialog-actions {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
	}
	.dialog-btn {
		padding: 6px 14px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 12px;
		cursor: pointer;
		transition: opacity var(--transition);
	}
	.dialog-btn:hover { opacity: 0.85; }
	.dialog-btn.primary {
		background: var(--color-accent);
		color: var(--color-bg);
		border-color: var(--color-accent);
	}
</style>
