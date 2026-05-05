<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import { FileText, Maximize2, Minus, X } from 'lucide-svelte';
	import WizardOverlay from '$lib/components/wizard/WizardOverlay.svelte';
	import DownloadProgressBar from '$lib/components/DownloadProgressBar.svelte';
	import FirstLaunchBanner from '$lib/components/FirstLaunchBanner.svelte';
	import NotificationCenter from '$lib/components/NotificationCenter.svelte';
	import { fileRouter } from '$lib/stores/file-router.svelte';
	import { notificationStore } from '$lib/stores/notification.svelte';
	import { editor } from '$lib/stores/editor.svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';

	let { children } = $props();

	let isShowcase = $derived($page.url.pathname.startsWith('/showcase'));
	let showEmbeddedMenu = $state(false);
	let isWindowsDesktop = $state(false);

	type MenuAction =
		| 'new'
		| 'open'
		| 'save'
		| 'saveas'
		| 'export'
		| 'close-window'
		| 'quit'
		| 'undo'
		| 'redo'
		| 'cut'
		| 'copy'
		| 'paste'
		| 'selectall';

	const menuGroups: {
		label: string;
		items: { label: string; action?: MenuAction; href?: string; shortcut?: string; separator?: boolean }[];
	}[] = [
		{
			label: '文件',
			items: [
				{ label: '新建', action: 'new', shortcut: 'Ctrl+N' },
				{ label: '打开文件...', action: 'open', shortcut: 'Ctrl+O' },
				{ label: '保存', action: 'save', shortcut: 'Ctrl+S' },
				{ label: '另存为...', action: 'saveas', shortcut: 'Ctrl+Shift+S' },
				{ label: '导出 PDF...', action: 'export', shortcut: 'Ctrl+E' },
				{ separator: true, label: '' },
				{ label: '设置...', href: '/settings', shortcut: 'Ctrl+,' },
				{ separator: true, label: '' },
				{ label: '关闭窗口', action: 'close-window', shortcut: 'Ctrl+W' },
				{ label: '退出 Presto', action: 'quit', shortcut: 'Ctrl+Q' },
			],
		},
		{
			label: '编辑',
			items: [
				{ label: '撤销', action: 'undo', shortcut: 'Ctrl+Z' },
				{ label: '重做', action: 'redo', shortcut: 'Ctrl+Shift+Z' },
				{ separator: true, label: '' },
				{ label: '剪切', action: 'cut', shortcut: 'Ctrl+X' },
				{ label: '复制', action: 'copy', shortcut: 'Ctrl+C' },
				{ label: '粘贴', action: 'paste', shortcut: 'Ctrl+V' },
				{ separator: true, label: '' },
				{ label: '全选', action: 'selectall', shortcut: 'Ctrl+A' },
			],
		},
		{
			label: '模板',
			items: [{ label: '模板商店', href: '/store-templates', shortcut: 'Ctrl+Shift+T' }],
		},
		{
			label: '技能',
			items: [{ label: '技能商店', href: '/store-skills', shortcut: 'Ctrl+Shift+K' }],
		},
		{
			label: '帮助',
			items: [
				{ label: '文档', href: 'https://presto.io/docs' },
				{ label: '关于 Presto', href: 'about:presto' },
				{ label: '检查更新', href: 'update:presto' },
			],
		},
	];

	async function resolveEmbeddedMenuMode() {
		if (isShowcase) {
			showEmbeddedMenu = false;
			isWindowsDesktop = false;
			return;
		}

		if (!window.go?.main?.App?.GetPlatform) {
			showEmbeddedMenu = true;
			isWindowsDesktop = false;
			return;
		}

		try {
			const platform = await window.go.main.App.GetPlatform();
			isWindowsDesktop = platform === 'windows';
			showEmbeddedMenu = isWindowsDesktop;
		} catch {
			showEmbeddedMenu = true;
			isWindowsDesktop = false;
		}
	}

	function emitPageMenuAction(action: MenuAction) {
		const event = new CustomEvent<MenuAction>('presto:menu-action', {
			detail: action,
			cancelable: true,
		});
		window.dispatchEvent(event);
		return event.defaultPrevented;
	}

	function handleMenuItem(item: { action?: MenuAction; href?: string }) {
		if (item.action) {
			if (['undo', 'redo', 'cut', 'copy', 'paste', 'selectall'].includes(item.action)) {
				const command = item.action === 'selectall' ? 'selectAll' : item.action;
				document.execCommand(command);
				return;
			}
			const handled = emitPageMenuAction(item.action);
			if (!handled && (item.action === 'close-window' || item.action === 'quit')) {
				void window.go?.main?.App?.QuitApp?.();
				window.runtime?.Quit?.();
			}
			return;
		}

		if (!item.href) return;
		if (item.href === 'about:presto') {
			void window.go?.main?.App?.ShowAboutDialog?.();
			return;
		}
		if (item.href === 'update:presto') {
			void window.go?.main?.App?.CheckAndNotifyUpdate?.();
			return;
		}
		if (item.href.startsWith('http')) {
			if (window.runtime?.BrowserOpenURL) {
				window.runtime.BrowserOpenURL(item.href);
			} else {
				window.open(item.href, '_blank', 'noopener,noreferrer');
			}
			return;
		}
		goto(item.href);
	}

	function handleWindowClose() {
		const handled = emitPageMenuAction('close-window');
		if (!handled) {
			void window.go?.main?.App?.QuitApp?.();
			window.runtime?.Quit?.();
		}
	}

	$effect(() => {
		void resolveEmbeddedMenuMode();
	});

	// --- Universal drag-drop (window-level capture phase) ---
	let dragOver = $state(false);
	let dragCounter = 0;
	let confirmDialog = $state<HTMLDialogElement>();

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

	async function handleNativeItems(items: any[], source: 'drop' | 'open') {
		dragCounter = 0;
		dragOver = false;

		const documentDirs = new Map<string, string>();
		const filePaths = new Map<string, string>();
		const files: File[] = [];
		const zipResults: any[] = [];

		for (const item of items) {
			if (item.isZip && item.path) {
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
				if (item.path) filePaths.set(item.name, item.path);
				files.push(new File([item.content], item.name, { type: 'text/markdown' }));
			}
		}

		if (files.length === 0 && zipResults.length === 0) return;

		if (files.length !== 1 || zipResults.length > 0) {
			editor.currentFilePath = '';
		}

		const targetPath = source === 'open' && files.length === 1 && zipResults.length === 0
			? '/'
			: window.location.pathname;

		await fileRouter.processFiles(
			files,
			targetPath,
			documentDirs.size > 0 ? documentDirs : undefined,
			filePaths.size > 0 ? filePaths : undefined,
			zipResults.length > 0 ? zipResults : undefined,
		);
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

		// Flush pending notifications when window regains focus
		window.addEventListener('focus', notificationStore.flushPending);

		if (window.runtime?.EventsOn) {
			// App notification events from Go backend
			window.runtime.EventsOn('app:notification', (data: any) => {
				notificationStore.show(data.message, data.type || 'info');
			});
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

			// Edit menu event listeners (from Go custom edit menu on Windows)
			window.runtime.EventsOn('menu:undo', () => document.execCommand('undo'));
			window.runtime.EventsOn('menu:redo', () => document.execCommand('redo'));
			window.runtime.EventsOn('menu:cut', () => document.execCommand('cut'));
			window.runtime.EventsOn('menu:copy', () => document.execCommand('copy'));
			window.runtime.EventsOn('menu:paste', () => document.execCommand('paste'));
			window.runtime.EventsOn('menu:selectall', () => document.execCommand('selectAll'));

			const handleNativeEvent = async (...args: any[]) => {
				const items: any[] = Array.isArray(args[0]) ? args[0] : args;
				await handleNativeItems(items, 'open');
			};

			window.runtime.EventsOn('native-file-drop', async (...args: any[]) => {
				const items: any[] = Array.isArray(args[0]) ? args[0] : args;
				await handleNativeItems(items, 'drop');
			});
			window.runtime.EventsOn('native-file-open', handleNativeEvent);

			try {
				await (window as any).go.main.App.SetFileOpenReady();
			} catch (e) {
				console.error('[file-open] SetFileOpenReady failed:', e);
			}

			try {
				const startupFiles = await (window as any).go.main.App.GetStartupFiles();
				if (startupFiles?.length) {
					await handleNativeItems(startupFiles, 'open');
				}
			} catch (e) {
				console.error('[file-open] GetStartupFiles failed:', e);
			}
		}
	});

	onDestroy(() => {
		window.removeEventListener('dragenter', handleDragEnter, true);
		window.removeEventListener('dragover', handleDragOver, true);
		window.removeEventListener('dragleave', handleDragLeave, true);
		window.removeEventListener('drop', handleDrop, true);
		window.removeEventListener('focus', notificationStore.flushPending);
		window.runtime?.EventsOff?.('native-file-drop');
		window.runtime?.EventsOff?.('native-file-open');
	});
</script>

<div class="app" class:with-embedded-menu={showEmbeddedMenu}>
	{#if showEmbeddedMenu}
		<nav class="embedded-menu" aria-label="应用菜单" style="--wails-draggable:drag">
			<div class="embedded-menu-left">
				<span class="embedded-menu-title">Presto</span>
				{#each menuGroups as group}
					<div class="menu-group">
						<button class="menu-trigger" type="button">{group.label}</button>
						<div class="menu-popover">
							{#each group.items as item}
								{#if item.separator}
									<div class="menu-separator"></div>
								{:else}
									<button class="menu-item" type="button" onclick={() => handleMenuItem(item)}>
										<span>{item.label}</span>
										{#if item.shortcut}<kbd>{item.shortcut}</kbd>{/if}
									</button>
								{/if}
							{/each}
						</div>
					</div>
				{/each}
			</div>
			{#if isWindowsDesktop}
				<div class="window-controls">
					<button type="button" class="window-control" aria-label="最小化" title="最小化" onclick={() => window.runtime?.WindowMinimise?.()}>
						<Minus size={13} />
					</button>
					<button type="button" class="window-control" aria-label="最大化" title="最大化" onclick={() => window.runtime?.WindowToggleMaximise?.()}>
						<Maximize2 size={12} />
					</button>
					<button type="button" class="window-control close" aria-label="关闭" title="关闭" onclick={handleWindowClose}>
						<X size={14} />
					</button>
				</div>
			{/if}
		</nav>
	{/if}
	<main id="main-content">
		{@render children()}
	</main>
	{#if !isShowcase}
		<div class="bottom-status">
			<FirstLaunchBanner />
			<DownloadProgressBar />
		</div>
	<WizardOverlay />

	{#if dragOver}
		<div class="drop-overlay">
			<div class="drop-content">
				<FileText size={32} />
				<span>释放以导入文件</span>
			</div>
		</div>
	{/if}

	<NotificationCenter />
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
	.embedded-menu {
		display: flex;
		align-items: stretch;
		justify-content: space-between;
		height: 34px;
		background: var(--color-bg);
		border-bottom: 1px solid var(--color-border);
		color: var(--color-text);
		flex-shrink: 0;
		position: relative;
		z-index: 9500;
	}
	.embedded-menu-left {
		display: flex;
		align-items: center;
		min-width: 0;
	}
	.embedded-menu-title {
		display: inline-flex;
		align-items: center;
		height: 100%;
		padding: 0 13px;
		color: var(--color-text-bright);
		font-size: 12px;
		font-weight: 600;
	}
	.menu-group {
		position: relative;
		height: 100%;
		--wails-draggable: no-drag;
	}
	.menu-trigger {
		height: 100%;
		padding: 0 10px;
		background: transparent;
		color: var(--color-text);
		font-size: 12px;
		border-radius: 0;
	}
	.menu-trigger:hover,
	.menu-group:focus-within .menu-trigger {
		background: var(--color-hover-overlay);
		color: var(--color-text-bright);
	}
	.menu-popover {
		position: absolute;
		left: 0;
		top: 100%;
		min-width: 188px;
		padding: 5px;
		background: var(--color-bg-elevated);
		border: 1px solid var(--color-border);
		box-shadow: var(--shadow-md);
		opacity: 0;
		transform: translateY(-4px);
		pointer-events: none;
		transition: opacity var(--transition), transform var(--transition);
	}
	.menu-group:hover .menu-popover,
	.menu-group:focus-within .menu-popover {
		opacity: 1;
		transform: translateY(0);
		pointer-events: auto;
	}
	.menu-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 22px;
		width: 100%;
		min-height: 28px;
		padding: 0 9px;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text);
		font-size: 12px;
		text-align: left;
		white-space: nowrap;
	}
	.menu-item:hover,
	.menu-item:focus-visible {
		background: var(--color-surface-hover);
		color: var(--color-text-bright);
	}
	.menu-item kbd {
		color: var(--color-muted);
		font-family: var(--font-ui);
		font-size: 11px;
		font-weight: 400;
	}
	.menu-separator {
		height: 1px;
		margin: 5px 4px;
		background: var(--color-border);
	}
	.window-controls {
		display: flex;
		align-items: stretch;
		--wails-draggable: no-drag;
	}
	.window-control {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 46px;
		height: 100%;
		border-radius: 0;
		background: transparent;
		color: var(--color-text);
	}
	.window-control:hover {
		background: var(--color-hover-overlay);
		color: var(--color-text-bright);
	}
	.window-control.close:hover {
		background: var(--color-danger);
		color: var(--color-on-danger);
	}
	.app.with-embedded-menu :global(.toolbar) {
		padding-top: var(--space-sm);
	}
	.bottom-status {
		position: fixed;
		bottom: 0;
		left: 0;
		right: 0;
		z-index: 9000;
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
