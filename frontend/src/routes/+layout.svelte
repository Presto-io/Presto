<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { FileText } from 'lucide-svelte';
	import WizardOverlay from '$lib/components/wizard/WizardOverlay.svelte';
	import FirstLaunchBanner from '$lib/components/FirstLaunchBanner.svelte';
	import { fileRouter } from '$lib/stores/file-router.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	let { children } = $props();

	let isShowcase = $derived($page.url.pathname.startsWith('/showcase'));

	// --- Universal drag-drop (window-level capture phase) ---
	let dragOver = $state(false);
	let dragCounter = 0;
	let confirmDialog: HTMLDialogElement;

	const ACCEPTED_EXTS = ['.md', '.markdown', '.txt', '.zip'];

	/** Check if this is an external file drag (not an internal presto drag). */
	function isExternalFileDrag(e: DragEvent): boolean {
		if (!e.dataTransfer?.types.includes('Files')) return false;
		// Don't intercept internal drags (file reassignment between templates)
		if (e.dataTransfer.types.includes('application/x-presto-files')) return false;
		return true;
	}

	function handleDragEnter(e: DragEvent) {
		if (!isExternalFileDrag(e)) return;
		dragCounter++;
		e.preventDefault();
		dragOver = true;
	}

	function handleDragOver(e: DragEvent) {
		if (!isExternalFileDrag(e)) return;
		e.preventDefault();
		e.stopPropagation();
	}

	function handleDragLeave(e: DragEvent) {
		if (!isExternalFileDrag(e)) return;
		dragCounter--;
		if (dragCounter <= 0) {
			dragCounter = 0;
			dragOver = false;
		}
	}

	async function handleDrop(e: DragEvent) {
		dragCounter = 0;
		dragOver = false;
		if (!isExternalFileDrag(e)) return;
		// Prevent default BEFORE filtering — stops browser from opening the file
		e.preventDefault();
		e.stopPropagation();

		// Desktop: Wails native handler provides paths with dir info
		// Skip browser processing to avoid double handling
		if (window.runtime?.EventsOn) return;

		// Browser mode: no dir info available
		const files = Array.from(e.dataTransfer!.files).filter(f =>
			ACCEPTED_EXTS.some(ext => f.name.toLowerCase().endsWith(ext))
		);
		if (files.length === 0) return;

		// Route based on the actual current page
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

	onMount(async () => {
		// Skip all event registration in showcase mode
		if (isShowcase) return;

		// Register drag handlers on window in capture phase
		// — runs before any child component (including CodeMirror) can intercept
		window.addEventListener('dragenter', handleDragEnter, true);
		window.addEventListener('dragover', handleDragOver, true);
		window.addEventListener('dragleave', handleDragLeave, true);
		window.addEventListener('drop', handleDrop, true);

		if (window.runtime?.EventsOn) {
			// URL scheme: presto://install/{name} → navigate to template detail page
			// Hot start: event pushed from Go via SingleInstanceLock
			window.runtime.EventsOn('url-scheme-open-template', (name: string) => {
				console.log('[url-scheme] hot start event received:', name);
				goto(`/store-templates?template=${encodeURIComponent(name)}`);
			});

			// Cold start: pull pending URL from Go (event timing unreliable at startup)
			try {
				console.log('[url-scheme] checking for startup URL...');
				const pendingURL = await (window as any).go.main.App.GetStartupURL();
				console.log('[url-scheme] startup URL:', pendingURL);
				if (pendingURL) {
					const match = pendingURL.match(/^presto:\/\/install\/(.+)/);
					if (match) {
						console.log('[url-scheme] navigating to template:', match[1]);
						goto(`/store-templates?template=${encodeURIComponent(match[1])}`);
					}
				}
			} catch (e) {
				console.error('[url-scheme] GetStartupURL failed:', e);
			}

			window.runtime.EventsOn('native-file-drop', async (...args: any[]) => {
				const items: any[] = Array.isArray(args[0]) ? args[0] : args;
				dragCounter = 0;
				dragOver = false;

				const documentDirs = new Map<string, string>();
				const files: File[] = [];
				const zipResults: any[] = [];

				for (const item of items) {
					if (item.isZip && item.path) {
						// Call Wails binding directly — bypasses WebView HTTP layer
						try {
							const result = await (window as any).go.main.App.ImportBatchZip(item.path);
							zipResults.push(result);
						} catch (err) {
							console.error('ImportBatchZip failed:', err);
							fileRouter.showToast(
								err instanceof Error ? err.message : 'ZIP 导入失败',
								'error',
							);
						}
					} else {
						documentDirs.set(item.name, item.dir);
						files.push(new File([item.content], item.name, { type: 'text/markdown' }));
					}
				}

				if (files.length > 0 || zipResults.length > 0) {
					fileRouter.processFiles(
						files,
						window.location.pathname,
						documentDirs.size > 0 ? documentDirs : undefined,
						zipResults.length > 0 ? zipResults : undefined,
					);
				}
			});
		}

		return () => {
			window.removeEventListener('dragenter', handleDragEnter, true);
			window.removeEventListener('dragover', handleDragOver, true);
			window.removeEventListener('dragleave', handleDragLeave, true);
			window.removeEventListener('drop', handleDrop, true);
		};
	});
</script>

<div class="app">
	<FirstLaunchBanner />
	<main id="main-content">
		{@render children()}
	</main>
	{#if !isShowcase}
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
	{/if}
</div>

{#if !isShowcase}
<dialog bind:this={confirmDialog} class="confirm-dialog">
	<h3>打开文件</h3>
	<p>当前编辑器有未保存的内容，打开新文件将替换当前内容。</p>
	<div class="dialog-actions">
		<button class="dialog-btn primary" onclick={fileRouter.confirmAccept}>替换</button>
		<button class="dialog-btn" onclick={fileRouter.confirmCancel}>取消</button>
	</div>
</dialog>
{/if}

<style>
	.app {
		--wails-drop-target: drop;
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
		background: var(--color-overlay-bg);
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
		background: var(--color-backdrop);
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
